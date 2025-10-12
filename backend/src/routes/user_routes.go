package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

func UserRoutes(app *fiber.App) {
	user := app.Group("/api/v1/users", middleware.ProtectRoute)

	user.Get("/suggestions", controllers.GetSuggestedConnections)
	user.Get("/search", controllers.SearchUsers)
	user.Get("/:username", controllers.GetPublicProfile)
	user.Put("/profile", controllers.UpdateProfile)
}
