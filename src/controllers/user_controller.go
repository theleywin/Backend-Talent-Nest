package controllers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
)

func TestGetUsers(c *fiber.Ctx) error {
	var users []models.User

	coll := lib.DB.Collection("users")
	results, err := coll.Find(context.TODO(), bson.M{})

	if err != nil {
		panic(err)
	}

	for results.Next(context.TODO()) {
		var user models.User
		results.Decode(&user)
		users = append(users, user)
	}

	return c.JSON(&fiber.Map{
		"data": users,
	})
}

func TestCreateUser(c *fiber.Ctx) error {
	var user models.User
	c.BodyParser(&user)
	coll := lib.DB.Collection("users")
	coll.InsertOne(context.TODO(), bson.D{{
		Key:   "name",
		Value: user.Name,
	}})

	return c.JSON(&fiber.Map{
		"data": "guardando usuario",
	})
}
