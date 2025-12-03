package cluster

import (
	"sync"
	"time"
)

type NodeRole string

const (
	Leader   NodeRole = "leader"
	Follower NodeRole = "follower"
)

type Node struct {
	ID        int
	Address   string
	Role      NodeRole
	LastSeen  time.Time
	IsHealthy bool
}

type ClusterState struct {
	mu            sync.RWMutex
	CurrentNodeID int
	CurrentRole   NodeRole
	LeaderID      int
	LeaderAddress string
	Nodes         map[int]*Node
	ServiceName   string
	IsReady       bool // Indica si el nodo está listo para aceptar requests
}

// ReplicationMessage representa un mensaje de replicación del líder a los seguidores
type ReplicationMessage struct {
	Operation string                 `json:"operation"` // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`     // Nombre de la tabla
	Data      map[string]interface{} `json:"data"`      // Datos a replicar
	LeaderID  int                    `json:"leader_id"`
	Timestamp time.Time              `json:"timestamp"`
	RecordID  uint                   `json:"record_id,omitempty"` // ID del registro afectado
}

// SyncRequest representa una solicitud de sincronización completa
type SyncRequest struct {
	NodeID    int       `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
}

// SyncResponse contiene la base de datos completa en formato base64
type SyncResponse struct {
	Database  string    `json:"database"` // Base64 del archivo SQLite
	LeaderID  int       `json:"leader_id"`
	Timestamp time.Time `json:"timestamp"`
}

func NewClusterState(serviceName string) *ClusterState {
	return &ClusterState{
		Nodes:       make(map[int]*Node),
		CurrentRole: Follower,
		ServiceName: serviceName,
		IsReady:     false,
	}
}
