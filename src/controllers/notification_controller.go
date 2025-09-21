package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetUserNotifications returns all notifications for the authenticated user, populating related user and post data
func GetUserNotifications(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Obtener notificaciones del usuario ordenadas por fecha
	collection := lib.DB.Collection("notifications")
	filter := bson.M{"recipient": user.Id}
	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(c.Context(), filter, opts)
	if err != nil {
		fmt.Printf("Error finding notifications: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}
	defer cursor.Close(c.Context())

	var notifications []models.Notification
	if err := cursor.All(c.Context(), &notifications); err != nil {
		fmt.Printf("Error decoding notifications: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	// Crear slice para la respuesta
	type NotificationResponse struct {
		ID          primitive.ObjectID      `json:"id" bson:"_id"`
		Recipient   primitive.ObjectID      `json:"recipient" bson:"recipient"`
		Type        models.NotificationType `json:"type" bson:"type"`
		Read        bool                    `json:"read" bson:"read"`
		CreatedAt   time.Time               `json:"createdAt" bson:"createdAt"`
		UpdatedAt   time.Time               `json:"updatedAt" bson:"updatedAt"`
		RelatedUser *models.User            `json:"relatedUser,omitempty" bson:"relatedUser,omitempty"`
		RelatedPost *models.Post            `json:"relatedPost,omitempty" bson:"relatedPost,omitempty"`
	}

	var response []NotificationResponse

	// Procesar cada notificación para popular los datos relacionados
	for _, notification := range notifications {
		respItem := NotificationResponse{
			ID:        notification.Id,
			Recipient: notification.Recipient,
			Type:      notification.Type,
			Read:      notification.Read,
			CreatedAt: notification.CreatedAt,
			UpdatedAt: notification.UpdatedAt,
		}

		// Popular usuario relacionado si existe
		if !notification.RelatedUser.IsZero() {
			var relatedUser models.User
			usersCollection := lib.DB.Collection("users")
			err := usersCollection.FindOne(
				c.Context(),
				bson.M{"_id": notification.RelatedUser},
				options.FindOne().SetProjection(bson.M{
					"name":            1,
					"username":        1,
					"profile_picture": 1,
				}),
			).Decode(&relatedUser)

			if err == nil {
				respItem.RelatedUser = &relatedUser
			} else if err != mongo.ErrNoDocuments {
				fmt.Printf("Error finding related user: %v\n", err)
			}
		}

		// Popular post relacionado si existe
		if !notification.RelatedPost.IsZero() {
			var relatedPost models.Post
			postsCollection := lib.DB.Collection("posts")
			err := postsCollection.FindOne(
				c.Context(),
				bson.M{"_id": notification.RelatedPost},
				options.FindOne().SetProjection(bson.M{
					"content": 1,
					"image":   1,
				}),
			).Decode(&relatedPost)

			if err == nil {
				respItem.RelatedPost = &relatedPost
			} else if err != mongo.ErrNoDocuments {
				fmt.Printf("Error finding related post: %v\n", err)
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
	notificationID, err := primitive.ObjectIDFromHex(notificationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid notification ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Configurar la actualización
	filter := bson.M{
		"_id":       notificationID,
		"recipient": user.Id, // Solo permitir actualizar notificaciones del usuario autenticado
	}
	update := bson.M{
		"$set": bson.M{
			"read":      true,
			"updatedAt": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After) // Devolver el documento actualizado

	// Ejecutar la actualización
	collection := lib.DB.Collection("notifications")
	var updatedNotification models.Notification
	err = collection.FindOneAndUpdate(c.Context(), filter, update, opts).Decode(&updatedNotification)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Notification not found or you don't have permission to update it",
			})
		}
		fmt.Printf("Error in MarkNotificationAsRead: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(updatedNotification)
}

// DeleteNotification deletes a notification for the authenticated user
func DeleteNotification(c *fiber.Ctx) error {
	// Obtener ID de la notificación desde los parámetros
	notificationIDStr := c.Params("id")
	notificationID, err := primitive.ObjectIDFromHex(notificationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid notification ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Configurar el filtro para eliminar solo notificaciones del usuario autenticado
	filter := bson.M{
		"_id":       notificationID,
		"recipient": user.Id,
	}

	// Ejecutar la eliminación
	collection := lib.DB.Collection("notifications")
	result, err := collection.DeleteOne(c.Context(), filter)
	if err != nil {
		fmt.Printf("Error in DeleteNotification: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Verificar si se eliminó algún documento
	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Notification not found or you don't have permission to delete it",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Notification deleted successfully",
	})
}
