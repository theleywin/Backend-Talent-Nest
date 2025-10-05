package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	Id          primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Recipient   primitive.ObjectID `json:"recipient" bson:"recipient"`
	Type        NotificationType   `json:"type" bson:"type"`
	RelatedUser primitive.ObjectID `json:"related_user,omitempty" bson:"related_user,omitempty"`
	RelatedPost primitive.ObjectID `json:"related_post,omitempty" bson:"related_post,omitempty"`
	Read        bool               `json:"read" bson:"read"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type NotificationType string

const (
	NotificationTypeLike               NotificationType = "like"
	NotificationTypeComment            NotificationType = "comment"
	NotificationTypeConnectionAccepted NotificationType = "connectionAccepted"
)
