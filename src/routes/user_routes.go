package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
)

func UserRoutes(app *fiber.App) {
	user := app.Group("/users")
	user.Post("/", controllers.TestCreateUser)
	user.Get("/", controllers.TestGetUsers)
	//Add more routes here
}
