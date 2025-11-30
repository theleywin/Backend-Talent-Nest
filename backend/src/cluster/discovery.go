package cluster

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// DiscoverNodes usa DNS lookup para descubrir todos los nodos del servicio backend
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
	if err != nil {
		return fmt.Errorf("error looking up service %s: %v", cs.ServiceName, err)
	}

	log.Printf("Discovered %d nodes via DNS", len(ips))

	// Marcar todos los nodos existentes como no vistos
	for _, node := range cs.Nodes {
		node.IsHealthy = false
	}

	// Actualizar o crear nodos descubiertos
	for _, ip := range ips {
		ipStr := ip.String()

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
