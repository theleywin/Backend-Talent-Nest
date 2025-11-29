package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
)

// ProtectRoute is a middleware that checks for a valid JWT token, authenticates the user, and attaches user data to the request context
func ProtectRoute(c *fiber.Ctx) error {

	// Obtener token del header Authorization
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token no proporcionado",
		})
	}

	// Extraer el token (formato esperado: "Bearer <token>")
	var token string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Formato de token inválido",
		})
	}

	decoded, err := lib.VerifyJWT(token)
	if err != nil || decoded == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token inválido",
		})
	}

	// El userID ahora es float64 porque viene del JWT
	userIDFloat, ok := decoded["userId"].(float64)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "No autorizado - Token inválido",
		})
	}

	userID := uint(userIDFloat)

	var user models.User
	err = lib.DB.First(&user, userID).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Usuario no encontrado",
		})
	}

	// Poblar conexiones
	user.Connections = user.GetConnections(lib.DB)

	user.Password = ""

	c.Locals("user", user)

	return c.Next()
}
