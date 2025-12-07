package models

import (
	"gorm.io/gorm"
)

type Connection struct {
	gorm.Model
	SenderID    uint             `json:"sender" gorm:"index"`
	RecipientID uint             `json:"recipient" gorm:"index"`
	Status      string           `json:"status"`
	Sender      User             `json:"-" gorm:"foreignKey:SenderID"`
	Recipient   User             `json:"-" gorm:"foreignKey:RecipientID"`
}

const (
	ConnectionStatusPending = "pending"
	ConnectionStatusAccepted = "accepted"
	ConnectionStatusRejected = "rejected"
)
