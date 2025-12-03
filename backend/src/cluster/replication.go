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
	"time"
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

	// Aquí se aplicará la operación en la base de datos
	// La implementación específica dependerá de cómo se maneje la BD
	// Por ahora, solo registramos la operación

	return nil
}

// RequestFullSync solicita una sincronización completa de la base de datos al líder
func (cs *ClusterState) RequestFullSync() error {
	leaderAddress := cs.GetLeaderAddress()
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

	// Guardar la base de datos recibida
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./talentnest.db"
	}

	if err := os.WriteFile(dbPath, dbData, 0644); err != nil {
		return fmt.Errorf("error writing database file: %v", err)
	}

	log.Printf("Successfully synced database from leader (size: %d bytes)", len(dbData))

	// Marcar el nodo como listo
	cs.mu.Lock()
	cs.IsReady = true
	cs.mu.Unlock()

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
