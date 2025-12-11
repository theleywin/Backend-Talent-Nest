package controllers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"gorm.io/gorm"
)

// SendConnectionRequest sends a connection request from the authenticated user to another user
func SendConnectionRequest(c *fiber.Ctx) error {
	// Obtener ID del usuario destino desde los parámetros
	targetUserIDStr := c.Params("userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no se envíe solicitud a uno mismo
	if user.ID == uint(targetUserID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "You can't send a connection request to yourself",
		})
	}

	// Validar que no estén ya conectados
	var existingConnection models.Connection
	err = lib.DB.Where("(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		user.ID, uint(targetUserID), uint(targetUserID), user.ID).
		Where("status = ?", models.ConnectionStatusAccepted).
		First(&existingConnection).Error

	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "You are already connected with this user",
		})
	}

	// Verificar si ya existe una solicitud pendiente
	var pendingRequest models.Connection
	err = lib.DB.Where("sender_id = ? AND recipient_id = ? AND status = ?",
		user.ID, uint(targetUserID), models.ConnectionStatusPending).
		First(&pendingRequest).Error

	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "A connection request already exists",
		})
	} else if err != gorm.ErrRecordNotFound {
		fmt.Printf("Error checking existing connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Crear nueva solicitud de conexión
	newRequest := models.Connection{
		SenderID:    user.ID,
		RecipientID: uint(targetUserID),
		Status:      models.ConnectionStatusPending,
	}

	// Guardar en la base de datos
	if err := lib.DB.Create(&newRequest).Error; err != nil {
		fmt.Printf("Error creating connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to send connection request",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Connection request sent successfully",
	})
}

// AcceptConnectionRequest accepts a pending connection request and updates both users' connections
func AcceptConnectionRequest(c *fiber.Ctx) error {
	// Obtener ID de la solicitud desde los parámetros
	requestIDStr := c.Params("requestId")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar la solicitud de conexión
	var request models.Connection
	err = lib.DB.First(&request, uint(requestID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Connection request not found",
			})
		}
		fmt.Printf("Error finding connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Verificar que el usuario es el destinatario de la solicitud
	if request.RecipientID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Not authorized to accept this request",
		})
	}

	// Verificar que la solicitud esté pendiente
	if request.Status != models.ConnectionStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "This request has already been processed",
		})
	}

	// Actualizar el estado de la solicitud a "accepted"
	request.Status = models.ConnectionStatusAccepted
	if err := lib.DB.Save(&request).Error; err != nil {
		fmt.Printf("Error updating connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to accept connection request",
		})
	}

	// Crear notificación para el usuario remitente
	notification := models.Notification{
		RecipientID:   request.SenderID,
		Type:          "connectionAccepted",
		RelatedUserID: user.ID,
		Read:          false,
	}

	if err := lib.DB.Create(&notification).Error; err != nil {
		// Log del error pero continuar (la notificación no es crítica)
		fmt.Printf("Error creating notification: %v\n", err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Connection accepted successfully",
	})
}

// RejectConnectionRequest rejects a pending connection request
func RejectConnectionRequest(c *fiber.Ctx) error {
	// Obtener ID de la solicitud desde los parámetros
	requestIDStr := c.Params("requestId")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar la solicitud de conexión
	var request models.Connection
	err = lib.DB.First(&request, uint(requestID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Connection request not found",
			})
		}
		fmt.Printf("Error finding connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Verificar que el usuario es el destinatario de la solicitud
	if request.RecipientID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Not authorized to reject this request",
		})
	}

	// Verificar que la solicitud esté pendiente
	if request.Status != models.ConnectionStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "This request has already been processed",
		})
	}

	// Actualizar el estado de la solicitud a "rejected"
	request.Status = models.ConnectionStatusRejected
	if err := lib.DB.Save(&request).Error; err != nil {
		fmt.Printf("Error rejecting connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to reject connection request",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Connection request rejected",
	})
}

// GetConnectionRequests returns all pending connection requests for the authenticated user
func GetConnectionRequests(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar solicitudes pendientes con Preload del Sender
	var connections []models.Connection
	err := lib.DB.Preload("Sender").
		Where("recipient_id = ? AND status = ?", user.ID, models.ConnectionStatusPending).
		Order("created_at DESC").
		Find(&connections).Error

	if err != nil {
		fmt.Printf("Error finding connection requests: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Crear respuesta con datos populares
	type ConnectionRequestResponse struct {
		ID        uint           `json:"_id"`
		Sender    models.UserDto `json:"sender"`
		Recipient uint           `json:"recipient"`
		Status    string         `json:"status"`
		CreatedAt string         `json:"createdAt"`
		UpdatedAt string         `json:"updatedAt"`
	}

	var response []ConnectionRequestResponse

	// Popular datos de cada sender
	for _, conn := range connections {
		response = append(response, ConnectionRequestResponse{
			ID: conn.ID,
			Sender: models.UserDto{
				ID:             conn.Sender.ID,
				Name:           conn.Sender.Name,
				Username:       conn.Sender.Username,
				ProfilePicture: conn.Sender.ProfilePicture,
				Headline:       conn.Sender.HeadLine,
			},
			Recipient: conn.RecipientID,
			Status:    string(conn.Status),
			CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetUserConnections returns all users connected to the authenticated user
func GetUserConnections(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar todas las conexiones aceptadas donde el usuario es sender o recipient
	var connections []models.Connection
	err := lib.DB.Preload("Sender").Preload("Recipient").
		Where("(sender_id = ? OR recipient_id = ?) AND status = ?",
			user.ID, user.ID, models.ConnectionStatusAccepted).
		Find(&connections).Error

	if err != nil {
		fmt.Printf("Error finding connections: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Si no tiene conexiones, devolver array vacío
	if len(connections) == 0 {
		return c.Status(fiber.StatusOK).JSON([]interface{}{})
	}

	// Extraer los usuarios conectados
	var connectedUsers []models.UserDto
	for _, conn := range connections {
		var connectedUser models.User
		if conn.SenderID == user.ID {
			connectedUser = conn.Recipient
		} else {
			connectedUser = conn.Sender
		}

		connectedUsers = append(connectedUsers, models.UserDto{
			ID:             connectedUser.ID,
			Name:           connectedUser.Name,
			Username:       connectedUser.Username,
			ProfilePicture: connectedUser.ProfilePicture,
			Headline:       connectedUser.HeadLine,
		})
	}

	return c.Status(fiber.StatusOK).JSON(connectedUsers)
}

// RemoveConnection removes a connection between the authenticated user and another user
func RemoveConnection(c *fiber.Ctx) error {
	// Obtener ID del usuario a desconectar desde los parámetros
	targetUserIDStr := c.Params("userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no sea el mismo usuario
	if user.ID == uint(targetUserID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "You cannot remove yourself as a connection",
		})
	}

	// Buscar la conexión entre los dos usuarios
	var connection models.Connection
	err = lib.DB.Where("(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		user.ID, uint(targetUserID), uint(targetUserID), user.ID).
		Where("status = ?", models.ConnectionStatusAccepted).
		First(&connection).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Connection does not exist",
			})
		}
		fmt.Printf("Error finding connection: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Eliminar la conexión
	if err := lib.DB.Delete(&connection).Error; err != nil {
		fmt.Printf("Error removing connection: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to remove connection",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Connection removed successfully",
	})
}

// GetConnectionStatus returns the connection status between the authenticated user and another user
func GetConnectionStatus(c *fiber.Ctx) error {
	// Obtener ID del usuario objetivo desde los parámetros
	targetUserIDStr := c.Params("userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no sea el mismo usuario
	if user.ID == uint(targetUserID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Cannot check connection status with yourself",
		})
	}

	// Verificar si ya están conectados
	var connectedConnection models.Connection
	err = lib.DB.Where("(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		user.ID, uint(targetUserID), uint(targetUserID), user.ID).
		Where("status = ?", models.ConnectionStatusAccepted).
		First(&connectedConnection).Error

	if err == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "connected",
		})
	}

	// Verificar si existe una solicitud pendiente
	var pendingRequest models.Connection
	err = lib.DB.Where("(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		user.ID, uint(targetUserID), uint(targetUserID), user.ID).
		Where("status = ?", models.ConnectionStatusPending).
		First(&pendingRequest).Error

	if err == nil {
		// Existe una solicitud pendiente
		if pendingRequest.SenderID == user.ID {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "pending",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":    "received",
				"requestId": pendingRequest.ID,
			})
		}
	} else if err != gorm.ErrRecordNotFound {
		// Error en la consulta
		fmt.Printf("Error checking pending connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// No están conectados y no hay solicitudes pendientes
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "not_connected",
	})
}
