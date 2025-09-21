package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/controllers"
	"github.com/theleywin/Backend-Talent-Nest/src/middleware"
)

// NotificationRoutes sets up notification-related routes for listing, marking as read, and deleting notifications
func NotificationRoutes(app *fiber.App) {
	notification := app.Group("/api/v1/notifications", middleware.ProtectRoute)

	notification.Get("/", controllers.GetUserNotifications)
	notification.Put("/:id/read", controllers.MarkNotificationAsRead)
	notification.Delete("/:id", controllers.DeleteNotification)
}
