package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	Id        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Author    primitive.ObjectID   `json:"author" bson:"author"`
	Content   string               `json:"content" bson:"content"`
	Image     string               `json:"image" bson:"image"`
	Likes     []primitive.ObjectID `json:"likes" bson:"likes"`
	Comments  []Comment            `json:"comments" bson:"comments"`
	CreatedAt time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type PostDto struct {
	ID        primitive.ObjectID `json:"id"`
	Author    UserDto            `json:"author"`
	Content   string             `json:"content,omitempty"`
	Image     string             `json:"image,omitempty"`
	Likes     []UserDto          `json:"likes,omitempty"`
	Comments  []CommentDto       `json:"comments,omitempty"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

type Comment struct {
	Id        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Content   string             `json:"content" bson:"content"`
	User      primitive.ObjectID `json:"user" bson:"user"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

type CommentDto struct {
	ID        primitive.ObjectID `json:"id"`
	Content   string             `json:"content"`
	User      UserDto            `json:"user"`
	CreatedAt time.Time          `json:"createdAt"`
}
