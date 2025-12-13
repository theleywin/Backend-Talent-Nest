package cluster

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ReplicateToFollowers envía un mensaje de replicación a todos los seguidores
func (cs *ClusterState) ReplicateToFollowers(operation, table string, data map[string]interface{}, recordID uint) {
	if !cs.IsLeader() {
		log.Println("Warning: Non-leader node attempted to replicate")
		return
	}

	cs.mu.RLock()
	followers := make([]*Node, 0)
	for _, node := range cs.Nodes {
		if node.ID != cs.CurrentNodeID && node.IsHealthy {
			followers = append(followers, node)
		}
	}
	cs.mu.RUnlock()

	if len(followers) == 0 {
		log.Println("No followers to replicate to")
		return
	}

	message := ReplicationMessage{
		Operation: operation,
		Table:     table,
		Data:      data,
		LeaderID:  cs.CurrentNodeID,
		Timestamp: time.Now(),
		RecordID:  recordID,
	}

	log.Printf("Replicating %s operation on table %s to %d followers", operation, table, len(followers))

	// Enviar a todos los seguidores en paralelo
	for _, follower := range followers {
		go func(node *Node) {
			if err := cs.sendReplicationMessage(node.Address, message); err != nil {
				log.Printf("Failed to replicate to node %d: %v", node.ID, err)
			} else {
				log.Printf("Successfully replicated to node %d", node.ID)
			}
		}(follower)
	}
}

// sendReplicationMessage envía un mensaje de replicación a un seguidor específico
func (cs *ClusterState) sendReplicationMessage(address string, message ReplicationMessage) error {
	url := fmt.Sprintf("%s/cluster/replicate", address)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling replication message: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending replication message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("replication failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ApplyReplication aplica un mensaje de replicación en un nodo seguidor
func (cs *ClusterState) ApplyReplication(message ReplicationMessage, db interface{}) error {
	if cs.IsLeader() {
		return fmt.Errorf("leader node should not receive replication messages")
	}

	// Verificar que el mensaje venga del líder actual
	if message.LeaderID != cs.LeaderID {
		return fmt.Errorf("replication message from non-leader node: %d (expected: %d)", message.LeaderID, cs.LeaderID)
	}

	log.Printf("Applying replication: %s on table %s (RecordID: %d)", message.Operation, message.Table, message.RecordID)

	// Convertir db interface a *gorm.DB
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return fmt.Errorf("invalid database instance")
	}

	// Aplicar la operación según el tipo
	switch message.Operation {
	case "INSERT":
		return cs.applyInsert(message.Table, message.Data, gormDB)
	case "UPDATE":
		return cs.applyUpdate(message.Table, message.RecordID, message.Data, gormDB)
	case "DELETE":
		return cs.applyDelete(message.Table, message.RecordID, gormDB)
	default:
		return fmt.Errorf("unknown operation: %s", message.Operation)
	}
}

// applyInsert aplica una operación INSERT replicada
func (cs *ClusterState) applyInsert(table string, data map[string]interface{}, db *gorm.DB) error {
	log.Printf("[Replication] Inserting into %s: %v", table, data)

	// Los datos ya vienen completos del hook de GORM con todos los campos correctos
	// Solo necesitamos asegurarnos de que los timestamps estén en el formato correcto

	// Ejecutar INSERT usando GORM
	result := db.Table(table).Create(data)
	if result.Error != nil {
		log.Printf("[Replication] ❌ Error inserting into %s: %v", table, result.Error)
		return fmt.Errorf("error inserting into %s: %v", table, result.Error)
	}
	log.Printf("[Replication] ✓ Successfully inserted into %s (rows affected: %d)", table, result.RowsAffected)
	return nil
}

// applyUpdate aplica una operación UPDATE replicada
func (cs *ClusterState) applyUpdate(table string, recordID uint, data map[string]interface{}, db *gorm.DB) error {
	log.Printf("Updating %s record %d: %v", table, recordID, data)

	// Ejecutar UPDATE en la tabla correspondiente
	result := db.Table(table).Where("id = ?", recordID).Updates(data)
	if result.Error != nil {
		return fmt.Errorf("error updating record: %v", result.Error)
	}

	log.Printf("Updated %d rows in %s", result.RowsAffected, table)
	return nil
}

// applyDelete aplica una operación DELETE replicada
func (cs *ClusterState) applyDelete(table string, recordID uint, db *gorm.DB) error {
	log.Printf("Deleting from %s record %d", table, recordID)

	// Ejecutar soft delete (deleted_at)
	result := db.Table(table).Where("id = ?", recordID).Update("deleted_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("error deleting record: %v", result.Error)
	}

	log.Printf("Deleted record %d from %s", recordID, table)
	return nil
}

// RequestFullSync solicita una sincronización completa de la base de datos al líder
func (cs *ClusterState) RequestFullSync(leaderAddress string) error {
	if leaderAddress == "" {
		return fmt.Errorf("no leader available for sync")
	}

	log.Printf("Requesting full sync from leader at %s", leaderAddress)

	url := fmt.Sprintf("%s/cluster/sync", leaderAddress)

	request := SyncRequest{
		NodeID:    cs.CurrentNodeID,
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling sync request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error requesting sync: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sync request failed with status: %d", resp.StatusCode)
	}

	var syncResponse SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResponse); err != nil {
		return fmt.Errorf("error decoding sync response: %v", err)
	}

	// Decodificar la base de datos de base64
	dbData, err := base64.StdEncoding.DecodeString(syncResponse.Database)
	if err != nil {
		return fmt.Errorf("error decoding database: %v", err)
	}

	log.Printf("📥 Received database from leader (%d bytes)", len(dbData))

	// 🔍 PREVIEW DE DATOS RECIBIDOS (antes de aplicar)
	if err := previewSyncedDatabase(dbData); err != nil {
		log.Printf("⚠️  Warning: Could not preview synced database: %v", err)
	}

	// Guardar la base de datos recibida
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./talentnest.db"
	}

	if err := os.WriteFile(dbPath, dbData, 0644); err != nil {
		return fmt.Errorf("error writing database file: %v", err)
	}

	log.Printf("✅ Successfully synced database from leader (size: %d bytes)", len(dbData))

	// Marcar el nodo como listo
	cs.mu.Lock()
	cs.IsReady = true
	cs.mu.Unlock()

	return nil
}

// previewSyncedDatabase abre temporalmente la DB recibida y muestra su contenido
func previewSyncedDatabase(dbData []byte) error {
	// Crear archivo temporal
	tempPath := "/tmp/preview_sync.db"
	if err := os.WriteFile(tempPath, dbData, 0644); err != nil {
		return fmt.Errorf("error writing temp db: %v", err)
	}
	defer os.Remove(tempPath)

	// Abrir DB temporal con GORM
	dsn := fmt.Sprintf("file:%s?mode=ro", tempPath) // modo read-only
	tempDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("error opening temp db: %v", err)
	}

	log.Println("\n========== PREVIEW: Database Received from Leader ==========")

	tables := []string{"users", "posts", "connections", "notifications"}

	for _, table := range tables {
		var count int64
		if err := tempDB.Table(table).Count(&count).Error; err != nil {
			log.Printf("[%s] Error counting: %v", table, err)
			continue
		}

		log.Printf("\n[Table: %s] Total records: %d", table, count)

		if count == 0 {
			log.Printf("  (empty)")
			continue
		}

		// Obtener primeros 5 registros para preview
		var records []map[string]interface{}
		if err := tempDB.Table(table).Limit(10).Find(&records).Error; err != nil {
			log.Printf("  Error fetching records: %v", err)
			continue
		}

		log.Printf("  Sample records (first 5):")
		for i, record := range records {
			var fields []string
			for key, value := range record {
				var formattedValue string
				switch v := value.(type) {
				case string:
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

	log.Println("============================================================")

	return nil
}

// ProvideSyncData proporciona la base de datos completa a un seguidor (solo líder)
func (cs *ClusterState) ProvideSyncData() (SyncResponse, error) {
	if !cs.IsLeader() {
		return SyncResponse{}, fmt.Errorf("only leader can provide sync data")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./talentnest.db"
	}

	// Leer el archivo de la base de datos
	dbData, err := os.ReadFile(dbPath)
	if err != nil {
		return SyncResponse{}, fmt.Errorf("error reading database file: %v", err)
	}

	// Codificar en base64
	encodedDB := base64.StdEncoding.EncodeToString(dbData)

	response := SyncResponse{
		Database:  encodedDB,
		LeaderID:  cs.CurrentNodeID,
		Timestamp: time.Now(),
	}

	log.Printf("Providing sync data (size: %d bytes)", len(dbData))

	return response, nil
}

// ForwardToLeader redirige una petición HTTP al líder
func (cs *ClusterState) ForwardToLeader(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	leaderAddress := cs.GetLeaderAddress()
	if leaderAddress == "" {
		return nil, fmt.Errorf("no leader available")
	}

	url := fmt.Sprintf("%s%s", leaderAddress, path)

	log.Printf("Forwarding %s request to leader: %s", method, url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating forward request: %v", err)
	}

	// Copiar headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return client.Do(req)
}

// IsReady verifica si el nodo está listo para aceptar requests
func (cs *ClusterState) IsNodeReady() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.IsReady
}

// SetReady marca el nodo como listo
func (cs *ClusterState) SetReady(ready bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.IsReady = ready
	log.Printf("Node ready status set to: %v", ready)
}
