package models

import (
	"gorm.io/gorm"
)

type Connection struct {
	gorm.Model
	SenderID    uint             `json:"sender" gorm:"index"`
	RecipientID uint             `json:"recipient" gorm:"index"`
	Status      ConnectionStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Sender      User             `json:"-" gorm:"foreignKey:SenderID"`
	Recipient   User             `json:"-" gorm:"foreignKey:RecipientID"`
}

type ConnectionStatus string

const (
	ConnectionStatusPending  ConnectionStatus = "pending"
	ConnectionStatusAccepted ConnectionStatus = "accepted"
	ConnectionStatusRejected ConnectionStatus = "rejected"
)
