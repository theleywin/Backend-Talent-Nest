package cluster

// IsLeader verifica si el nodo actual es el líder
func (cs *ClusterState) IsLeader() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.CurrentRole == Leader
}

// GetLeaderAddress retorna la dirección del líder actual
func (cs *ClusterState) GetLeaderAddress() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.LeaderAddress
}

// GetCurrentNodeID retorna el ID del nodo actual
func (cs *ClusterState) GetCurrentNodeID() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.CurrentNodeID
}

// GetCurrentRole retorna el rol del nodo actual
func (cs *ClusterState) GetCurrentRole() NodeRole {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.CurrentRole
}

// GetClusterInfo retorna información del cluster para API
func (cs *ClusterState) GetClusterInfo() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	nodes := make([]map[string]interface{}, 0)
	for _, node := range cs.Nodes {
		nodes = append(nodes, map[string]interface{}{
			"id":        node.ID,
			"address":   node.Address,
			"role":      node.Role,
			"is_leader": node.ID == cs.LeaderID,
			"healthy":   node.IsHealthy,
		})
	}

	return map[string]interface{}{
		"current_node_id": cs.CurrentNodeID,
		"current_role":    cs.CurrentRole,
		"leader_id":       cs.LeaderID,
		"leader_address":  cs.LeaderAddress,
		"total_nodes":     len(cs.Nodes),
		"nodes":           nodes,
	}
}
