package lib

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB initializes the SQLite connection and sets the global DB variable
func ConnectDB() {
	var dbPath string = os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./talentnest.db"
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	log.Println("Connected to SQLite!")
}
