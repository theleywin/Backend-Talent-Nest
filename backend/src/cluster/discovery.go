package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DiscoverNodes usa DNS lookup para descubrir todos los nodos del servicio backend
// Si el DNS falla, escanea el rango de IPs del Swarm network
func (cs *ClusterState) DiscoverNodes() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Obtener el hostname actual del contenedor
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("error getting hostname: %v", err)
	}

	// Obtener la IP del nodo actual
	currentIP, err := getCurrentIP()
	if err != nil {
		return fmt.Errorf("error getting current IP: %v", err)
	}

	log.Printf("Current node hostname: %s, IP: %s", hostname, currentIP)

	// Realizar DNS lookup del servicio backend
	// Docker Swarm crea un registro DNS para el alias de red
	ips, err := net.LookupIP(cs.ServiceName)

	var discoveredIPs []string

	if err != nil {
		log.Printf("DNS lookup failed for service %s: %v", cs.ServiceName, err)
		log.Println("Falling back to IP range scanning...")

		// Fallback: escanear el rango de IPs del Swarm network
		discoveredIPs, err = cs.scanNetworkRange(currentIP)
		if err != nil {
			return fmt.Errorf("both DNS lookup and IP scanning failed: %v", err)
		}
		log.Printf("Discovered %d nodes via IP scanning", len(discoveredIPs))
	} else {
		// Convertir net.IP a strings
		for _, ip := range ips {
			discoveredIPs = append(discoveredIPs, ip.String())
		}
		log.Printf("Discovered %d nodes via DNS", len(discoveredIPs))
	}

	// Marcar todos los nodos existentes como no vistos
	for _, node := range cs.Nodes {
		node.IsHealthy = false
	}

	// Actualizar o crear nodos descubiertos
	for _, ipStr := range discoveredIPs {
		// Generar ID basado en la IP (último octeto como base)
		nodeID := generateNodeIDFromIP(ipStr)

		if existingNode, exists := cs.Nodes[nodeID]; exists {
			// Actualizar nodo existente
			existingNode.LastSeen = time.Now()
			existingNode.IsHealthy = true
			log.Printf("Updated existing node: ID=%d, IP=%s", nodeID, ipStr)
		} else {
			// Crear nuevo nodo
			newNode := &Node{
				ID:        nodeID,
				Address:   fmt.Sprintf("http://%s:3000", ipStr),
				Role:      Follower,
				LastSeen:  time.Now(),
				IsHealthy: true,
			}
			cs.Nodes[nodeID] = newNode
			log.Printf("Discovered new node: ID=%d, IP=%s", nodeID, ipStr)
		}

		// Si esta IP es la del nodo actual, establecer el ID actual
		if ipStr == currentIP {
			cs.CurrentNodeID = nodeID
			log.Printf("Current node ID set to: %d", nodeID)
		}
	}

	// Eliminar nodos que no se han visto recientemente (más de 30 segundos)
	cs.cleanupStaleNodes()

	return nil
}

// generateNodeIDFromIP genera un ID único basado en la dirección IP
func generateNodeIDFromIP(ip string) int {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		// Si no es IPv4, usar hash del string
		hash := 0
		for _, ch := range ip {
			hash = hash*31 + int(ch)
		}
		return hash % 10000
	}

	// Usar los últimos dos octetos para crear un ID único
	thirdOctet, _ := strconv.Atoi(parts[2])
	fourthOctet, _ := strconv.Atoi(parts[3])
	return thirdOctet*256 + fourthOctet
}

// getCurrentIP obtiene la IP del contenedor actual
func getCurrentIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid IP address found")
}

// cleanupStaleNodes elimina nodos que no han sido vistos recientemente
func (cs *ClusterState) cleanupStaleNodes() {
	cutoff := time.Now().Add(-30 * time.Second)
	for id, node := range cs.Nodes {
		if node.LastSeen.Before(cutoff) {
			log.Printf("Removing stale node: ID=%d", id)
			delete(cs.Nodes, id)
		}
	}
}

// GetAllNodes retorna una lista ordenada de todos los nodos por ID
func (cs *ClusterState) GetAllNodes() []*Node {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	nodes := make([]*Node, 0, len(cs.Nodes))
	for _, node := range cs.Nodes {
		if node.IsHealthy {
			nodes = append(nodes, node)
		}
	}

	// Ordenar por ID (mayor a menor)
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].ID < nodes[j].ID {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}

	return nodes
}

// scanNetworkRange escanea el rango de IPs especificado en las variables de entorno
// para encontrar nodos healthy del cluster
func (cs *ClusterState) scanNetworkRange(currentIP string) ([]string, error) {
	// Obtener el rango de red de las variables de entorno
	networkSubnet := os.Getenv("SWARM_NETWORK_SUBNET")
	if networkSubnet == "" {
		// Si no está definido, intentar inferir del IP actual
		networkSubnet = inferSubnetFromIP(currentIP)
		log.Printf("SWARM_NETWORK_SUBNET not set, inferred: %s", networkSubnet)
	}

	// Parsear CIDR
	_, ipNet, err := net.ParseCIDR(networkSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid network subnet %s: %v", networkSubnet, err)
	}

	log.Printf("Scanning network range: %s", networkSubnet)

	// Generar lista de IPs a escanear
	ipsToScan := generateIPsFromCIDR(ipNet)

	// Limitar el número de IPs a escanear (evitar escaneos masivos)
	maxIPs := 254
	if len(ipsToScan) > maxIPs {
		log.Printf("Network range too large (%d IPs), limiting to %d", len(ipsToScan), maxIPs)
		ipsToScan = ipsToScan[:maxIPs]
	}

	log.Printf("Scanning %d IP addresses for healthy nodes...", len(ipsToScan))

	// Canal para resultados de escaneo
	results := make(chan string, len(ipsToScan))

	// Escanear IPs en paralelo (con límite de goroutines)
	maxConcurrent := 20
	semaphore := make(chan struct{}, maxConcurrent)

	for _, ip := range ipsToScan {
		go func(ipAddr string) {
			semaphore <- struct{}{}        // Adquirir semáforo
			defer func() { <-semaphore }() // Liberar semáforo

			if isNodeHealthy(ipAddr) {
				results <- ipAddr
			}
		}(ip)
	}

	// Esperar un tiempo razonable para respuestas
	timeout := time.After(5 * time.Second)
	healthyIPs := []string{}

	for i := 0; i < len(ipsToScan); i++ {
		select {
		case ip := <-results:
			if ip != "" {
				healthyIPs = append(healthyIPs, ip)
				log.Printf("Found healthy node at: %s", ip)
			}
		case <-timeout:
			log.Printf("Scan timeout reached, found %d healthy nodes", len(healthyIPs))
			return healthyIPs, nil
		}
	}

	if len(healthyIPs) == 0 {
		return nil, fmt.Errorf("no healthy nodes found in network range")
	}

	return healthyIPs, nil
}

// inferSubnetFromIP intenta inferir el subnet CIDR basado en la IP actual
func inferSubnetFromIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "10.0.0.0/24" // Default fallback
	}

	// Asumir una red /24 (clase C privada)
	return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
}

// generateIPsFromCIDR genera una lista de todas las IPs en un rango CIDR
func generateIPsFromCIDR(ipNet *net.IPNet) []string {
	var ips []string

	// Obtener la primera IP de la red
	ip := ipNet.IP.Mask(ipNet.Mask)

	// Iterar sobre todas las IPs en el rango
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		// Evitar IP de red y broadcast
		if !isNetworkOrBroadcast(ip, ipNet) {
			ips = append(ips, ip.String())
		}
	}

	return ips
}

// incIP incrementa una dirección IP en 1
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isNetworkOrBroadcast verifica si una IP es la dirección de red o broadcast
func isNetworkOrBroadcast(ip net.IP, ipNet *net.IPNet) bool {
	// IP de red (todos los bits de host en 0)
	if ip.Equal(ipNet.IP.Mask(ipNet.Mask)) {
		return true
	}

	// IP de broadcast (todos los bits de host en 1)
	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}

	return ip.Equal(broadcast)
}

// isNodeHealthy verifica si un nodo en la IP dada está healthy
// intentando conectarse al endpoint /cluster/status
func isNodeHealthy(ip string) bool {
	url := fmt.Sprintf("http://%s:3000/cluster/status", ip)

	client := &http.Client{
		Timeout: 2 * time.Second, // Timeout corto para escaneo rápido
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Verificar que el código de respuesta sea exitoso
	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Intentar parsear la respuesta para confirmar que es un nodo válido
	var clusterInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&clusterInfo); err != nil {
		return false
	}

	// Verificar que tenga la estructura esperada
	if _, ok := clusterInfo["current_node_id"]; !ok {
		return false
	}

	return true
}
