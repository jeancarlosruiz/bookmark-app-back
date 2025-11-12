package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/config"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

func main() {
	// Define command line flags
	clearOnly := flag.Bool("clear", false, "Only clear the database without seeding")
	reset := flag.Bool("reset", false, "Clear and reseed the database")
	userID := flag.String("user", "", "User ID to associate bookmarks with (required)")

	flag.Parse()

	// Validate user ID
	if *userID == "" && !*clearOnly {
		fmt.Println("❌ Error: --user flag is required for seeding")
		fmt.Println("\nUsage:")
		fmt.Println("  go run cmd/seed/main.go --user=<user-id>")
		fmt.Println("  go run cmd/seed/main.go --user=<user-id> --reset")
		fmt.Println("  go run cmd/seed/main.go --clear")
		os.Exit(1)
	}

	// Load configuration
	if err := config.Load(); err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	if err := database.Connect(); err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Run migrations to ensure tables exist
	// Note: User table is NOT migrated as it exists in external auth schema (neon_auth.users_sync)
	fmt.Println("🔧 Running migrations...")
	database.DB.AutoMigrate(&models.Bookmarks{}, &models.Tag{}, &models.BookmarkTag{})

	// Execute based on flags
	var err error

	if *clearOnly {
		err = database.ClearDatabase()
	} else if *reset {
		err = database.ResetAndSeed(*userID)
	} else {
		err = database.SeedDatabase(*userID)
	}

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🎉 Done!")
}
