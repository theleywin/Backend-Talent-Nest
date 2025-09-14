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

	app.Use(cors.New())
	// Or extend your config for customization
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "https://gofiber.io, https://gofiber.net",
	// 	AllowHeaders: "Origin, Content-Type, Accept",
	// }))

	lib.ConnectDB()

	routes.UserRoutes(app)

	var port string = os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}

	app.Static("/", "./public")

	fmt.Println("Server is running on http://localhost:" + port)
	app.Listen(":" + port)
}
