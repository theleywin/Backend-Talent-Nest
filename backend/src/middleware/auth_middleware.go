package middleware

import (
    "fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProtectRoute is a middleware that checks for a valid JWT token, authenticates the user, and attaches user data to the request context
func ProtectRoute(c *fiber.Ctx) error {
	token := c.Cookies("jwt-talentnest")
	fmt.Println(token)
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token no proporcionado",
		})
	}

	decoded, err := lib.VerifyJWT(token)
	if err != nil || decoded == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token inválido",
		})
	}

	userID, ok := decoded["userId"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token inválido",
		})
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "ID de usuario inválido",
		})
	}

	userCollection := lib.DB.Collection("users")
	var user models.User
	err = userCollection.FindOne(c.Context(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Usuario no encontrado",
		})
	}

	user.Password = ""

	c.Locals("user", user)

	return c.Next()
}
