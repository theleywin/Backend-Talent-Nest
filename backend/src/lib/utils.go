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

func PopulatePosts(c *fiber.Ctx, posts []models.Post) ([]models.PostDto, error) {
	userCollection := DB.Collection("users")
	var populatedPosts []models.PostDto

	for _, post := range posts {
		// Populate author
		var author models.UserDto
		err := userCollection.FindOne(c.Context(), bson.M{"_id": post.Author}).Decode(&author)
		if err != nil {
			continue // O manejar el error según necesites
		}

		// Populate comments users
		var populatedComments []models.CommentDto
		for _, comment := range post.Comments {
			var commentUser models.UserDto
			err := userCollection.FindOne(c.Context(), bson.M{"_id": comment.User}).Decode(&commentUser)
			if err != nil {
				continue
			}

			populatedComment := models.CommentDto{
				ID:        comment.Id,
				Content:   comment.Content,
				User:      commentUser,
				CreatedAt: comment.CreatedAt,
			}
			populatedComments = append(populatedComments, populatedComment)
		}

		// Populate likes users (si necesitas información de los usuarios que dieron like)
		var likedUsers []models.UserDto
		for _, likeID := range post.Likes {
			var likeUser models.UserDto
			err := userCollection.FindOne(c.Context(), bson.M{"_id": likeID}).Decode(&likeUser)
			if err != nil {
				continue
			}
			likedUsers = append(likedUsers, likeUser)
		}

		populatedPost := models.PostDto{
			ID:        post.Id,
			Author:    author,
			Content:   post.Content,
			Image:     post.Image,
			Likes:     likedUsers, // o mantener como []primitive.ObjectID si prefieres
			Comments:  populatedComments,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
		}

		populatedPosts = append(populatedPosts, populatedPost)
	}

	return populatedPosts, nil
}
