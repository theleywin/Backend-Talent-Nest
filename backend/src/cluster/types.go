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
}

func NewClusterState(serviceName string) *ClusterState {
	return &ClusterState{
		Nodes:       make(map[int]*Node),
		CurrentRole: Follower,
		ServiceName: serviceName,
	}
}
