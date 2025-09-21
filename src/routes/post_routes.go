package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

// PostRoutes sets up post-related routes for feed, creation, deletion, details, comments, and likes
func PostRoutes(app *fiber.App) {
	post := app.Group("/api/v1/posts", middleware.ProtectRoute)

	post.Get("/", controllers.GetFeedPosts)
	post.Post("/create", controllers.CreatePost)
	post.Delete("/delete/:id", controllers.DeletePost)
	post.Get("/:id", controllers.GetPostByID)
	post.Post("/:id/comment", controllers.CreateComment)
	post.Post("/:id/like", controllers.LikePost)
}
