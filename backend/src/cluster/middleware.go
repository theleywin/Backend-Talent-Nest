package cluster

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AutoReplicationMiddleware captura operaciones exitosas del líder y las replica automáticamente
func AutoReplicationMiddleware(clusterState *ClusterState) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()

		// Solo aplica para escrituras en el líder
		if !clusterState.IsLeader() || !isWriteOperation(method) || isClusterEndpoint(path) || isReadOnlyOperation(path) {
			return c.Next()
		}

		log.Printf("[AutoReplication] Intercepting %s %s on leader", method, path) // Capturar body original
		var bodyData map[string]interface{}
		if len(c.Body()) > 0 {
			bodyBytes := make([]byte, len(c.Body()))
			copy(bodyBytes, c.Body())
			json.Unmarshal(bodyBytes, &bodyData)
		}

		// Ejecutar el handler
		err := c.Next()

		// Si fue exitoso (2xx), replicar
		statusCode := c.Response().StatusCode()
		log.Printf("[AutoReplication] Response status: %d", statusCode)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			// Capturar respuesta ANTES del goroutine (el contexto expira después)
			responseBodyBytes := make([]byte, len(c.Response().Body()))
			copy(responseBodyBytes, c.Response().Body())

			log.Printf("[AutoReplication] Response body: %s", string(responseBodyBytes))

			// Extraer ID del registro de la respuesta
			var recordID uint
			var responseBody map[string]interface{}

			if json.Unmarshal(responseBodyBytes, &responseBody) == nil {
				if id, ok := responseBody["_id"].(float64); ok {
					recordID = uint(id)
				} else if user, ok := responseBody["user"].(map[string]interface{}); ok {
					if id, ok := user["_id"].(float64); ok {
						recordID = uint(id)
					}
				}
			}

			// Determinar tabla y operación
			table := getTableFromPath(path)
			operation := "INSERT"
			if method == "PUT" || method == "PATCH" {
				operation = "UPDATE"
			} else if method == "DELETE" {
				operation = "DELETE"
			}
			
			log.Printf("[AutoReplication] Extracted - Table: %s, Operation: %s, RecordID: %d", table, operation, recordID)

			// Replicar a seguidores en goroutine
			go func() {
				log.Printf("[AutoReplication] Starting replication to followers...")
				log.Printf("[AutoReplication] 📦 Data being sent: %+v", bodyData)
				clusterState.ReplicateToFollowers(operation, table, bodyData, recordID)
				log.Printf("✓ Replicated %s on %s (ID:%d) to followers", operation, table, recordID)
			}()
		}

		return err
	}
}

// ReplicationMiddleware intercepta peticiones de escritura y las redirige al líder si es necesario
func ReplicationMiddleware(clusterState *ClusterState) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()

		// Permitir endpoints del cluster siempre
		if isClusterEndpoint(path) {
			return c.Next()
		}

		// Permitir operaciones de lectura en cualquier nodo
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		// Permitir operaciones de solo lectura (login, logout) en cualquier nodo
		if isReadOnlyOperation(path) {
			return c.Next()
		}

		// Operaciones de escritura (POST, PUT, DELETE, PATCH)
		if isWriteOperation(method) {
			// Si este nodo es el líder, procesar normalmente
			if clusterState.IsLeader() {
				return c.Next()
			}

			// Si este nodo es seguidor, redirigir al líder
			return forwardToLeader(c, clusterState)
		}

		// Por defecto, continuar
		return c.Next()
	}
}

// getTableFromPath determina la tabla de BD basándose en el path de la API
func getTableFromPath(path string) string {
	pathLower := strings.ToLower(path)

	if strings.Contains(pathLower, "/auth/signup") || strings.Contains(pathLower, "/auth/login") {
		return "users"
	}
	if strings.Contains(pathLower, "/users") {
		return "users"
	}
	if strings.Contains(pathLower, "/posts") {
		return "posts"
	}
	if strings.Contains(pathLower, "/comments") {
		return "comments"
	}
	if strings.Contains(pathLower, "/notifications") {
		return "notifications"
	}
	if strings.Contains(pathLower, "/connections") {
		return "connections"
	}
	if strings.Contains(pathLower, "/likes") {
		return "likes"
	}

	return "unknown"
}

// isClusterEndpoint verifica si el path es un endpoint del cluster
func isClusterEndpoint(path string) bool {
	clusterPaths := []string{
		"/cluster/status",
		"/cluster/replicate",
		"/cluster/sync",
	}

	for _, cp := range clusterPaths {
		if path == cp {
			return true
		}
	}
	return false
}

// isWriteOperation verifica si el método HTTP es una operación de escritura
func isWriteOperation(method string) bool {
	writeMethods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, wm := range writeMethods {
		if method == wm {
			return true
		}
	}
	return false
}

// isReadOnlyOperation identifica operaciones POST que no modifican la base de datos
func isReadOnlyOperation(path string) bool {
	readOnlyPaths := []string{
		"/api/v1/auth/login",   // Autenticación, no crea/modifica datos
		"/api/v1/auth/logout",  // Solo limpia sesión, no modifica DB
	}
	
	for _, rop := range readOnlyPaths {
		if path == rop {
			return true
		}
	}
	return false
}

// forwardToLeader redirige la petición al líder
func forwardToLeader(c *fiber.Ctx, clusterState *ClusterState) error {
	leaderAddress := clusterState.GetLeaderAddress()
	if leaderAddress == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "No leader available",
			"message": "The cluster has no active leader to process write operations",
		})
	}

	// Extraer headers relevantes
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Leer el body
	bodyBytes := c.Body()
	bodyReader := bytes.NewReader(bodyBytes)

	// Hacer forward al líder
	resp, err := clusterState.ForwardToLeader(
		c.Method(),
		c.Path(),
		bodyReader,
		headers,
	)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":   "Failed to forward to leader",
			"message": err.Error(),
		})
	}
	defer resp.Body.Close()

	// Copiar la respuesta del líder
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to read leader response",
			"message": err.Error(),
		})
	}

	// Copiar status code y headers
	c.Status(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			c.Set(key, value)
		}
	}

	return c.Send(responseBody)
}

// ReadinessCheck middleware verifica que el nodo esté listo
func ReadinessCheck(clusterState *ClusterState) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Permitir siempre endpoints del cluster
		if isClusterEndpoint(c.Path()) {
			return c.Next()
		}

		// Verificar que el nodo esté listo
		if !clusterState.IsNodeReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "Node not ready",
				"message": "This node is still synchronizing data from the leader",
			})
		}

		return c.Next()
	}
}
