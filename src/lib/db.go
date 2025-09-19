package lib

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

// ConnectDB initializes the MongoDB connection and sets the global DB variable
func ConnectDB() {

	var db_url string = os.Getenv("MONGO_URI")
	if db_url == "" {
		db_url = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(db_url))
	if err != nil {
		panic(err)
	}

	var db_name string = os.Getenv("DB_NAME")
	if db_name == "" {
		db_name = "databaseName"
	}

	DB = client.Database(db_name)
	log.Println("Connected to MongoDB!")
}
