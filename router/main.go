package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

// FrontendEndpoint represents a discovered frontend instance
type FrontendEndpoint struct {
	IP        string
	URL       string
	IsHealthy bool
	LastCheck time.Time
}

// RouterManager manages frontend discovery and load balancing
type RouterManager struct {
	endpoints      []*FrontendEndpoint
	mu             sync.RWMutex
	currentIndex   int
	serviceName    string
	servicePort    string
	healthPath     string
	updateInterval time.Duration
	healthTimeout  time.Duration
}

// NewRouterManager creates a new router manager
func NewRouterManager(serviceName, servicePort, healthPath string, updateInterval time.Duration) *RouterManager {
	return &RouterManager{
		endpoints:      make([]*FrontendEndpoint, 0),
		serviceName:    serviceName,
		servicePort:    servicePort,
		healthPath:     healthPath,
		updateInterval: updateInterval,
		healthTimeout:  3 * time.Second,
		currentIndex:   0,
	}
}

// Start begins the discovery and health checking routines
func (rm *RouterManager) Start(ctx context.Context) {
	log.Printf("🚀 Starting Router Manager for service: %s", rm.serviceName)

	// Initial discovery
	rm.discoverFrontends()

	// Start discovery goroutine (updates every 10 seconds)
	go rm.discoveryLoop(ctx)

	// Start health check goroutine
	go rm.healthCheckLoop(ctx)
}

// discoveryLoop continuously discovers frontend containers via DNS
func (rm *RouterManager) discoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(rm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Discovery loop stopped")
			return
		case <-ticker.C:
			rm.discoverFrontends()
		}
	}
}

// discoverFrontends uses Docker Swarm DNS to discover frontend containers
func (rm *RouterManager) discoverFrontends() {
	// Docker Swarm DNS: resolve using network alias "frontend"
	// This will return all IPs of containers with that alias
	dnsName := rm.serviceName

	log.Printf("🔍 Discovering frontends via DNS: %s", dnsName)

	// Perform DNS lookup
	ips, err := net.LookupIP(dnsName)
	if err != nil {
		log.Printf("❌ DNS lookup failed for %s: %v", dnsName, err)
		return
	}

	if len(ips) == 0 {
		log.Printf("⚠️  No IPs found for %s", dnsName)
		return
	}

	log.Printf("📡 Discovered %d frontend IPs", len(ips))

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Create a map of discovered IPs
	discoveredIPs := make(map[string]bool)
	for _, ip := range ips {
		ipStr := ip.String()
		discoveredIPs[ipStr] = true

		// Check if this IP already exists in our endpoints
		found := false
		for _, endpoint := range rm.endpoints {
			if endpoint.IP == ipStr {
				found = true
				break
			}
		}

		// Add new endpoint if not exists
		if !found {
			targetURL := fmt.Sprintf("http://%s:%s", ipStr, rm.servicePort)
			endpoint := &FrontendEndpoint{
				IP:        ipStr,
				URL:       targetURL,
				IsHealthy: false, // Will be checked by health loop
				LastCheck: time.Now(),
			}
			rm.endpoints = append(rm.endpoints, endpoint)
			log.Printf("✅ New frontend discovered: %s", targetURL)
		}
	}

	// Remove endpoints that are no longer in DNS
	newEndpoints := make([]*FrontendEndpoint, 0)
	for _, endpoint := range rm.endpoints {
		if discoveredIPs[endpoint.IP] {
			newEndpoints = append(newEndpoints, endpoint)
		} else {
			log.Printf("🗑️  Removing frontend (no longer in DNS): %s", endpoint.URL)
		}
	}
	rm.endpoints = newEndpoints
}

// healthCheckLoop continuously checks health of all endpoints
func (rm *RouterManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Health check loop stopped")
			return
		case <-ticker.C:
			rm.checkAllEndpoints()
		}
	}
}

// checkAllEndpoints performs health checks on all discovered endpoints
func (rm *RouterManager) checkAllEndpoints() {
	rm.mu.RLock()
	endpoints := make([]*FrontendEndpoint, len(rm.endpoints))
	copy(endpoints, rm.endpoints)
	rm.mu.RUnlock()

	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep *FrontendEndpoint) {
			defer wg.Done()
			rm.checkEndpointHealth(ep)
		}(endpoint)
	}
	wg.Wait()

	// Log health status
	rm.mu.RLock()
	healthyCount := 0
	for _, ep := range rm.endpoints {
		if ep.IsHealthy {
			healthyCount++
		}
	}
	rm.mu.RUnlock()

	log.Printf("🏥 Health Check: %d/%d frontends healthy", healthyCount, len(endpoints))
}

// checkEndpointHealth checks if a single endpoint is healthy
func (rm *RouterManager) checkEndpointHealth(endpoint *FrontendEndpoint) {
	healthURL := endpoint.URL + rm.healthPath

	client := &http.Client{
		Timeout: rm.healthTimeout,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		if endpoint.IsHealthy {
			log.Printf("❌ Frontend unhealthy: %s", endpoint.URL)
		}
		endpoint.IsHealthy = false
		endpoint.LastCheck = time.Now()
		return
	}
	defer resp.Body.Close()

	wasUnhealthy := !endpoint.IsHealthy
	endpoint.IsHealthy = resp.StatusCode == http.StatusOK
	endpoint.LastCheck = time.Now()

	if wasUnhealthy && endpoint.IsHealthy {
		log.Printf("✅ Frontend recovered: %s", endpoint.URL)
	}
}

// GetNextHealthyEndpoint returns the next healthy endpoint using Round Robin
func (rm *RouterManager) GetNextHealthyEndpoint() *FrontendEndpoint {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if len(rm.endpoints) == 0 {
		return nil
	}

	// Find healthy endpoints
	healthyEndpoints := make([]*FrontendEndpoint, 0)
	for _, ep := range rm.endpoints {
		if ep.IsHealthy {
			healthyEndpoints = append(healthyEndpoints, ep)
		}
	}

	if len(healthyEndpoints) == 0 {
		return nil
	}

	// Round Robin selection
	selectedEndpoint := healthyEndpoints[rm.currentIndex%len(healthyEndpoints)]
	rm.currentIndex++

	return selectedEndpoint
}

// GetStatus returns the current status of all endpoints
func (rm *RouterManager) GetStatus() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	endpoints := make([]map[string]interface{}, 0)
	healthyCount := 0

	for _, ep := range rm.endpoints {
		if ep.IsHealthy {
			healthyCount++
		}
		endpoints = append(endpoints, map[string]interface{}{
			"ip":         ep.IP,
			"url":        ep.URL,
			"is_healthy": ep.IsHealthy,
			"last_check": ep.LastCheck.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"service":      rm.serviceName,
		"total":        len(rm.endpoints),
		"healthy":      healthyCount,
		"endpoints":    endpoints,
		"last_updated": time.Now().Format(time.RFC3339),
	}
}

// ProxyHandler handles incoming requests and forwards to healthy frontends
func (rm *RouterManager) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	endpoint := rm.GetNextHealthyEndpoint()

	if endpoint == nil {
		log.Printf("❌ No healthy frontends available")
		http.Error(w, "Service Unavailable - No healthy frontends", http.StatusServiceUnavailable)
		return
	}

	// Parse target URL
	targetURL, err := url.Parse(endpoint.URL)
	if err != nil {
		log.Printf("❌ Failed to parse URL %s: %v", endpoint.URL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify request
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = targetURL.Host

	log.Printf("🔀 Proxying request to: %s", endpoint.URL)

	// Serve the request
	proxy.ServeHTTP(w, r)
}

func main() {
	log.Println("============================================")
	log.Println("  TalentNest Router - Docker Swarm")
	log.Println("============================================")

	// Configuration
	serviceName := getEnv("SERVICE_NAME", "frontend")
	servicePort := getEnv("SERVICE_PORT", "5173")
	healthPath := getEnv("HEALTH_PATH", "/")
	routerPort := getEnv("ROUTER_PORT", "8080")
	updateInterval := 10 * time.Second // Update every 10 seconds

	log.Printf("📋 Configuration:")
	log.Printf("  - Service Name: %s", serviceName)
	log.Printf("  - Service Port: %s", servicePort)
	log.Printf("  - Health Path: %s", healthPath)
	log.Printf("  - Router Port: %s", routerPort)
	log.Printf("  - Update Interval: %v", updateInterval)

	// Create router manager
	manager := NewRouterManager(serviceName, servicePort, healthPath, updateInterval)

	// Start background routines
	ctx := context.Background()
	manager.Start(ctx)

	// Wait for initial discovery
	time.Sleep(2 * time.Second)

	// Setup HTTP handlers
	http.HandleFunc("/", manager.ProxyHandler)

	// Status endpoint
	http.HandleFunc("/router/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := manager.GetStatus()

		// Simple JSON encoding
		fmt.Fprintf(w, `{
			"service": "%s",
			"total": %d,
			"healthy": %d,
			"last_updated": "%s"
		}`,
			status["service"],
			status["total"],
			status["healthy"],
			status["last_updated"])
	})

	// Health endpoint for router itself
	http.HandleFunc("/router/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Start HTTP server
	addr := ":" + routerPort
	log.Printf("🌐 Router listening on %s", addr)
	log.Printf("📍 Access: http://localhost:%s", routerPort)
	log.Printf("📊 Status: http://localhost:%s/router/status", routerPort)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
