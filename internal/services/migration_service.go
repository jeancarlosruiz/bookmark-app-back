package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"gorm.io/gorm"
)

type MigrationService struct {
	db           *gorm.DB
	cacheService *CacheService
}

func NewMigrationService() *MigrationService {
	return &MigrationService{
		db:           database.DB,
		cacheService: &CacheService{},
	}
}

type MigrationResult struct {
	BookmarksMigrated int `json:"bookmarks_migrated"`
	BookmarksMerged   int `json:"bookmarks_merged"`
	TagsMigrated      int `json:"tags_migrated"`
	TagsMerge         int `json:"tags_merged"`
}

// Migrate user transfers all bookmarks and tags from anonymouseUserID to AuthenticatedUserID

func (s *MigrationService) MigrateUser(ctx context.Context, anonymousUserID, authenticatedUserID string) (*MigrationResult, error) {

	if anonymousUserID == "" || authenticatedUserID == "" {
		return nil, fmt.Errorf("Both anonymousUserID and authenticatedUserID user id's are required")
	}

	if anonymousUserID == authenticatedUserID {
		return nil, fmt.Errorf("anonymous and authenticated user ID's can not be the same")
	}

	result := &MigrationResult{}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Migrate tags first
		tagsMigrated, tagsMerge, err := s.migrateTags(tx, anonymousUserID, authenticatedUserID)

		if err != nil {
			return fmt.Errorf("tag migration failed: %w", err)
		}

		result.TagsMigrated = tagsMigrated
		result.TagsMerge = tagsMerge

		bookmarksMigrated, bookmarksMerged, err := s.migrateBookmarks(tx, anonymousUserID, authenticatedUserID)

		if err != nil {
			return fmt.Errorf("bookmark migration failed: %w", err)
		}

		result.BookmarksMigrated = bookmarksMigrated
		result.BookmarksMerged = bookmarksMerged

		//Both migration succeeded
		return nil
	})

	if err != nil {
		return nil, err
	}

	// invalidate user cache
	context := context.Background()
	_ = s.cacheService.InvalidateUserCache(context, authenticatedUserID)

	log.Printf("Migration completed: %d bookmarks (%d merged), %d tags (%d merged) from user %user %s to %s", result.BookmarksMigrated, result.BookmarksMerged, result.TagsMigrated, result.TagsMerge, anonymousUserID, authenticatedUserID)

	return result, nil
}

func (s *MigrationService) migrateTags(tx *gorm.DB, fromUserID, toUserID string) (int, int, error) {
	var anonymousTags []models.Tag
	if err := tx.Where("user_id = ?", fromUserID).Find(&anonymousTags).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch anonymous user tags: %w", err)
	}

	// No tags, early return
	if len(anonymousTags) == 0 {
		return 0, 0, nil
	}

	var authenticatedTags []models.Tag
	if err := tx.Where("user_id = ?", toUserID).Find(&authenticatedTags).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch authenticated user tags: %w", err)
	}

	// map for fast lookup

	authenticatedTagMap := make(map[string]uint)
	for _, tag := range authenticatedTags {
		authenticatedTagMap[strings.ToLower(tag.Title)] = tag.ID
	}

	migrated := 0
	merged := 0

	for _, anonTag := range anonymousTags {
		normalizedTitle := strings.ToLower(anonTag.Title)

		// Check if authenticated user already has this tag
		if existingTagID, exists := authenticatedTagMap[normalizedTitle]; exists {

			// Conflict detected: user has this tag with this name

			// Update bookmark_tags table: change references from anonTag.ID to existinTagID
			if err := tx.Model(&models.BookmarkTag{}).Where("tag_id = ?", anonTag.ID).Update("tag_id", existingTagID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to migrate bookmarks_tags for tag '$s': %w", anonTag.Title, err)
			}

			// delete the anon tag, no nedded it anymore
			if err := tx.Delete(&anonTag).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to delete anonymous tag '%s': %w", anonTag.Title)
			}

			merged++
		} else {
			// No conflict: simply transfer ownership of this tag to the authenticated one

			// update the tag's user_id
			if err := tx.Model(&anonTag).Update("user_id", toUserID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to update user_id for tag '%s': %w", anonTag.Title, err)
			}
			authenticatedTagMap[normalizedTitle] = anonTag.ID

			migrated++
		}
	}

	return migrated, merged, nil
}

// bookmarks migrations functions:
func (s *MigrationService) migrateBookmarks(tx *gorm.DB, fromUserID, toUserID string) (int, int, error) {

	var anonymousBookmarks []models.Bookmarks

	// get all bookmarks from anonymous user with tag preloaded
	if err := tx.Preload("Tags").Where("user_id = ?", fromUserID).Find(&anonymousBookmarks).Error; err != nil {
		return 0, 0, fmt.Errorf("Failed to fetch anonymous user bookmarks: %w", err)
	}

	if len(anonymousBookmarks) == 0 {
		return 0, 0, nil
	}

	// get all bookmarks from authenticated user (including soft-deleted to avoid unique constraint violations)
	var authenticatedBookmarks []models.Bookmarks

	if err := tx.Unscoped().Preload("Tags").Where("user_id = ?", toUserID).Find(&authenticatedBookmarks).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch authenticated user bookmarks: %w", err)
	}

	// Map for fast url conflict detection
	authenticatedBookmarkMap := make(map[string]*models.Bookmarks)

	for i := range authenticatedBookmarks {
		normalizedURL := normalizeURL(authenticatedBookmarks[i].Url)
		authenticatedBookmarkMap[normalizedURL] = &authenticatedBookmarks[i]
	}

	migrated := 0
	merged := 0

	// process each anonymous bookmark
	for _, anonBookmark := range anonymousBookmarks {
		normalizedURL := normalizeURL(anonBookmark.Url)

		// check if authenticated user has this url
		if existingBookmark, exists := authenticatedBookmarkMap[normalizedURL]; exists {

			// If the existing bookmark is soft-deleted, permanently delete it and transfer the anonymous one
			if existingBookmark.DeletedAt.Valid {
				// Permanently delete the soft-deleted bookmark
				if err := tx.Unscoped().Delete(existingBookmark).Error; err != nil {
					return 0, 0, fmt.Errorf("failed to permanently delete soft-deleted bookmark '%s': %w", existingBookmark.Url, err)
				}

				// Transfer ownership of anonymous bookmark
				if err := tx.Model(&anonBookmark).Update("user_id", toUserID).Error; err != nil {
					return 0, 0, fmt.Errorf("failed to update user_id for bookmark '%s': %w", anonBookmark.Url, err)
				}

				// Update map for future conflict detection
				authenticatedBookmarkMap[normalizedURL] = &anonBookmark

				migrated++
			} else {
				// Existing bookmark is active - merge data

				//user already bookmarked this url
				existingTagsIDs := make(map[uint]bool)

				// merge tags: add anonymous bookmarks's tags to existing bookmark
				for _, tag := range existingBookmark.Tags {
					existingTagsIDs[tag.ID] = true
				}

				// Add tags from anonymous bookmark that dont already exist
				for _, tag := range anonBookmark.Tags {

					//Associate this tag with the existing bookmark
					if !existingTagsIDs[tag.ID] {
						if err := tx.Model(existingBookmark).Association("Tags").Append(&tag); err != nil {
							return 0, 0, fmt.Errorf("failed to merge tags for bookmark '%s': %w", anonBookmark.Url, err)
						}
					}
				}

				// Aggregate visit counts: sum both
				newVisitCount := existingBookmark.VisitCount + anonBookmark.VisitCount

				// Update the last visited
				var lastVisitedUpdated *time.Time

				switch {
				case existingBookmark.LastVisited == nil:
					lastVisitedUpdated = anonBookmark.LastVisited

				case anonBookmark.LastVisited == nil:
					lastVisitedUpdated = existingBookmark.LastVisited

				case anonBookmark.LastVisited.After(*existingBookmark.LastVisited):
					lastVisitedUpdated = anonBookmark.LastVisited

				default:
					lastVisitedUpdated = existingBookmark.LastVisited
				}

				updates := make(map[string]interface{})

				updates["visit_count"] = newVisitCount
				updates["last_visited"] = lastVisitedUpdated

				if err := tx.Model(existingBookmark).Updates(updates).Error; err != nil {
					return 0, 0, fmt.Errorf("failed to update visit and last visited for bookmark '%s': %w", anonBookmark.Url, err)
				}

				// Delete the anonymous bookmark (data is now merged)
				if err := tx.Unscoped().Delete(&anonBookmark).Error; err != nil {
					return 0, 0, fmt.Errorf("failed to delete anonymous bookmark '%s': %w", anonBookmark.Url)
				}

				merged++
			}
		} else {
			// No Conflict: just transfer the ownership to authenticated user
			// Update the bookmark user id
			if err := tx.Model(&anonBookmark).Update("user_id", toUserID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to update user_id for bookmark '%s': %w", anonBookmark.Url, err)
			}

			// add to map for future conflict detection
			authenticatedBookmarkMap[normalizedURL] = &anonBookmark

			migrated++
		}
	}

	return migrated, merged, nil
}

// NormalizeURL standardizes urls for comparison
func normalizeURL(url string) string {
	//convert to lower case
	normalized := strings.ToLower(url)

	//remove the whitespace
	normalized = strings.TrimSpace(normalized)

	//remove protocol
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "https://")

	// remove trailing slash for consisteny
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized
}
