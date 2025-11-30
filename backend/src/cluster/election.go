package cluster

import (
	"log"
	"time"
)

// ElectLeader selecciona el líder basándose en el ID más alto
func (cs *ClusterState) ElectLeader() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Obtener todos los nodos saludables ordenados por ID (descendente)
	nodes := cs.getAllNodesUnsafe()

	if len(nodes) == 0 {
		log.Println("No healthy nodes found for leader election")
		return
	}

	// El nodo con el ID más alto es el líder
	newLeaderID := nodes[0].ID
	newLeaderAddress := nodes[0].Address

	// Verificar si hay cambio de líder
	if cs.LeaderID != newLeaderID {
		oldLeaderID := cs.LeaderID
		cs.LeaderID = newLeaderID
		cs.LeaderAddress = newLeaderAddress

		log.Printf("Leader changed: Old=%d, New=%d", oldLeaderID, newLeaderID)

		// Actualizar el rol del nodo actual
		if cs.CurrentNodeID == newLeaderID {
			cs.CurrentRole = Leader
			log.Printf("This node (ID=%d) is now the LEADER", cs.CurrentNodeID)
		} else {
			cs.CurrentRole = Follower
			log.Printf("This node (ID=%d) is now a FOLLOWER. Leader is ID=%d", cs.CurrentNodeID, newLeaderID)
		}

		// Actualizar roles en el mapa de nodos
		for _, node := range cs.Nodes {
			if node.ID == newLeaderID {
				node.Role = Leader
			} else {
				node.Role = Follower
			}
		}
	}
}

// getAllNodesUnsafe retorna nodos sin lock (usar solo dentro de funciones con lock)
func (cs *ClusterState) getAllNodesUnsafe() []*Node {
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

// StartLeaderElection inicia el proceso de elección de líder cada 10 segundos
func (cs *ClusterState) StartLeaderElection() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			log.Println("Starting leader election cycle...")

			// Descubrir nodos
			if err := cs.DiscoverNodes(); err != nil {
				log.Printf("Error discovering nodes: %v", err)
				continue
			}

			// Elegir líder
			cs.ElectLeader()

			// Mostrar estado del cluster
			cs.PrintClusterState()
		}
	}()

	log.Println("Leader election process started (every 10 seconds)")
}

// PrintClusterState imprime el estado actual del cluster
func (cs *ClusterState) PrintClusterState() {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	log.Println("========== Cluster State ==========")
	log.Printf("Current Node ID: %d", cs.CurrentNodeID)
	log.Printf("Current Role: %s", cs.CurrentRole)
	log.Printf("Leader ID: %d", cs.LeaderID)
	log.Printf("Leader Address: %s", cs.LeaderAddress)
	log.Printf("Total Healthy Nodes: %d", len(cs.Nodes))

	for _, node := range cs.Nodes {
		log.Printf("  Node ID=%d, Role=%s, Address=%s, Healthy=%v",
			node.ID, node.Role, node.Address, node.IsHealthy)
	}
	log.Println("===================================")
}
