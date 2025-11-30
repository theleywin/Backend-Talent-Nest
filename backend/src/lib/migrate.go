package lib

import (
	"log"

	"github.com/theleywin/Backend-Talent-Nest/src/models"
)

// AutoMigrate runs all database migrations
func AutoMigrate() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Connection{},
		&models.Post{},
		&models.Comment{},
		&models.Like{},
		&models.Notification{},
	)

	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed!")
}
