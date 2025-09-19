package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

// AuthRoutes sets up authentication-related routes for signup, login, logout, and getting the current user
func AuthRoutes(app *fiber.App) {
	auth := app.Group("/api/v1/auth")

	auth.Post("/signup", controllers.Signup)
	auth.Post("/login", controllers.Login)
	auth.Post("/logout", controllers.Logout)
	auth.Get("/me", middleware.ProtectRoute, controllers.GetCurrentUser)
}
