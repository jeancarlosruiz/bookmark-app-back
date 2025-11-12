package database

import (
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})

	if err != nil {
		return err
	}

	// Set search path to include both public and neon_auth schemas
	// This allows PostgreSQL to resolve cross-schema foreign key constraints
	if err := db.Exec("SET search_path TO public, neon_auth").Error; err != nil {
		return err
	}

	DB = db
	return nil
}
