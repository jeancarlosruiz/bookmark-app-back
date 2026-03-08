---
title: Backend Migration Implementation Guide
created: 2025-01-05
updated: 2026-03-08
---

# Backend Migration Implementation Guide

## Table of Contents

1. [Overview](#overview)
2. [Migration Flow Diagram](#migration-flow-diagram)
3. [Step 1: Create Internal API Key Middleware](#step-1-create-internal-api-key-middleware)
4. [Step 2: Create Migration Service](#step-2-create-migration-service)
5. [Step 3: Create Migration Controller](#step-3-create-migration-controller)
6. [Step 4: Register Internal Routes](#step-4-register-internal-routes)
7. [Environment Variables Setup](#environment-variables-setup)
8. [Testing Guide](#testing-guide)
9. [Edge Cases and Troubleshooting](#edge-cases-and-troubleshooting)
10. [Security Checklist](#security-checklist)

---

## Overview

### Purpose

This guide explains how to implement a secure backend endpoint that migrates bookmarks and tags from an **anonymous user** (identified by a temporary ID) to an **authenticated user** (identified by a JWT token).

### Why We Need This

When users browse your bookmark app without logging in, their data is stored under a temporary anonymous user ID. When they sign in for the first time, we need to:

1. Transfer all their anonymous bookmarks to their real user account
2. Merge tags intelligently (avoiding duplicates)
3. Handle conflicts when bookmarks already exist
4. Maintain data integrity with database transactions

### Why Internal API Authentication?

This migration endpoint is **not meant for direct client access**. It should only be called by your trusted Next.js backend during the authentication flow. Using an internal API key ensures:

- Only your Next.js server can trigger migrations
- Malicious actors can't migrate data between arbitrary users
- You maintain control over when and how migrations happen

### Architecture Pattern

Following your existing architecture:

```
HTTP POST → Internal Auth Middleware → Protect (JWT) → Controller → Service → Repository → PostgreSQL
```

The service layer handles all business logic (conflict detection, merging) and wraps everything in a GORM transaction for atomicity.

---

## Migration Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Next.js Backend                             │
│  (Better Auth callback after user signs in for first time)         │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             │ POST /internal/migrate
                             │ Headers: X-Internal-API-Key, Authorization (JWT)
                             │ Body: { "anonymous_user_id": "anon_123" }
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Go Backend                                 │
├─────────────────────────────────────────────────────────────────────┤
│  1. Internal Auth Middleware                                        │
│     └─> Verify X-Internal-API-Key matches INTERNAL_API_KEY          │
│                                                                     │
│  2. Protect Middleware (JWT)                                        │
│     └─> Extract authenticated user_id from token                    │
│                                                                     │
│  3. Migration Controller                                            │
│     └─> Validate request body                                       │
│     └─> Call MigrationService.MigrateUser()                         │
│                                                                     │
│  4. Migration Service (in GORM transaction)                         │
│     ├─> Step A: Migrate Tags                                        │
│     │   ├─> Get all tags from anonymous user                        │
│     │   ├─> For each tag:                                           │
│     │   │   ├─> Check if authenticated user has same tag name       │
│     │   │   ├─> If exists: merge bookmark_tags, delete anon tag     │
│     │   │   └─> If not: update tag.user_id to authenticated user    │
│     │   └─> Return migrated/merged tag count                        │
│     │                                                                │
│     └─> Step B: Migrate Bookmarks                                   │
│         ├─> Get all bookmarks from anonymous user                   │
│         ├─> For each bookmark:                                      │
│         │   ├─> Normalize URL (lowercase, trim, remove protocol)    │
│         │   ├─> Check if authenticated user has same URL            │
│         │   ├─> If exists: merge tags, sum visit_count, delete anon │
│         │   └─> If not: update bookmark.user_id to authenticated    │
│         └─> Return migrated/merged bookmark count                   │
│                                                                     │
│  5. Commit Transaction → Return JSON Response                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Step 1: Create Internal API Key Middleware

### File: `internal/middleware/internal_auth.go`

This middleware validates that requests come from your trusted Next.js backend by checking a shared secret key.

### Implementation

```go
package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// InternalAPIAuth validates the X-Internal-API-Key header against the environment variable
// This ensures only trusted services (like your Next.js backend) can call internal endpoints
func InternalAPIAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract the API key from the request header
		// The client must send: X-Internal-API-Key: <secret>
		clientKey := c.GetHeader("X-Internal-API-Key")

		// 2. Get the expected API key from environment variables
		// This should be a long, randomly generated secret shared between Next.js and Go
		expectedKey := os.Getenv("INTERNAL_API_KEY")

		// 3. Security check: if no key is configured, reject all requests
		// This prevents accidental deployment without proper security
		if expectedKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal API key not configured",
			})
			c.Abort()
			return
		}

		// 4. Validate the key using constant-time comparison
		// CRITICAL: Use subtle.ConstantTimeCompare to prevent timing attacks
		// A timing attack could reveal the key by measuring how long comparisons take
		// Normal string comparison (clientKey == expectedKey) would be vulnerable
		if subtle.ConstantTimeCompare([]byte(clientKey), []byte(expectedKey)) != 1 {
			// Keys don't match - unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing internal API key",
			})
			c.Abort()
			return
		}

		// 5. Key is valid - continue to next middleware/handler
		c.Next()
	}
}
```

### Key Concepts Explained

**Why `subtle.ConstantTimeCompare`?**

Regular string comparison (`==`) returns immediately when it finds the first differing character. An attacker could measure response times to guess the key character-by-character. `subtle.ConstantTimeCompare` always takes the same time regardless of where the strings differ.

**Why check for empty `expectedKey`?**

If someone forgets to set `INTERNAL_API_KEY` in production, we want to fail securely (reject all requests) rather than accept any key.

**Header naming convention:**

`X-Internal-API-Key` is a custom header. The `X-` prefix indicates it's application-specific (though modern HTTP standards discourage the prefix, it's still widely used for custom headers).

---

## Step 2: Create Migration Service

### File: `internal/services/migration_service.go`

This service contains all the business logic for migrating user data. It's split into logical parts for clarity.

### Part A: Main Migration Function

```go
package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go-bookmark/internal/database"
	"go-bookmark/internal/models"

	"gorm.io/gorm"
)

// MigrationService handles user data migration
type MigrationService struct {
	db *gorm.DB
}

// NewMigrationService creates a new migration service instance
func NewMigrationService() *MigrationService {
	return &MigrationService{
		db: database.DB, // Use the global database connection
	}
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	BookmarksMigrated int `json:"bookmarks_migrated"`
	BookmarksMerged   int `json:"bookmarks_merged"`
	TagsMigrated      int `json:"tags_migrated"`
	TagsMerged        int `json:"tags_merged"`
}

// MigrateUser transfers all bookmarks and tags from anonymousUserID to authenticatedUserID
// Returns detailed migration results or an error
func (s *MigrationService) MigrateUser(ctx context.Context, anonymousUserID, authenticatedUserID string) (*MigrationResult, error) {
	// Validation: ensure both user IDs are provided
	if anonymousUserID == "" || authenticatedUserID == "" {
		return nil, fmt.Errorf("both anonymous and authenticated user IDs are required")
	}

	// Validation: ensure they're not the same (prevent no-op migrations)
	if anonymousUserID == authenticatedUserID {
		return nil, fmt.Errorf("anonymous and authenticated user IDs cannot be the same")
	}

	// Initialize result tracking
	result := &MigrationResult{}

	// Start a database transaction
	// WHY: We need atomicity - either ALL data migrates successfully, or NONE of it does
	// If tag migration succeeds but bookmark migration fails, we roll back the entire operation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// STEP 1: Migrate tags first
		// WHY: Bookmarks reference tags, so we need tags to exist with correct user_id
		// before we can properly associate them with migrated bookmarks
		tagsMigrated, tagsMerged, err := s.migrateTags(tx, anonymousUserID, authenticatedUserID)
		if err != nil {
			// Transaction will automatically rollback on error return
			return fmt.Errorf("tag migration failed: %w", err)
		}
		result.TagsMigrated = tagsMigrated
		result.TagsMerged = tagsMerged

		// STEP 2: Migrate bookmarks
		// Tags are now ready, so bookmark-tag associations will be correct
		bookmarksMigrated, bookmarksMerged, err := s.migrateBookmarks(tx, anonymousUserID, authenticatedUserID)
		if err != nil {
			return fmt.Errorf("bookmark migration failed: %w", err)
		}
		result.BookmarksMigrated = bookmarksMigrated
		result.BookmarksMerged = bookmarksMerged

		// If we reach here, both migrations succeeded
		// Transaction will auto-commit when we return nil
		return nil
	})

	// Check if transaction failed
	if err != nil {
		return nil, err
	}

	// Log successful migration for debugging/auditing
	log.Printf("Migration completed: %d bookmarks (%d merged), %d tags (%d merged) from user %s to %s",
		result.BookmarksMigrated, result.BookmarksMerged,
		result.TagsMigrated, result.TagsMerged,
		anonymousUserID, authenticatedUserID)

	return result, nil
}
```

### Part B: Tag Migration Function

```go
// migrateTags handles tag migration with conflict resolution
// Returns: (tags migrated, tags merged, error)
func (s *MigrationService) migrateTags(tx *gorm.DB, fromUserID, toUserID string) (int, int, error) {
	// 1. Get all tags from the anonymous user
	var anonymousTags []models.Tag
	if err := tx.Where("user_id = ?", fromUserID).Find(&anonymousTags).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch anonymous user tags: %w", err)
	}

	// Early return if no tags to migrate
	if len(anonymousTags) == 0 {
		return 0, 0, nil
	}

	// 2. Get all existing tags from the authenticated user for conflict detection
	var authenticatedTags []models.Tag
	if err := tx.Where("user_id = ?", toUserID).Find(&authenticatedTags).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch authenticated user tags: %w", err)
	}

	// Build a map for fast lookup: tag_name -> tag_id
	// WHY: O(1) lookup instead of O(n) for each anonymous tag
	authenticatedTagMap := make(map[string]uint)
	for _, tag := range authenticatedTags {
		// Normalize to case-insensitive comparison (user sees "Go" and "go" as duplicates)
		authenticatedTagMap[strings.ToLower(tag.Title)] = tag.ID
	}

	migrated := 0
	merged := 0

	// 3. Process each anonymous tag
	for _, anonTag := range anonymousTags {
		normalizedTitle := strings.ToLower(anonTag.Title)

		// Check if authenticated user already has this tag
		if existingTagID, exists := authenticatedTagMap[normalizedTitle]; exists {
			// CONFLICT DETECTED: Authenticated user already has a tag with this name

			// 3a. Migrate bookmark associations from anonymous tag to existing authenticated tag
			// Update bookmark_tags table: change references from anonTag.ID to existingTagID
			if err := tx.Model(&models.BookmarkTag{}).
				Where("tag_id = ?", anonTag.ID).
				Update("tag_id", existingTagID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to migrate bookmark_tags for tag '%s': %w", anonTag.Title, err)
			}

			// 3b. Delete the anonymous tag (no longer needed, data is merged)
			if err := tx.Delete(&anonTag).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to delete anonymous tag '%s': %w", anonTag.Title, err)
			}

			merged++
		} else {
			// NO CONFLICT: Simply transfer ownership of this tag to authenticated user

			// 3c. Update the tag's user_id
			if err := tx.Model(&anonTag).Update("user_id", toUserID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to update user_id for tag '%s': %w", anonTag.Title, err)
			}

			// Add to map so subsequent tags can detect conflicts
			authenticatedTagMap[normalizedTitle] = anonTag.ID

			migrated++
		}
	}

	return migrated, merged, nil
}
```

### Part C: Bookmark Migration Function

```go
// migrateBookmarks handles bookmark migration with conflict resolution
// Returns: (bookmarks migrated, bookmarks merged, error)
func (s *MigrationService) migrateBookmarks(tx *gorm.DB, fromUserID, toUserID string) (int, int, error) {
	// 1. Get all bookmarks from anonymous user with their tags preloaded
	// WHY Preload: Avoid N+1 queries when merging tags later
	var anonymousBookmarks []models.Bookmarks
	if err := tx.Preload("Tags").Where("user_id = ?", fromUserID).Find(&anonymousBookmarks).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch anonymous user bookmarks: %w", err)
	}

	if len(anonymousBookmarks) == 0 {
		return 0, 0, nil
	}

	// 2. Get all existing bookmarks from authenticated user
	var authenticatedBookmarks []models.Bookmarks
	if err := tx.Preload("Tags").Where("user_id = ?", toUserID).Find(&authenticatedBookmarks).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch authenticated user bookmarks: %w", err)
	}

	// Build a map for fast URL conflict detection: normalized_url -> bookmark
	authenticatedBookmarkMap := make(map[string]*models.Bookmarks)
	for i := range authenticatedBookmarks {
		normalizedURL := normalizeURL(authenticatedBookmarks[i].Url)
		authenticatedBookmarkMap[normalizedURL] = &authenticatedBookmarks[i]
	}

	migrated := 0
	merged := 0

	// 3. Process each anonymous bookmark
	for _, anonBookmark := range anonymousBookmarks {
		normalizedURL := normalizeURL(anonBookmark.Url)

		// Check if authenticated user already has this URL
		if existingBookmark, exists := authenticatedBookmarkMap[normalizedURL]; exists {
			// CONFLICT DETECTED: Authenticated user already bookmarked this URL

			// 3a. Merge tags: add anonymous bookmark's tags to existing bookmark
			// Build a set of existing tag IDs to avoid duplicates
			existingTagIDs := make(map[uint]bool)
			for _, tag := range existingBookmark.Tags {
				existingTagIDs[tag.ID] = true
			}

			// Add tags from anonymous bookmark that don't already exist
			for _, tag := range anonBookmark.Tags {
				if !existingTagIDs[tag.ID] {
					// Associate this tag with the existing bookmark
					if err := tx.Model(existingBookmark).Association("Tags").Append(&tag); err != nil {
						return 0, 0, fmt.Errorf("failed to merge tags for bookmark '%s': %w", anonBookmark.Url, err)
					}
				}
			}

			// 3b. Aggregate visit counts: sum anonymous + authenticated
			newVisitCount := existingBookmark.VisitCount + anonBookmark.VisitCount
			if err := tx.Model(existingBookmark).Update("visit_count", newVisitCount).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to update visit count for bookmark '%s': %w", anonBookmark.Url, err)
			}

			// 3c. Delete the anonymous bookmark (data is now merged)
			// Use Unscoped to permanently delete (bypass soft delete)
			if err := tx.Unscoped().Delete(&anonBookmark).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to delete anonymous bookmark '%s': %w", anonBookmark.Url, err)
			}

			merged++
		} else {
			// NO CONFLICT: Simply transfer ownership to authenticated user

			// 3d. Update the bookmark's user_id
			if err := tx.Model(&anonBookmark).Update("user_id", toUserID).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to update user_id for bookmark '%s': %w", anonBookmark.Url, err)
			}

			// Add to map for future conflict detection
			authenticatedBookmarkMap[normalizedURL] = &anonBookmark

			migrated++
		}
	}

	return migrated, merged, nil
}
```

### Part D: Helper Functions

```go
// normalizeURL standardizes URLs for comparison
// This ensures we treat "HTTPS://Example.com/" and "http://example.com" as the same
func normalizeURL(url string) string {
	// 1. Convert to lowercase (case-insensitive comparison)
	normalized := strings.ToLower(url)

	// 2. Remove leading/trailing whitespace
	normalized = strings.TrimSpace(normalized)

	// 3. Remove protocol prefix (http://, https://)
	// WHY: Users consider "example.com" and "https://example.com" as duplicates
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "https://")

	// 4. Remove trailing slash for consistency
	// "example.com/" and "example.com" should be treated as the same
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized
}
```

### Why This Design?

**Transaction-based:** All-or-nothing ensures data integrity. If anything fails, we roll back completely.

**Tags first:** We migrate tags before bookmarks because bookmarks reference tags. If we did it backwards, bookmark-tag associations might break.

**Conflict resolution:** Instead of rejecting migrations when duplicates exist, we intelligently merge data (combine tags, sum visit counts).

**Performance:** Using maps for lookups (`O(1)`) instead of nested loops (`O(n²)`).

**Error context:** Wrapping errors with `fmt.Errorf("...: %w", err)` provides debugging context while preserving the original error.

---

## Step 3: Create Migration Controller

### File: `internal/controllers/migration_controller.go`

The controller handles HTTP concerns (parsing requests, returning JSON) and delegates business logic to the service.

### Implementation

```go
package controllers

import (
	"net/http"

	"go-bookmark/internal/services"

	"github.com/gin-gonic/gin"
)

// MigrationController handles migration-related HTTP requests
type MigrationController struct {
	migrationService *services.MigrationService
}

// NewMigrationController creates a new migration controller
func NewMigrationController() *MigrationController {
	return &MigrationController{
		migrationService: services.NewMigrationService(),
	}
}

// MigrateUserRequest defines the expected request body structure
type MigrateUserRequest struct {
	// The anonymous user ID whose data should be migrated
	// This comes from the Next.js frontend (stored in localStorage before auth)
	AnonymousUserID string `json:"anonymous_user_id" binding:"required"`
}

// MigrateUser handles POST /internal/migrate
// Transfers bookmarks and tags from an anonymous user to an authenticated user
func (mc *MigrationController) MigrateUser(c *gin.Context) {
	// 1. Parse and validate request body
	var req MigrateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 400 Bad Request: invalid JSON or missing required fields
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: anonymous_user_id is required",
		})
		return
	}

	// 2. Extract authenticated user ID from JWT token context
	// This was set by the Protect middleware after validating the JWT
	authenticatedUserID := c.GetString("user_id")
	if authenticatedUserID == "" {
		// This should never happen if Protect middleware is working correctly
		// But defensive programming is good practice
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token context",
		})
		return
	}

	// 3. Call the migration service to perform the actual migration
	result, err := mc.migrationService.MigrateUser(c.Request.Context(), req.AnonymousUserID, authenticatedUserID)
	if err != nil {
		// 500 Internal Server Error: migration failed
		// The service already wrapped errors with context, so we can return them directly
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Migration failed",
			"details": err.Error(),
		})
		return
	}

	// 4. Return success response with migration statistics
	// 200 OK: migration completed successfully
	c.JSON(http.StatusOK, gin.H{
		"message": "User data migrated successfully",
		"data":    result,
	})
}
```

### Controller Best Practices

**Thin controllers:** All business logic lives in the service layer. The controller only handles HTTP concerns.

**Context propagation:** We pass `c.Request.Context()` to the service so it can respect request cancellation.

**Defensive checks:** Even though middleware should set `user_id`, we check it anyway to fail gracefully.

**Clear error responses:** We return both a generic message and specific details to help with debugging.

---

## Step 4: Register Internal Routes

### File: `internal/routes/routes.go`

Add a new route group for internal endpoints that require the internal API key.

### Implementation

Find your existing `SetupRoutes` function and add this section:

```go
package routes

import (
	"go-bookmark/internal/controllers"
	"go-bookmark/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// ... your existing routes ...

	// ============================================================
	// INTERNAL ROUTES (only accessible with internal API key)
	// ============================================================
	// These endpoints are for backend-to-backend communication only
	// They should NEVER be called directly by the frontend
	internal := r.Group("/internal")

	// Apply the internal API key authentication middleware
	// This ensures only requests with the correct X-Internal-API-Key header are allowed
	internal.Use(middleware.InternalAPIAuth())
	{
		// Migration endpoint
		// POST /internal/migrate
		// Headers required:
		//   - X-Internal-API-Key: <INTERNAL_API_KEY from .env>
		//   - Authorization: Bearer <JWT token of authenticated user>
		// Body: { "anonymous_user_id": "anon_abc123" }
		migrationController := controllers.NewMigrationController()

		// Apply both InternalAPIAuth (via group) AND Protect (JWT validation)
		// WHY: We need to know BOTH that the request is from our backend (internal key)
		// AND which user to migrate data TO (from JWT)
		internal.POST("/migrate", middleware.Protect(), migrationController.MigrateUser)
	}

	// ... rest of your routes ...
}
```

### Route Security Layers

This endpoint has **two layers of authentication**:

1. **Internal API Key** (via `InternalAPIAuth` middleware): Proves the request is from your Next.js backend
2. **JWT Token** (via `Protect` middleware): Identifies which authenticated user to migrate data to

This dual authentication prevents:
- External attackers calling the endpoint (no internal key)
- Malicious insiders migrating data to wrong users (JWT tied to specific user)

---

## Environment Variables Setup

### File: `.env`

Add this new variable to your environment configuration:

```bash
# Internal API key for backend-to-backend communication
# This should be a long, random, cryptographically secure string
# It must match the key configured in your Next.js backend
INTERNAL_API_KEY=your-generated-key-here
```

### Generating a Secure API Key

Use one of these methods to generate a cryptographically secure random key:

#### Option 1: OpenSSL (Recommended - 32 bytes = 256 bits)

```bash
openssl rand -base64 32
```

Example output: `J8fK3mN9qP2rS5tV8wX0yZ1aB4cD6eF7gH9iJ0kL2mN=`

#### Option 2: Node.js (if you have Node installed)

```bash
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

#### Option 3: Python

```bash
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

### Important Security Notes

- **Never commit this key to git:** Add `.env` to your `.gitignore`
- **Use different keys for different environments:** Development, staging, and production should have unique keys
- **Store securely in production:** Use environment variables in your deployment platform (Vercel, Railway, etc.)
- **Rotate periodically:** Change the key every 6-12 months as a security best practice

### Next.js Configuration

In your Next.js backend, store the **same key**:

```typescript
// .env.local (Next.js)
INTERNAL_API_KEY=J8fK3mN9qP2rS5tV8wX0yZ1aB4cD6eF7gH9iJ0kL2mN=
GO_BACKEND_URL=http://localhost:8080
```

---

## Testing Guide

### Test Scenario 1: Normal Migration (No Conflicts)

**Setup:**
1. Create some bookmarks as an anonymous user (save the anonymous user ID)
2. Sign in with a new account (get JWT token)
3. Call the migration endpoint

**Request:**

```bash
curl -X POST http://localhost:8080/internal/migrate \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: YOUR_INTERNAL_API_KEY" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "anonymous_user_id": "anon_abc123"
  }'
```

**Expected Response (200 OK):**

```json
{
  "message": "User data migrated successfully",
  "data": {
    "bookmarks_migrated": 5,
    "bookmarks_merged": 0,
    "tags_migrated": 3,
    "tags_merged": 0
  }
}
```

**Verification:**
- Query the database: all bookmarks should now have the authenticated user's ID
- Anonymous user should have no bookmarks left

### Test Scenario 2: Duplicate URL Handling

**Setup:**
1. As anonymous user: bookmark `https://example.com`
2. Sign in, then bookmark `https://example.com` again as authenticated user
3. Try migration

**Expected Behavior:**
- Only one bookmark for `example.com` exists after migration
- Visit counts are summed
- Tags from both bookmarks are merged

**Expected Response (200 OK):**

```json
{
  "message": "User data migrated successfully",
  "data": {
    "bookmarks_migrated": 4,
    "bookmarks_merged": 1,
    "tags_migrated": 3,
    "tags_merged": 0
  }
}
```

### Test Scenario 3: Duplicate Tag Handling

**Setup:**
1. As anonymous user: create tag "JavaScript"
2. Sign in and create tag "javascript" (case-insensitive duplicate)
3. Try migration

**Expected Behavior:**
- Only one "javascript" tag exists after migration
- All bookmark associations are preserved under the authenticated user's tag

**Expected Response (200 OK):**

```json
{
  "message": "User data migrated successfully",
  "data": {
    "bookmarks_migrated": 5,
    "bookmarks_merged": 0,
    "tags_migrated": 2,
    "tags_merged": 1
  }
}
```

### Test Scenario 4: Invalid Internal API Key

**Request:**

```bash
curl -X POST http://localhost:8080/internal/migrate \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: wrong-key" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "anonymous_user_id": "anon_abc123"
  }'
```

**Expected Response (401 Unauthorized):**

```json
{
  "error": "Invalid or missing internal API key"
}
```

### Test Scenario 5: Missing JWT Token

**Request:**

```bash
curl -X POST http://localhost:8080/internal/migrate \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: YOUR_INTERNAL_API_KEY" \
  -d '{
    "anonymous_user_id": "anon_abc123"
  }'
```

**Expected Response (401 Unauthorized):**

```json
{
  "error": "Authorization header missing or invalid"
}
```

### Test Scenario 6: Empty Anonymous User

**Setup:**
- Provide an anonymous user ID that has no bookmarks

**Expected Response (200 OK):**

```json
{
  "message": "User data migrated successfully",
  "data": {
    "bookmarks_migrated": 0,
    "bookmarks_merged": 0,
    "tags_migrated": 0,
    "tags_merged": 0
  }
}
```

### Testing with Postman

1. Create a new POST request to `http://localhost:8080/internal/migrate`
2. In **Headers** tab, add:
   - `Content-Type`: `application/json`
   - `X-Internal-API-Key`: `<your key>`
   - `Authorization`: `Bearer <your JWT>`
3. In **Body** tab (raw JSON):
   ```json
   {
     "anonymous_user_id": "anon_abc123"
   }
   ```
4. Click Send

---

## Edge Cases and Troubleshooting

### Issue: Migration appears successful but data is missing

**Possible Cause:** Transaction rolled back due to a silent error.

**Solution:**
- Check your Go server logs for errors
- Add more logging in the migration service:
  ```go
  log.Printf("Migrating %d bookmarks from %s to %s", len(anonymousBookmarks), fromUserID, toUserID)
  ```

### Issue: Duplicate bookmarks after migration

**Possible Cause:** URL normalization is not working correctly.

**Debug:**
- Add logging to `normalizeURL()`:
  ```go
  func normalizeURL(url string) string {
      normalized := strings.ToLower(strings.TrimSpace(url))
      normalized = strings.TrimPrefix(normalized, "http://")
      normalized = strings.TrimPrefix(normalized, "https://")
      normalized = strings.TrimSuffix(normalized, "/")
      log.Printf("URL normalized: '%s' -> '%s'", url, normalized)
      return normalized
  }
  ```

**Solution:**
- Enhance normalization to handle more edge cases (www prefix, query parameters, etc.)

### Issue: Tags are duplicated after migration

**Possible Cause:** Tag title comparison is case-sensitive.

**Solution:**
- Verify you're using `strings.ToLower()` when building the `authenticatedTagMap`
- Consider trimming whitespace: `strings.TrimSpace(strings.ToLower(tag.Title))`

### Issue: Migration fails mid-way

**Symptom:** Some tags migrated but bookmarks didn't, or vice versa.

**Explanation:** This should **never** happen if transactions are working correctly.

**Troubleshooting:**
1. Check database transaction support (PostgreSQL should handle this automatically)
2. Verify `s.db.Transaction()` is being used
3. Check for `ROLLBACK` statements in PostgreSQL logs

### Issue: Performance is slow with many bookmarks

**Symptom:** Migration times out or takes >10 seconds.

**Optimization:**
- Batch update operations instead of one-by-one
- Add database indexes on `user_id` and `url` (you likely already have these)
- Consider paginating very large migrations

### Issue: Bookmark-tag associations are lost

**Possible Cause:** Tag migration is deleting tags that bookmarks still reference.

**Solution:**
- Ensure you're updating `bookmark_tags` foreign key references **before** deleting tags
- Verify the `UPDATE bookmark_tags SET tag_id = ?` query is executing
- Check database constraints (foreign keys should prevent orphans)

---

## Security Checklist

### ✅ Before Deploying to Production

- [ ] **Strong API Key**: Generated with `openssl rand -base64 32` or equivalent
- [ ] **Environment Variables**: `INTERNAL_API_KEY` set in both Go backend and Next.js backend
- [ ] **No Hardcoded Secrets**: Verify `.env` is in `.gitignore` and never committed
- [ ] **HTTPS Only**: Ensure production uses HTTPS to prevent API key interception
- [ ] **Different Keys Per Environment**: Dev/staging/prod have unique keys
- [ ] **JWT Validation**: Confirm `Protect()` middleware is applied to the migration route
- [ ] **Constant-Time Comparison**: Verify `subtle.ConstantTimeCompare` is used in middleware
- [ ] **Error Messages**: Ensure error responses don't leak sensitive information
- [ ] **Rate Limiting**: Consider adding rate limiting to prevent abuse (see below)
- [ ] **Audit Logging**: Log all migration attempts with user IDs and timestamps

### Additional Security Measures

#### Rate Limiting (Recommended)

Add rate limiting to prevent abuse:

```go
// In middleware/rate_limit.go
func RateLimitInternal() gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Every(1*time.Minute), 5) // 5 requests per minute

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many migration requests, please try again later",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

Apply to internal routes:

```go
internal.Use(middleware.InternalAPIAuth(), middleware.RateLimitInternal())
```

#### Audit Logging (Production)

Log all migration attempts for security auditing:

```go
// In migration_service.go, add to MigrateUser()
log.Printf("[AUDIT] Migration requested: from=%s to=%s timestamp=%s",
    anonymousUserID, authenticatedUserID, time.Now().UTC())

// After successful migration
log.Printf("[AUDIT] Migration completed: from=%s to=%s bookmarks=%d tags=%d",
    anonymousUserID, authenticatedUserID,
    result.BookmarksMigrated + result.BookmarksMerged,
    result.TagsMigrated + result.TagsMerged)
```

### Common Security Mistakes to Avoid

❌ **DON'T** expose this endpoint publicly without internal API key
❌ **DON'T** use predictable API keys like "secret123"
❌ **DON'T** log the internal API key in error messages
❌ **DON'T** allow migration from any user ID (always validate ownership)
❌ **DON'T** skip transaction handling (atomicity is critical)

✅ **DO** rotate API keys periodically
✅ **DO** use HTTPS in production
✅ **DO** monitor for suspicious migration patterns
✅ **DO** add metrics/monitoring for migration success rates
✅ **DO** test with real-world data volumes

---

## Appendix: Integration with Next.js

### Example Next.js API Route

Here's how your Next.js backend might call this endpoint:

```typescript
// app/api/auth/migrate/route.ts (Next.js 13+ App Router)
import { NextRequest, NextResponse } from 'next/server';
import { getSession } from '@/lib/auth'; // Your auth helper

export async function POST(req: NextRequest) {
  // 1. Get authenticated user's session
  const session = await getSession();
  if (!session?.user?.id) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }

  // 2. Get anonymous user ID from request
  const { anonymousUserId } = await req.json();
  if (!anonymousUserId) {
    return NextResponse.json({ error: 'Anonymous user ID required' }, { status: 400 });
  }

  // 3. Call Go backend migration endpoint
  const response = await fetch(`${process.env.GO_BACKEND_URL}/internal/migrate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Internal-API-Key': process.env.INTERNAL_API_KEY!,
      'Authorization': `Bearer ${session.accessToken}`, // JWT token
    },
    body: JSON.stringify({
      anonymous_user_id: anonymousUserId,
    }),
  });

  // 4. Return result to frontend
  const data = await response.json();
  return NextResponse.json(data, { status: response.status });
}
```

### Frontend Flow

```typescript
// When user signs in for the first time
const handleSignIn = async () => {
  // 1. Get anonymous user ID from localStorage
  const anonymousUserId = localStorage.getItem('anonymous_user_id');

  // 2. User signs in (Better Auth handles this)
  await signIn.email({ email, password });

  // 3. If anonymous ID exists, trigger migration
  if (anonymousUserId) {
    try {
      const response = await fetch('/api/auth/migrate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ anonymousUserId }),
      });

      const result = await response.json();
      console.log('Migration completed:', result);

      // 4. Clear anonymous ID
      localStorage.removeItem('anonymous_user_id');
    } catch (error) {
      console.error('Migration failed:', error);
      // Don't block sign-in if migration fails
    }
  }
};
```

---

## Summary

You now have a complete guide for implementing anonymous user data migration. Here's the checklist:

1. ✅ Create `internal/middleware/internal_auth.go` with constant-time API key validation
2. ✅ Create `internal/services/migration_service.go` with transaction-based migration logic
3. ✅ Create `internal/controllers/migration_controller.go` to handle HTTP requests
4. ✅ Update `internal/routes/routes.go` to register the `/internal/migrate` endpoint
5. ✅ Add `INTERNAL_API_KEY` to `.env` and generate a secure key
6. ✅ Test all scenarios (normal, conflicts, errors)
7. ✅ Review security checklist before production deployment

**Next Steps:**
- Implement the code following this guide
- Test thoroughly with real data
- Monitor migration success rates in production
- Consider adding metrics/observability

**Questions to Consider:**
- Should migrations be idempotent (safe to call multiple times)?
- Do you need to support partial rollbacks?
- Should you archive anonymous data instead of deleting it?
- Do you need to notify users of migration results?

Good luck with your implementation! 🚀
