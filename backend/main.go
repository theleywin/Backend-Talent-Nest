package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/theleywin/Backend-Talent-Nest/src/cluster"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/routes"
)

var ClusterState *cluster.ClusterState

func main() {

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://frontend-service:5173, http://localhost:5173",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Inicializar el sistema de cluster
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "backend-service" // Nombre por defecto del alias de red
	}

	ClusterState = cluster.NewClusterState(serviceName)
	
	// Connect to SQLite database
	lib.ConnectDB()
	lib.AutoMigrate()

	// Descubrimiento inicial de nodos
	if err := ClusterState.DiscoverNodes(); err != nil {
		fmt.Printf("Warning: Initial node discovery failed: %v\n", err)
	}

	// Elección inicial de líder
	ClusterState.ElectLeader(lib.DB)

	// Iniciar proceso de elección de líder cada 10 segundos (con acceso a la DB)
	ClusterState.StartLeaderElection(lib.DB)

	// Registrar el hook de replicación en GORM
	replicationHook := &cluster.ReplicationHook{
		ClusterState: ClusterState,
	}
	if err := lib.DB.Use(replicationHook); err != nil {
		fmt.Printf("Error: Failed to register replication hook: %v\n", err)
	}

	// Si el nodo es seguidor, solicitar sincronización completa del líder
	if !ClusterState.IsLeader() && ClusterState.GetLeaderAddress() != "" {
		fmt.Println("This node is a follower, requesting full sync from leader...")
		if err := ClusterState.RequestFullSync(ClusterState.GetLeaderAddress()); err != nil {
			fmt.Printf("Warning: Failed to sync from leader: %v\n", err)
			fmt.Println("Node will start with local database")
		}
	}

	// Si el nodo es líder, marcarlo como listo inmediatamente
	if ClusterState.IsLeader() {
		ClusterState.SetReady(true)
	}

	// Aplicar middleware de readiness check
	app.Use(cluster.ReadinessCheck(ClusterState))

	// Aplicar middleware de redirección al líder
	app.Use(cluster.ReplicationMiddleware(ClusterState))

	// Register routes
	routes.UserRoutes(app)
	routes.AuthRoutes(app)
	routes.PostRoutes(app)
	routes.NotificationRoutes(app)
	routes.ConnectionRoutes(app)

	// Ruta para consultar estado del cluster
	app.Get("/cluster/status", func(c *fiber.Ctx) error {
		return c.JSON(ClusterState.GetClusterInfo())
	})

	// Ruta para recibir mensajes de replicación (solo seguidores)
	app.Post("/cluster/replicate", func(c *fiber.Ctx) error {
		if ClusterState.IsLeader() {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Leader node should not receive replication messages",
			})
		}

		var message cluster.ReplicationMessage
		if err := c.BodyParser(&message); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid replication message",
			})
		}

		if err := ClusterState.ApplyReplication(message, lib.DB); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "replicated",
		})
	})

	// Ruta para proporcionar sincronización completa (solo líder)
	app.Post("/cluster/sync", func(c *fiber.Ctx) error {
		if !ClusterState.IsLeader() {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Only leader can provide sync data",
			})
		}

		var request cluster.SyncRequest
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid sync request",
			})
		}

		fmt.Printf("Received sync request from node %d\n", request.NodeID)

		response, err := ClusterState.ProvideSyncData()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(response)
	})

	// Get the server port from environment variable or use default
	var port string = os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Serve static files from the public directory
	app.Static("/", "./public")

	fmt.Printf("Server is running on port %s (Node ID: %d, Role: %s)\n",
		port, ClusterState.GetCurrentNodeID(), ClusterState.GetCurrentRole())
	// Start the Fiber server on the specified port
	app.Listen(":" + port)
}
