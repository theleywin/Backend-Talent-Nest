package controllers

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"gorm.io/gorm"
)

// GetSuggestedConnections returns a list of suggested users for the current user to connect with
func GetSuggestedConnections(c *fiber.Ctx) error {
	var user models.User = c.Locals("user").(models.User)

	// Obtener IDs de usuarios ya conectados
	var connections []models.Connection
	err := lib.DB.Where("(sender_id = ? OR recipient_id = ?) AND status = ?",
		user.ID, user.ID, models.ConnectionStatusAccepted).
		Find(&connections).Error

	if err != nil {
		log.Printf("Error al buscar conexiones: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error interno del servidor",
		})
	}

	// Crear lista de IDs a excluir (usuario actual + conexiones)
	excludeIDs := []uint{user.ID}
	for _, conn := range connections {
		if conn.SenderID == user.ID {
			excludeIDs = append(excludeIDs, conn.RecipientID)
		} else {
			excludeIDs = append(excludeIDs, conn.SenderID)
		}
	}

	// Buscar usuarios sugeridos
	var suggestedUsers []models.User
	err = lib.DB.Select("id", "name", "username", "profile_picture", "head_line").
		Where("id NOT IN ?", excludeIDs).
		Limit(3).
		Find(&suggestedUsers).Error

	if err != nil {
		log.Printf("Error en la consulta: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error al buscar usuarios sugeridos",
		})
	}

	// Convertir a formato de respuesta
	var response []models.UserDto
	for _, u := range suggestedUsers {
		response = append(response, models.UserDto{
			ID:             u.ID,
			Name:           u.Name,
			Username:       u.Username,
			ProfilePicture: u.ProfilePicture,
			Headline:       u.HeadLine,
		})
	}

	return c.JSON(response)
}

// GetPublicProfile returns the public profile of a user by username
func GetPublicProfile(c *fiber.Ctx) error {

	username := c.Params("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username es requerido",
		})
	}

	var user models.User
	err := lib.DB.Select("id", "name", "username", "email", "profile_picture", "cover_picture",
		"head_line", "about", "location", "skills", "experience", "education",
		"created_at", "updated_at").
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Usuario no encontrado",
			})
		}

		log.Printf("Error en GetPublicProfile controller: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error del servidor",
		})
	}

	// Poblar conexiones
	user.Connections = user.GetConnections(lib.DB)

	return c.JSON(user)
}

func UpdateProfile(c *fiber.Ctx) error {

	var user models.User = c.Locals("user").(models.User)

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}

	// Cargar el usuario actual de la base de datos
	var currentUser models.User
	if err := lib.DB.First(&currentUser, user.ID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	// Actualizar campos permitidos
	if name, ok := body["name"].(string); ok {
		currentUser.Name = name
	}
	if username, ok := body["username"].(string); ok {
		currentUser.Username = username
	}
	if headline, ok := body["headline"].(string); ok {
		currentUser.HeadLine = headline
	}
	if about, ok := body["about"].(string); ok {
		currentUser.About = about
	}
	if location, ok := body["location"].(string); ok {
		currentUser.Location = location
	}
	if profilePicture, ok := body["profilePicture"].(string); ok {
		currentUser.ProfilePicture = profilePicture
	}
	if bannerImg, ok := body["bannerImg"].(string); ok {
		currentUser.CoverPicture = bannerImg
	}

	// Manejar skills (array de strings)
	if skills, ok := body["skills"].([]interface{}); ok {
		skillsStr := make([]string, 0, len(skills))
		for _, s := range skills {
			if str, ok := s.(string); ok {
				skillsStr = append(skillsStr, str)
			}
		}
		currentUser.Skills = skillsStr
	}

	// Manejar experience (array de objetos)
	if experience, ok := body["experience"].([]interface{}); ok {
		expArr := make([]map[string]interface{}, 0, len(experience))
		for _, exp := range experience {
			if expMap, ok := exp.(map[string]interface{}); ok {
				expArr = append(expArr, expMap)
			}
		}
		currentUser.Experience = expArr
	}

	// Manejar education (array de objetos)
	if education, ok := body["education"].([]interface{}); ok {
		eduArr := make([]map[string]interface{}, 0, len(education))
		for _, edu := range education {
			if eduMap, ok := edu.(map[string]interface{}); ok {
				eduArr = append(eduArr, eduMap)
			}
		}
		currentUser.Education = eduArr
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

	// Guardar los cambios
	if err := lib.DB.Save(&currentUser).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "username") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El nombre de usuario ya está en uso",
			})
		}

		fmt.Printf("Error al actualizar el usuario: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al actualizar el usuario",
		})
	}

	// Poblar conexiones
	currentUser.Connections = currentUser.GetConnections(lib.DB)

	// Limpiar password antes de devolver
	currentUser.Password = ""

	return c.JSON(currentUser)
}

func SearchUsers(c *fiber.Ctx) error {
	query := c.Query("query")

	if query == "" {
		return c.JSON([]models.UserDto{})
	}

	// Búsqueda case-insensitive en SQLite usando LIKE
	searchPattern := "%" + query + "%"

	var users []models.User
	err := lib.DB.Select("id", "name", "username", "profile_picture", "head_line").
		Where("name LIKE ? OR username LIKE ?", searchPattern, searchPattern).
		Limit(10).
		Find(&users).Error

	if err != nil {
		log.Printf("Error searching users: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Error al buscar usuarios",
		})
	}

	// Convertir a UserDto
	var response []models.UserDto
	for _, u := range users {
		response = append(response, models.UserDto{
			ID:             u.ID,
			Name:           u.Name,
			Username:       u.Username,
			ProfilePicture: u.ProfilePicture,
			Headline:       u.HeadLine,
		})
	}

	if response == nil {
		response = []models.UserDto{}
	}

	return c.JSON(response)
}
