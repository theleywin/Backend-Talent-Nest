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

// SendConnectionRequest sends a connection request from the authenticated user to another user
func SendConnectionRequest(c *fiber.Ctx) error {
	// Obtener ID del usuario destino desde los parámetros
	targetUserIDStr := c.Params("userId")
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no se envíe solicitud a uno mismo
	if user.Id == targetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "You can't send a connection request to yourself",
		})
	}

	// Validar que no estén ya conectados
	for _, conn := range user.Connections {
		if conn == targetUserID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "You are already connected with this user",
			})
		}
	}

	// Verificar si ya existe una solicitud pendiente
	connectionCollection := lib.DB.Collection("connections")
	filter := bson.M{
		"sender":    user.Id,
		"recipient": targetUserID,
		"status":    "pending",
	}

	var existingRequest models.Connection
	err = connectionCollection.FindOne(c.Context(), filter).Decode(&existingRequest)
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "A connection request already exists",
		})
	} else if err != mongo.ErrNoDocuments {
		fmt.Printf("Error checking existing connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Crear nueva solicitud de conexión
	newRequest := models.Connection{
		Id:        primitive.NewObjectID(),
		Sender:    user.Id,
		Recipient: targetUserID,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Guardar en la base de datos
	_, err = connectionCollection.InsertOne(c.Context(), newRequest)
	if err != nil {
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
	requestID, err := primitive.ObjectIDFromHex(requestIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar la solicitud de conexión
	connectionCollection := lib.DB.Collection("connections")
	var request models.Connection
	err = connectionCollection.FindOne(c.Context(), bson.M{"_id": requestID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
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
	if request.Recipient != user.Id {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Not authorized to accept this request",
		})
	}

	// Verificar que la solicitud esté pendiente
	if request.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "This request has already been processed",
		})
	}

	// Ejecutar operaciones sin transacción (para desarrollo)
	// 1. Actualizar el estado de la solicitud a "accepted"
	update := bson.M{
		"$set": bson.M{
			"status":    "accepted",
			"updatedAt": time.Now(),
		},
	}
	_, err = connectionCollection.UpdateOne(c.Context(), bson.M{"_id": requestID}, update)
	if err != nil {
		fmt.Printf("Error updating connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to accept connection request",
		})
	}

	// 2. Agregar conexión al usuario remitente
	usersCollection := lib.DB.Collection("users")
	_, err = usersCollection.UpdateOne(
		c.Context(),
		bson.M{"_id": request.Sender},
		bson.M{"$addToSet": bson.M{"connections": user.Id}},
	)
	if err != nil {
		fmt.Printf("Error updating sender connections: %v\n", err)
		// Intentar revertir el estado de la solicitud
		connectionCollection.UpdateOne(c.Context(), bson.M{"_id": requestID}, bson.M{"$set": bson.M{"status": "pending"}})
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update connections",
		})
	}

	// 3. Agregar conexión al usuario actual (destinatario)
	_, err = usersCollection.UpdateOne(
		c.Context(),
		bson.M{"_id": user.Id},
		bson.M{"$addToSet": bson.M{"connections": request.Sender}},
	)
	if err != nil {
		fmt.Printf("Error updating recipient connections: %v\n", err)
		// Revertir cambios
		usersCollection.UpdateOne(c.Context(), bson.M{"_id": request.Sender}, bson.M{"$pull": bson.M{"connections": user.Id}})
		connectionCollection.UpdateOne(c.Context(), bson.M{"_id": requestID}, bson.M{"$set": bson.M{"status": "pending"}})
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update connections",
		})
	}

	// 4. Crear notificación para el usuario remitente
	notification := models.Notification{
		Id:          primitive.NewObjectID(),
		Recipient:   request.Sender,
		Type:        "connectionAccepted",
		RelatedUser: user.Id,
		Read:        false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	notificationsCollection := lib.DB.Collection("notifications")
	_, err = notificationsCollection.InsertOne(c.Context(), notification)
	if err != nil {
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
	requestID, err := primitive.ObjectIDFromHex(requestIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar la solicitud de conexión
	connectionCollection := lib.DB.Collection("connections")
	var request models.Connection
	err = connectionCollection.FindOne(c.Context(), bson.M{"_id": requestID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
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
	if request.Recipient != user.Id {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Not authorized to reject this request",
		})
	}

	// Verificar que la solicitud esté pendiente
	if request.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "This request has already been processed",
		})
	}

	// Actualizar el estado de la solicitud a "rejected"
	update := bson.M{
		"$set": bson.M{
			"status":    "rejected",
			"updatedAt": time.Now(),
		},
	}

	result, err := connectionCollection.UpdateOne(c.Context(), bson.M{"_id": requestID}, update)
	if err != nil {
		fmt.Printf("Error rejecting connection request: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to reject connection request",
		})
	}

	if result.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Connection request not found",
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

	// Buscar solicitudes pendientes
	collection := lib.DB.Collection("connections")
	filter := bson.M{
		"recipient": user.Id,
		"status":    "pending",
	}
	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(c.Context(), filter, opts)
	if err != nil {
		fmt.Printf("Error finding connection requests: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}
	defer cursor.Close(c.Context())

	var connections []models.Connection
	if err := cursor.All(c.Context(), &connections); err != nil {
		fmt.Printf("Error decoding connections: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Crear respuesta con datos populares
	type ConnectionRequestResponse struct {
		ID        primitive.ObjectID `json:"_id"`
		Sender    models.User        `json:"sender"`
		Recipient primitive.ObjectID `json:"recipient"`
		Status    string             `json:"status"`
		CreatedAt time.Time          `json:"createdAt"`
		UpdatedAt time.Time          `json:"updatedAt"`
	}

	var response []ConnectionRequestResponse

	// Popular datos de cada sender
	usersCollection := lib.DB.Collection("users")
	for _, conn := range connections {
		var sender models.User
		err := usersCollection.FindOne(
			c.Context(),
			bson.M{"_id": conn.Sender},
			options.FindOne().SetProjection(bson.M{
				"name":            1,
				"username":        1,
				"profile_picture": 1,
				"headline":        1,
				"connections":     1,
			}),
		).Decode(&sender)

		if err != nil && err != mongo.ErrNoDocuments {
			fmt.Printf("Error finding sender user: %v\n", err)
			continue
		}

		response = append(response, ConnectionRequestResponse{
			ID:        conn.Id,
			Sender:    sender,
			Recipient: conn.Recipient,
			Status:    string(conn.Status),
			CreatedAt: conn.CreatedAt,
			UpdatedAt: conn.UpdatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetUserConnections returns all users connected to the authenticated user
func GetUserConnections(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar el usuario actual con solo el campo connections
	usersCollection := lib.DB.Collection("users")
	var currentUser models.User
	err := usersCollection.FindOne(
		c.Context(),
		bson.M{"_id": user.Id},
		options.FindOne().SetProjection(bson.M{
			"connections": 1,
		}),
	).Decode(&currentUser)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusOK).JSON([]interface{}{})
		}
		fmt.Printf("Error finding user: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Si no tiene conexiones, devolver array vacío
	if len(currentUser.Connections) == 0 {
		return c.Status(fiber.StatusOK).JSON([]interface{}{})
	}

	// Buscar los usuarios conectados con los campos necesarios
	filter := bson.M{
		"_id": bson.M{"$in": currentUser.Connections},
	}
	opts := options.Find().SetProjection(bson.M{
		"name":            1,
		"username":        1,
		"profile_picture": 1,
		"headline":        1,
		"connections":     1,
	})

	cursor, err := usersCollection.Find(c.Context(), filter, opts)
	if err != nil {
		fmt.Printf("Error finding connected users: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}
	defer cursor.Close(c.Context())

	var connections []models.User
	if err := cursor.All(c.Context(), &connections); err != nil {
		fmt.Printf("Error decoding connections: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(connections)
}

// RemoveConnection removes a connection between the authenticated user and another user
func RemoveConnection(c *fiber.Ctx) error {
	// Obtener ID del usuario a desconectar desde los parámetros
	targetUserIDStr := c.Params("userId")
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no sea el mismo usuario
	if user.Id == targetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "You cannot remove yourself as a connection",
		})
	}

	// Verificar que exista la conexión antes de eliminarla
	usersCollection := lib.DB.Collection("users")

	var currentUser models.User
	err = usersCollection.FindOne(
		c.Context(),
		bson.M{"_id": user.Id},
		options.FindOne().SetProjection(bson.M{"connections": 1}),
	).Decode(&currentUser)

	if err != nil {
		fmt.Printf("Error finding current user: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Server error",
		})
	}

	// Verificar que exista la conexión
	connectionExists := false
	for _, conn := range currentUser.Connections {
		if conn == targetUserID {
			connectionExists = true
			break
		}
	}

	if !connectionExists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Connection does not exist",
		})
	}

	// Eliminar la conexión de ambos usuarios
	_, err = usersCollection.UpdateOne(
		c.Context(),
		bson.M{"_id": user.Id},
		bson.M{"$pull": bson.M{"connections": targetUserID}},
	)
	if err != nil {
		fmt.Printf("Error removing connection from current user: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to remove connection",
		})
	}

	_, err = usersCollection.UpdateOne(
		c.Context(),
		bson.M{"_id": targetUserID},
		bson.M{"$pull": bson.M{"connections": user.Id}},
	)
	if err != nil {
		// Si falla la segunda actualización, intentar revertir la primera
		fmt.Printf("Error removing connection from target user: %v\n", err)

		// Revertir la primera operación
		usersCollection.UpdateOne(
			c.Context(),
			bson.M{"_id": user.Id},
			bson.M{"$addToSet": bson.M{"connections": targetUserID}},
		)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to remove connection completely",
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
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Validar que no sea el mismo usuario
	if user.Id == targetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Cannot check connection status with yourself",
		})
	}

	// Verificar si ya están conectados
	for _, conn := range user.Connections {
		if conn == targetUserID {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "connected",
			})
		}
	}

	// Verificar si existe una solicitud pendiente
	connectionCollection := lib.DB.Collection("connections")
	filter := bson.M{
		"$or": []bson.M{
			{"sender": user.Id, "recipient": targetUserID},
			{"sender": targetUserID, "recipient": user.Id},
		},
		"status": "pending",
	}

	var pendingRequest models.Connection
	err = connectionCollection.FindOne(c.Context(), filter).Decode(&pendingRequest)

	if err == nil {
		// Existe una solicitud pendiente
		if pendingRequest.Sender == user.Id {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "pending",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":    "received",
				"requestId": pendingRequest.Id,
			})
		}
	} else if err != mongo.ErrNoDocuments {
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
