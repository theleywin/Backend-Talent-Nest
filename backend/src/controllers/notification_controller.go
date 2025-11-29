package controllers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"gorm.io/gorm"
)

// GetUserNotifications returns all notifications for the authenticated user, populating related user and post data
func GetUserNotifications(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Obtener notificaciones del usuario ordenadas por fecha con relaciones precargadas
	var notifications []models.Notification
	err := lib.DB.Preload("RelatedUser", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "username", "profile_picture")
	}).Preload("RelatedPost", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "content", "image")
	}).Where("recipient_id = ?", user.ID).
		Order("created_at DESC").
		Find(&notifications).Error

	if err != nil {
		fmt.Printf("Error finding notifications: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	// Crear slice para la respuesta
	type NotificationResponse struct {
		ID          uint        `json:"id"`
		Recipient   uint        `json:"recipient"`
		Type        string      `json:"type"`
		Read        bool        `json:"read"`
		CreatedAt   string      `json:"createdAt"`
		UpdatedAt   string      `json:"updatedAt"`
		RelatedUser interface{} `json:"relatedUser,omitempty"`
		RelatedPost interface{} `json:"relatedPost,omitempty"`
	}

	var response []NotificationResponse

	// Procesar cada notificación
	for _, notification := range notifications {
		respItem := NotificationResponse{
			ID:        notification.ID,
			Recipient: notification.RecipientID,
			Type:      notification.Type,
			Read:      notification.Read,
			CreatedAt: notification.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: notification.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Popular usuario relacionado si existe
		if notification.RelatedUserID != nil && notification.RelatedUser != nil {
			respItem.RelatedUser = map[string]interface{}{
				"_id":            notification.RelatedUser.ID,
				"name":           notification.RelatedUser.Name,
				"username":       notification.RelatedUser.Username,
				"profilePicture": notification.RelatedUser.ProfilePicture,
			}
		}

		// Popular post relacionado si existe
		if notification.RelatedPostID != nil && notification.RelatedPost != nil {
			respItem.RelatedPost = map[string]interface{}{
				"_id":     notification.RelatedPost.ID,
				"content": notification.RelatedPost.Content,
				"image":   notification.RelatedPost.Image,
			}
		}

		response = append(response, respItem)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// MarkNotificationAsRead marks a notification as read for the authenticated user
func MarkNotificationAsRead(c *fiber.Ctx) error {
	// Obtener ID de la notificación desde los parámetros
	notificationIDStr := c.Params("id")
	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid notification ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar y actualizar la notificación
	var notification models.Notification
	err = lib.DB.Where("id = ? AND recipient_id = ?", uint(notificationID), user.ID).
		First(&notification).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Notification not found or you don't have permission to update it",
			})
		}
		fmt.Printf("Error in MarkNotificationAsRead: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	// Actualizar el campo read
	notification.Read = true
	if err := lib.DB.Save(&notification).Error; err != nil {
		fmt.Printf("Error updating notification: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(notification)
}

// DeleteNotification deletes a notification for the authenticated user
func DeleteNotification(c *fiber.Ctx) error {
	// Obtener ID de la notificación desde los parámetros
	notificationIDStr := c.Params("id")
	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid notification ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar la notificación primero para verificar que existe y pertenece al usuario
	var notification models.Notification
	err = lib.DB.Where("id = ? AND recipient_id = ?", uint(notificationID), user.ID).
		First(&notification).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Notification not found or you don't have permission to delete it",
			})
		}
		fmt.Printf("Error in DeleteNotification: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Eliminar la notificación
	if err := lib.DB.Delete(&notification).Error; err != nil {
		fmt.Printf("Error deleting notification: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Notification deleted successfully",
	})
}
