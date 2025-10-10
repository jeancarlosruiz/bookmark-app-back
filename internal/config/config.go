package config

import (
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	return godotenv.Load()
}

func GetPort() string {
	return os.Getenv("PORT")
}
