package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Connection struct {
	Id        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Sender    primitive.ObjectID `json:"sender" bson:"sender"`
	Recipient primitive.ObjectID `json:"recipient" bson:"recipient"`
	Status    ConnectionStatus   `json:"status" bson:"status"` // pending, accepted, rejected
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type ConnectionStatus string

const (
	ConnectionStatusPending  ConnectionStatus = "pending"
	ConnectionStatusAccepted ConnectionStatus = "accepted"
	ConnectionStatusRejected ConnectionStatus = "rejected"
)
