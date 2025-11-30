package lib

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
)

// Returns a map with a message key for API responses
func MessageResponse(message string) fiber.Map {
	return fiber.Map{
		"message": message,
	}
}

// Generates a JWT token for the given user ID
func GenerateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback-secret-key"
	}

	return token.SignedString([]byte(secret))
}

// Verifies and decodes a JWT token, returning its claims
func VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "fallback-secret-key"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Método de firma inválido")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fiber.NewError(fiber.StatusUnauthorized, "Token inválido")
}

// Searches for a user by ID and excludes the password from the result
func FindUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := DB.Select("id", "name", "username", "email", "profile_picture", "cover_picture", "headline", "about", "location").
		First(&user, userID).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
