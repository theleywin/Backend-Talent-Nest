package cluster

import (
	"bytes"
	"io"

	"github.com/gofiber/fiber/v2"
)

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
