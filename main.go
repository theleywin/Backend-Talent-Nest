package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/routes"
)

func main() {

	app := fiber.New()

	// Enable CORS middleware for cross-origin requests
	app.Use(cors.New())
	// Or extend your config for customization
	// app.Use(cors.New(cors.Config{
	//  AllowOrigins: "https://gofiber.io, https://gofiber.net",
	//  AllowHeaders: "Origin, Content-Type, Accept",
	// }))

	// Connect to MongoDB database
	lib.ConnectDB()

	// Register routes
	routes.UserRoutes(app)
	routes.AuthRoutes(app)
	routes.PostRoutes(app)

	// Get the server port from environment variable or use default
	var port string = os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Serve static files from the public directory
	app.Static("/", "./public")

	fmt.Println("Server is running on http://localhost:" + port)
	// Start the Fiber server on the specified port
	app.Listen(":" + port)
}
