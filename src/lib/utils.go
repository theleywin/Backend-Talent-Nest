package lib

import (
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Returns a map with a message key for API responses
func MessageResponse(message string) fiber.Map {
	return fiber.Map{
		"message": message,
	}
}

// Generates a JWT token for the given user ID
func GenerateJWT(userID interface{}) (string, error) {
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
func FindUserByID(userID string) (*models.User, error) {
	userCollection := DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = userCollection.FindOne(ctx, bson.M{
		"_id": objectID,
	}, options.FindOne().SetProjection(bson.M{"password": 0})).Decode(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
