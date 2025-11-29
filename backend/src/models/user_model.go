package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name           string                   `json:"name"`
	Username       string                   `json:"username" gorm:"uniqueIndex"`
	Email          string                   `json:"email" gorm:"uniqueIndex"`
	Password       string                   `json:"password"`
	ProfilePicture string                   `json:"profile_picture"`
	CoverPicture   string                   `json:"cover_picture"`
	HeadLine       string                   `json:"headline"`
	About          string                   `json:"about"`
	Location       string                   `json:"location"`
	Skills         []string                 `json:"skills" gorm:"serializer:json"`
	Experience     []map[string]interface{} `json:"experience" gorm:"serializer:json"`
	Education      []map[string]interface{} `json:"education" gorm:"serializer:json"`
	Connections    []uint                   `json:"connections" gorm:"-"` // No se guarda en DB, se llena dinámicamente
}

// MarshalJSON personaliza la serialización para cambiar ID a _id
func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		ID uint `json:"_id"`
		*Alias
	}{
		ID:    u.ID,
		Alias: (*Alias)(&u),
	})
}

// GetConnections obtiene los IDs de todos los usuarios conectados
func (u *User) GetConnections(db *gorm.DB) []uint {
	var connections []Connection
	db.Where("(sender_id = ? OR recipient_id = ?) AND status = ?",
		u.ID, u.ID, ConnectionStatusAccepted).
		Find(&connections)

	connectionIDs := make([]uint, 0, len(connections))
	for _, conn := range connections {
		if conn.SenderID == u.ID {
			connectionIDs = append(connectionIDs, conn.RecipientID)
		} else {
			connectionIDs = append(connectionIDs, conn.SenderID)
		}
	}
	return connectionIDs
}

type UserDto struct {
	ID             uint   `json:"_id"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	ProfilePicture string `json:"profilePicture"`
	Headline       string `json:"headline,omitempty"`
}

type Experience struct {
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Description string    `json:"description"`
}

type Education struct {
	School string `json:"school"`
	Degree string `json:"degree"`
	From   int    `json:"from"`
	To     int    `json:"to"`
}
