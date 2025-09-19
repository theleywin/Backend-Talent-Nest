package controllers

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetSuggestedConnections returns a list of suggested users for the current user to connect with
func GetSuggestedConnections(c *fiber.Ctx) error {
	var user models.User = c.Locals("user").(models.User)
	userID := user.Id

	var currentUser models.User
	userCollection := lib.DB.Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := userCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&currentUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"message": "Usuario no encontrado",
			})
		}
		log.Printf("Error al buscar usuario: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error interno del servidor",
		})
	}

	filter := bson.M{
		"_id": bson.M{
			"$ne":  userID,
			"$nin": currentUser.Connections,
		},
	}

	findOptions := options.Find()
	findOptions.SetLimit(3)
	findOptions.SetProjection(bson.M{
		"name":            1,
		"username":        1,
		"profile_picture": 1,
		"headline":        1,
	})

	cursor, err := userCollection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error en la consulta: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error al buscar usuarios sugeridos",
		})
	}
	defer cursor.Close(ctx)

	var suggestedUsers []models.User
	if err = cursor.All(ctx, &suggestedUsers); err != nil {
		log.Printf("Error al decodificar resultados: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error al procesar resultados",
		})
	}

	return c.JSON(suggestedUsers)
}

// GetPublicProfile returns the public profile of a user by username
func GetPublicProfile(c *fiber.Ctx) error {

	username := c.Params("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username es requerido",
		})
	}

	userCollection := lib.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.FindOne().SetProjection(bson.M{"password": 0})

	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"username": username}, opts).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Usuario no encontrado",
			})
		}

		log.Printf("Error en GetPublicProfile controller: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error del servidor",
		})
	}

	return c.JSON(user)
}

// UpdateProfile updates the authenticated user's profile with allowed fields
func UpdateProfile(c *fiber.Ctx) error {

	userCollection := lib.DB.Collection("users")

	var user models.User = c.Locals("user").(models.User)
	objID := user.Id

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allowedFields := []string{
		"name",
		"username",
		"headline",
		"about",
		"location",
		"profilePicture",
		"bannerImg",
		"skills",
		"experience",
		"education",
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}

	updatedData := bson.M{}

	for _, field := range allowedFields {
		if value, exists := body[field]; exists && value != nil {

			switch field {
			case "skills":
				if skills, ok := value.([]interface{}); ok {
					updatedData[field] = skills
				}
			case "experience", "education":
				if slice, ok := value.([]interface{}); ok {
					updatedData[field] = slice
				}
			default:
				updatedData[field] = value
			}
		}
	}

	// TODO
	// Procesar imágenes si están presentes
	// if profilePicture, exists := body["profilePicture"]; exists && profilePicture != nil {
	// 	if imgStr, ok := profilePicture.(string); ok && imgStr != "" {
	// 		// Subir imagen a Cloudinary
	// 		uploadResult, err := h.cld.Upload.Upload(ctx, imgStr, uploader.UploadParams{})
	// 		if err != nil {
	// 			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 				"error": "Error al subir la imagen de perfil",
	// 			})
	// 		}
	// 		updatedData["profilePicture"] = uploadResult.SecureURL
	// 	}
	// }

	// if bannerImg, exists := body["bannerImg"]; exists && bannerImg != nil {
	// 	if imgStr, ok := bannerImg.(string); ok && imgStr != "" {
	// 		// Subir imagen a Cloudinary
	// 		uploadResult, err := h.cld.Upload.Upload(ctx, imgStr, uploader.UploadParams{})
	// 		if err != nil {
	// 			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 				"error": "Error al subir la imagen de banner",
	// 			})
	// 		}
	// 		updatedData["bannerImg"] = uploadResult.SecureURL
	// 	}
	// }

	updatedData["updatedAt"] = time.Now()

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updatedData}

	var updatedUser models.User
	err := userCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}

		if strings.Contains(err.Error(), "duplicate key error") && strings.Contains(err.Error(), "username") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El nombre de usuario ya está en uso",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al actualizar el usuario",
		})
	}

	updatedUser.Password = ""

	return c.JSON(updatedUser)
}
