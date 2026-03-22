package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres initializes and returns a GORM database connection
func ConnectPostgres(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to database via GORM")
	return db, nil
}
