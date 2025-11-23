package controllers

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Signup handles user registration, validates input, checks for duplicates, hashes password, creates user, and sets JWT cookie
func Signup(c *fiber.Ctx) error {

	var userData struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Datos inválidos",
		})
	}

	if userData.Name == "" || userData.Username == "" || userData.Email == "" || userData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Todos los campos son requeridos",
		})
	}

	if len(userData.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "La contraseña debe tener al menos 6 caracteres",
		})
	}

	userCollection := lib.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var existingUser models.User
	err := userCollection.FindOne(ctx, bson.M{"email": userData.Email}).Decode(&existingUser)
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "El email ya existe",
		})
	}

	err = userCollection.FindOne(ctx, bson.M{"username": userData.Username}).Decode(&existingUser)
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "El username ya existe",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 11)
	if err != nil {
		log.Printf("Error al encriptar contraseña: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error interno del servidor",
		})
	}

	newUser := models.User{
		Name:           userData.Name,
		Username:       userData.Username,
		Email:          userData.Email,
		Password:       string(hashedPassword),
		ProfilePicture: "", // Valor por defecto
		CoverPicture:   "", // Valor por defecto
		HeadLine:       "", // Valor por defecto
		About:          "", // Valor por defecto
		Location:       "", // Valor por defecto
		Skills:         []string{},
		Experience:     []models.Experience{},
		Education:      []models.Education{},
		Connections:    []primitive.ObjectID{},
	}

	result, err := userCollection.InsertOne(ctx, newUser)
	if err != nil {
		log.Printf("Error al insertar usuario: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error al crear usuario",
		})
	}

	token, err := lib.GenerateJWT(result.InsertedID)
	if err != nil {
		log.Printf("Error al generar token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error al generar token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Usuario registrado exitosamente",
		"token":   token,
	})
}

// Login authenticates a user by username and password, generates JWT, and sets cookie
func Login(c *fiber.Ctx) error {

	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&loginData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Datos inválidos",
		})
	}

	if loginData.Username == "" || loginData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Usuario y contraseña son requeridos",
		})
	}

	userCollection := lib.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"username": loginData.Username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Credenciales inválidas",
			})
		}

		log.Printf("Error al buscar usuario: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error del servidor",
		})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password))
	if err != nil {

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Credenciales inválidas",
		})
	}

	token, err := lib.GenerateJWT(user.Id.Hex())
	if err != nil {
		log.Printf("Error al generar token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error del servidor",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Inicio de sesión exitoso",
		"token":   token,
	})
}

// GetCurrentUser returns the currently authenticated user's data
func GetCurrentUser(c *fiber.Ctx) error {

	user := c.Locals("user")
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Usuario no autenticado",
		})
	}
	return c.JSON(user)
}

// Logout clears the authentication cookie to log out the user
func Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt-talentnest",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Strict", // Usa "Lax" si tienes problemas en local
		Secure:   false,    // true en producción con HTTPS
		Path:     "/",      // Debe coincidir con el path original
	})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

