package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

// UserRoutes sets up user-related routes for suggestions, public profile, and profile update
func UserRoutes(app *fiber.App) {
	user := app.Group("/api/v1/users", middleware.ProtectRoute)

	user.Get("/suggestions", controllers.GetSuggestedConnections)
	user.Get("/:username", controllers.GetPublicProfile)
	user.Put("/profile", controllers.UpdateProfile)
}
