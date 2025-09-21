package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Id             primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name           string               `json:"name" bson:"name"`
	Username       string               `json:"username" bson:"username"`
	Email          string               `json:"email" bson:"email"`
	Password       string               `json:"password" bson:"password"`
	ProfilePicture string               `json:"profile_picture" bson:"profile_picture"`
	CoverPicture   string               `json:"cover_picture" bson:"cover_picture"`
	HeadLine       string               `json:"headline" bson:"headline"`
	About          string               `json:"about" bson:"about"`
	Location       string               `json:"location" bson:"location"`
	Skills         []string             `json:"skills" bson:"skills"`
	Experience     []Experience         `json:"experience" bson:"experience"`
	Education      []Education          `json:"education" bson:"education"`
	Connections    []primitive.ObjectID `json:"connections" bson:"connections"`
}

type UserDto struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	Name           string             `bson:"name" json:"name"`
	Username       string             `bson:"username" json:"username"`
	ProfilePicture string             `bson:"profilePicture" json:"profilePicture"`
	Headline       string             `bson:"headline" json:"headline,omitempty"`
}

type Experience struct {
	Title       string    `json:"title" bson:"title"`
	Company     string    `json:"company" bson:"company"`
	From        time.Time `json:"from" bson:"from"`
	To          time.Time `json:"to" bson:"to"`
	Description string    `json:"description" bson:"description"`
}

type Education struct {
	School string `json:"school" bson:"school"`
	Degree string `json:"degree" bson:"degree"`
	From   int    `json:"from" bson:"from"`
	To     int    `json:"to" bson:"to"`
}
