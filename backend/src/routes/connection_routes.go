package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

// ConnectionRoutes sets up connection-related routes for sending, accepting, rejecting requests, listing requests, getting connections, removing connections, and checking connection status
func ConnectionRoutes(app *fiber.App) {
	connection := app.Group("/api/v1/connections", middleware.ProtectRoute)

	connection.Post("/request/:userId", controllers.SendConnectionRequest)
	connection.Put("/accept/:requestId", controllers.AcceptConnectionRequest)
	connection.Put("/reject/:requestId", controllers.RejectConnectionRequest)
	connection.Get("/requests", controllers.GetConnectionRequests)
	connection.Get("/", controllers.GetUserConnections)
	connection.Delete("/:userId", controllers.RemoveConnection)
	connection.Get("/status/:userId", controllers.GetConnectionStatus)
}
