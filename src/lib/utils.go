package lib

import "github.com/gofiber/fiber/v2"

func MessageResponse(message string) fiber.Map {
	return fiber.Map{
		"message": message,
	}
}
