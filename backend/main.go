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

	// Descubrimiento inicial de nodos
	if err := ClusterState.DiscoverNodes(); err != nil {
		fmt.Printf("Warning: Initial node discovery failed: %v\n", err)
	}

	// Elección inicial de líder
	ClusterState.ElectLeader()

	// Iniciar proceso de elección de líder cada 10 segundos
	ClusterState.StartLeaderElection()

	// Connect to SQLite database
	lib.ConnectDB()

	lib.AutoMigrate()

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
