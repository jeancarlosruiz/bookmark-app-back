package database

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

type BookmarkSeed struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Favicon     string     `json:"favicon"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags"`
	Pinned      bool       `json:"pinned"`
	IsArchived  bool       `json:"isArchived"`
	VisitCount  int        `json:"visitCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastVisited *time.Time `json:"lastVisited"`
}

type SeedData struct {
	Bookmarks []BookmarkSeed `json:"bookmarks"`
}

// SeedDatabase seeds the database with test data from data.json
func SeedDatabase(userID string) error {
	fmt.Println("🌱 Starting database seeding...")

	// Validate that the user exists
	fmt.Printf("🔍 Validating user ID: %s\n", userID)
	var user models.User
	result := DB.First(&user, "id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("❌ User with ID '%s' not found in neon_auth.users_sync.\n   Please create a user first or use a valid user ID.\n   Error: %w", userID, result.Error)
	}
	fmt.Printf("✓ Validated user: %s (%s)\n\n", user.Name, user.Email)

	// Read the JSON file
	file, err := os.ReadFile("data.json")
	if err != nil {
		return fmt.Errorf("failed to read data.json: %w", err)
	}

	var seedData SeedData
	if err := json.Unmarshal(file, &seedData); err != nil {
		return fmt.Errorf("failed to parse data.json: %w", err)
	}

	// Create a map to track created tags
	tagMap := make(map[string]*models.Tag)

	fmt.Printf("📚 Seeding %d bookmarks...\n", len(seedData.Bookmarks))

	for i, bookmarkSeed := range seedData.Bookmarks {
		// Create or find tags
		var tags []models.Tag
		for _, tagName := range bookmarkSeed.Tags {
			// Check if tag already exists in our map
			if existingTag, exists := tagMap[tagName]; exists {
				tags = append(tags, *existingTag)
			} else {
				// Check if tag exists in database for this user
				var tag models.Tag
				result := DB.Where("title = ? AND user_id = ?", tagName, userID).First(&tag)

				if result.Error != nil {
					// Create new tag
					tag = models.Tag{
						Title:  tagName,
						UserID: userID,
					}
					if err := DB.Create(&tag).Error; err != nil {
						fmt.Printf("⚠️  Warning: Failed to create tag '%s': %v\n", tagName, err)
						continue
					}
					fmt.Printf("   ✓ Created tag: %s\n", tagName)
				}

				tagMap[tagName] = &tag
				tags = append(tags, tag)
			}
		}

		// Create bookmark
		bookmark := models.Bookmarks{
			Title:       bookmarkSeed.Title,
			Url:         bookmarkSeed.URL,
			Favicon:     bookmarkSeed.Favicon,
			Description: bookmarkSeed.Description,
			Pinned:      bookmarkSeed.Pinned,
			IsArchived:  bookmarkSeed.IsArchived,
			VisitCount:  bookmarkSeed.VisitCount,
			Tags:        tags,
			UserID:      userID,
		}

		// Set LastVisited if it exists
		if bookmarkSeed.LastVisited != nil {
			bookmark.LastVisited = *bookmarkSeed.LastVisited
		}

		// Create the bookmark
		if err := DB.Create(&bookmark).Error; err != nil {
			// Check if it's a foreign key constraint error - fail fast
			if strings.Contains(err.Error(), "fk_bookmarks_user") ||
				strings.Contains(err.Error(), "foreign key constraint") ||
				strings.Contains(err.Error(), "violates foreign key") {
				return fmt.Errorf("❌ Foreign key constraint error: User ID '%s' does not exist in neon_auth.users_sync.\n   This should not happen as we validated the user earlier.\n   Error: %w", userID, err)
			}

			// For other errors (like duplicate URLs/titles), just warn and continue
			fmt.Printf("⚠️  Warning: Failed to create bookmark '%s': %v\n", bookmarkSeed.Title, err)
			continue
		}

		// Update CreatedAt to match seed data
		DB.Model(&bookmark).Update("created_at", bookmarkSeed.CreatedAt)

		fmt.Printf("   [%d/%d] ✓ Created: %s\n", i+1, len(seedData.Bookmarks), bookmarkSeed.Title)
	}

	fmt.Println("✅ Database seeding completed!")
	return nil
}

// ClearDatabase removes all data from the database (use with caution!)
func ClearDatabase() error {
	fmt.Println("🧹 Clearing database...")

	// Delete in order to respect foreign key constraints
	if err := DB.Exec("DELETE FROM bookmark_tags").Error; err != nil {
		return fmt.Errorf("failed to clear bookmark_tags: %w", err)
	}

	if err := DB.Exec("DELETE FROM bookmarks").Error; err != nil {
		return fmt.Errorf("failed to clear bookmarks: %w", err)
	}

	if err := DB.Exec("DELETE FROM tags").Error; err != nil {
		return fmt.Errorf("failed to clear tags: %w", err)
	}

	fmt.Println("✅ Database cleared!")
	return nil
}

// ResetAndSeed clears the database and seeds it with fresh data
func ResetAndSeed(userID string) error {
	if err := ClearDatabase(); err != nil {
		return err
	}
	return SeedDatabase(userID)
}
