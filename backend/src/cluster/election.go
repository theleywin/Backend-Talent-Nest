package cluster

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ElectLeader selecciona el líder basándose en el ID más bajo
func (cs *ClusterState) ElectLeader() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Obtener todos los nodos saludables ordenados por ID (aescendente)
	nodes := cs.getAllNodesUnsafe()

	if len(nodes) == 0 {
		log.Println("No healthy nodes found for leader election")
		return
	}

	// El nodo con el ID más bajo es el líder
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
			cs.IsReady = true // El líder siempre está listo
			log.Printf("This node (ID=%d) is now the LEADER", cs.CurrentNodeID)
		} else {
			cs.CurrentRole = Follower
			log.Printf("This node (ID=%d) is now a FOLLOWER. Leader is ID=%d", cs.CurrentNodeID, newLeaderID)
			// necesita resincronizarse
			cs.IsReady = false
			log.Println("Former leader demoted to follower, will request sync")
			go func() {
				time.Sleep(5 * time.Second) // Esperar un poco para que el nuevo líder se estabilice
				if err := cs.RequestFullSync(); err != nil {
					log.Printf("Error syncing after demotion: %v", err)
				}
			}()

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

	// Ordenar por ID (menor a mayor)
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].ID > nodes[j].ID {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}

	return nodes
}

// StartLeaderElection inicia el proceso de elección de líder cada 10 segundos
func (cs *ClusterState) StartLeaderElection(db *gorm.DB) {
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

			// Mostrar estado de la base de datos
			cs.PrintDatabaseState(db)
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

// PrintDatabaseState imprime el contenido de todas las tablas de la base de datos
func (cs *ClusterState) PrintDatabaseState(db *gorm.DB) {
	log.Println("\n========== Database State ==========")
	log.Printf("Node ID: %d | Role: %s", cs.CurrentNodeID, cs.CurrentRole)

	tables := []string{"users", "posts", "connections", "notifications"}

	for _, table := range tables {
		var count int64

		// Contar registros
		if err := db.Table(table).Count(&count).Error; err != nil {
			log.Printf("[%s] Error counting: %v", table, err)
			continue
		}

		log.Printf("\n[Table: %s] Total records: %d", table, count)

		if count == 0 {
			log.Printf("  (empty)")
			continue
		}

		// Obtener todos los registros como mapas
		var records []map[string]interface{}
		if err := db.Table(table).Find(&records).Error; err != nil {
			log.Printf("  Error fetching records: %v", err)
			continue
		}

		// Imprimir cada registro
		log.Printf("  Records:")
		for i, record := range records {
			// Construir string con todos los campos dinámicamente
			var fields []string
			for key, value := range record {
				// Formatear el valor según su tipo
				var formattedValue string
				switch v := value.(type) {
				case string:
					// Limitar strings largos a 50 caracteres
					if len(v) > 50 {
						formattedValue = v[:47] + "..."
					} else {
						formattedValue = v
					}
				case nil:
					formattedValue = "<nil>"
				default:
					formattedValue = fmt.Sprintf("%v", v)
				}
				fields = append(fields, fmt.Sprintf("%s=%s", key, formattedValue))
			}
			log.Printf("    [%d] %s", i+1, strings.Join(fields, " | "))
		}
	}

	log.Println("\n=====================================")
}
