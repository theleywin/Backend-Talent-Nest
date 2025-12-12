package cluster

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ElectLeader selecciona el líder basándose en el ID más bajo
func (cs *ClusterState) ElectLeader(db *gorm.DB) {
	cs.mu.Lock()

	// Obtener todos los nodos saludables ordenados por ID (aescendente)
	nodes := cs.getAllNodesUnsafe()

	if len(nodes) == 0 {
		log.Println("No healthy nodes found for leader election")
		return
	}

	// El nodo con el ID más bajo es el líder
	newLeaderID := nodes[0].ID
	newLeaderAddress := nodes[0].Address
	oldLeaderID := cs.LeaderID

	// CASO ESPECIAL: Si este nodo sería el nuevo líder, verificar si es nuevo (DB vacía)
	if cs.CurrentNodeID == newLeaderID && len(nodes) > 1 {
		// Verificar si la base de datos está vacía
		var userCount int64
		if db != nil {
			db.Table("users").Count(&userCount)
		}

		// Si este nodo tiene DB vacía y hay otros nodos, necesita sincronizar primero
		if userCount == 0 {
			log.Println("⚠️  WARNING: This node would become leader but has empty database")
			log.Printf("   Found %d other nodes in the network", len(nodes)-1)

			// Identificar al segundo nodo (el que sería líder si este no existiera)
			var previousLeaderNode *Node
			if len(nodes) > 1 {
				previousLeaderNode = nodes[1]
			}

			if previousLeaderNode != nil {
				log.Printf("   Previous/Alternative leader: ID=%d, Address=%s", previousLeaderNode.ID, previousLeaderNode.Address)
				log.Println("   🔄 Syncing with previous leader before taking leadership...")

				// Liberar el lock temporalmente para hacer la sincronización
				cs.mu.Unlock()

				// Sincronizar con el nodo anterior
				if err := cs.syncFromNode(previousLeaderNode.Address); err != nil {
					log.Printf("   ❌ Failed to sync from previous leader: %v", err)
					log.Println("   Node will become leader with empty database (this may cause data loss!)")
				} else {
					log.Println("   ✅ Successfully synced database from previous leader")
					// Verificar cuántos usuarios tenemos ahora
					if db != nil {
						db.Table("users").Count(&userCount)
						log.Printf("   Database now has %d users", userCount)
					}
				}

				// Re-adquirir el lock
				cs.mu.Lock()
			}
		}
	}

	// Verificar si hay cambio de líder
	hasLeaderChanged := cs.LeaderID != newLeaderID

	// Actualizar líder
	cs.LeaderID = newLeaderID
	cs.LeaderAddress = newLeaderAddress

	if hasLeaderChanged {
		log.Printf("Leader changed: Old=%d, New=%d", oldLeaderID, newLeaderID)

		// Actualizar el rol del nodo actual
		if cs.CurrentNodeID == newLeaderID {
			cs.CurrentRole = Leader
			cs.IsReady = true // El líder siempre está listo
			log.Printf("This node (ID=%d) is now the LEADER", cs.CurrentNodeID)
		} else {
			cs.CurrentRole = Follower
			log.Printf("This node (ID=%d) is now a FOLLOWER. Leader is ID=%d", cs.CurrentNodeID, newLeaderID)

			// Marcar como no listo hasta sincronizar
			cs.IsReady = false

			// Liberar lock antes de iniciar goroutine
			cs.mu.Unlock()

			// Si hay cambio de líder, este nodo necesita resincronizarse
			log.Println("Leader changed, will request sync from new leader")
			go func() {
				time.Sleep(8 * time.Second) // Esperar para que el nuevo líder se estabilice

				// Evitar sincronizaciones concurrentes con retry
				if err := cs.syncWithBackoff(); err != nil {
					log.Printf("❌ Error syncing with new leader after retries: %v", err)
				}
			}()

			return // Ya liberamos el lock
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

	cs.mu.Unlock()
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

// syncWithBackoff intenta sincronizar con retry exponencial
func (cs *ClusterState) syncWithBackoff() error {
	maxRetries := 3
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔄 Sync attempt %d/%d", attempt, maxRetries)

		// Verificar si ya estamos listos (por si otra goroutine ya sincronizó)
		cs.mu.RLock()
		if cs.IsReady {
			cs.mu.RUnlock()
			log.Println("✅ Already synced by another process")
			return nil
		}
		cs.mu.RUnlock()

		// Intentar sincronizar
		if err := cs.RequestFullSync(); err != nil {
			log.Printf("⚠️  Sync attempt %d failed: %v", attempt, err)

			if attempt < maxRetries {
				log.Printf("   Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
				continue
			}

			return fmt.Errorf("all sync attempts failed: %v", err)
		}

		log.Printf("✅ Sync successful on attempt %d", attempt)
		return nil
	}

	return fmt.Errorf("sync failed after %d attempts", maxRetries)
}

// StartLeaderElection inicia el proceso de elección de líder cada 10 segundos
func (cs *ClusterState) StartLeaderElection(db *gorm.DB) {
	ticker := time.NewTicker(10 * time.Second)

	// WaitGroup para evitar race conditions en shutdown
	var wg sync.WaitGroup

	go func() {
		for range ticker.C {
			wg.Add(1)
			func() {
				defer wg.Done()

				log.Println("Starting leader election cycle...")

				// Descubrir nodos
				if err := cs.DiscoverNodes(); err != nil {
					log.Printf("Error discovering nodes: %v", err)
					return
				}

				// Elegir líder (con acceso a DB)
				cs.ElectLeader(db)

				// Mostrar estado del cluster
				cs.PrintClusterState()

				// Mostrar estado de la base de datos
				cs.PrintDatabaseState(db)
			}()
		}

		wg.Wait()
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
	log.Printf("Is Ready: %v", cs.IsReady)
	log.Printf("Total Healthy Nodes: %d", len(cs.Nodes))

	for _, node := range cs.Nodes {
		log.Printf("  Node ID=%d, Role=%s, Address=%s, Healthy=%v",
			node.ID, node.Role, node.Address, node.IsHealthy)
	}
	log.Println("===================================")
}

// PrintDatabaseState imprime el contenido de todas las tablas de la base de datos
func (cs *ClusterState) PrintDatabaseState(db *gorm.DB) {
	// No bloquear con mutex aquí, solo lectura de DB
	log.Println("\n========== Database State ==========")

	cs.mu.RLock()
	nodeID := cs.CurrentNodeID
	role := cs.CurrentRole
	cs.mu.RUnlock()

	log.Printf("Node ID: %d | Role: %s", nodeID, role)

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

		// Obtener todos los registros como mapas (limitar a 10)
		var records []map[string]interface{}
		if err := db.Table(table).Limit(10).Find(&records).Error; err != nil {
			log.Printf("  Error fetching records: %v", err)
			continue
		}

		// Imprimir cada registro (máximo 10)
		log.Printf("  Records (showing up to 10):")
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
