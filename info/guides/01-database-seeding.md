# Database Seeding Guide

This guide explains how to seed your database with test data for development.

## Overview

The seeder reads bookmark data from `data.json` and populates your database with:
- 18 example bookmarks (various dev resources)
- Multiple tags (Tools, Community, CSS, JavaScript, etc.)
- Realistic visit counts and timestamps

## Prerequisites

1. Make sure your `.env` file is configured with a valid `DATABASE_URL`
2. Ensure you have a valid user ID from your external auth system (`neon_auth.users_sync` table)

## Getting a User ID

Since users come from an external auth system, you need to find or create a user first:

```bash
# Option 1: Check if you have any existing users
go run cmd/api/main.go

# Then query via API:
curl http://localhost:<PORT>/api/users

# Option 2: Manually insert a test user in PostgreSQL
# Connect to your database and run:
# INSERT INTO neon_auth.users_sync (id, name, email, created_at)
# VALUES ('test-user-123', 'Test User', 'test@example.com', NOW());
```

## Usage

### Basic Seed (adds data without clearing)

```bash
go run cmd/seed/main.go --user=your-user-id-here
```

This will:
- Create all tags from the bookmarks
- Create all 18 bookmarks
- Associate them with the specified user
- Skip duplicates if data already exists

### Reset and Seed (clears everything first)

```bash
go run cmd/seed/main.go --user=your-user-id-here --reset
```

This will:
- Delete all bookmark_tags relationships
- Delete all bookmarks
- Delete all tags
- Then seed fresh data

**⚠️ WARNING**: This will delete ALL bookmarks and tags in your database!

### Clear Only (remove all data)

```bash
go run cmd/seed/main.go --clear
```

This removes all bookmarks and tags without reseeding.

## Example Output

```
🔧 Running migrations...
🌱 Starting database seeding...
📚 Seeding 18 bookmarks...
   ✓ Created tag: Tools
   ✓ Created tag: Community
   ✓ Created tag: Git
   [1/18] ✓ Created: GitHub
   [2/18] ✓ Created: Stack Overflow
   [3/18] ✓ Created: MDN Web Docs
   ...
   [18/18] ✓ Created: Flexbox Zombies
✅ Database seeding completed!
🎉 Done!
```

## Customizing Seed Data

To modify the test data, edit `data.json`. The structure is:

```json
{
  "bookmarks": [
    {
      "id": "bm-001",
      "title": "Site Title",
      "url": "https://example.com",
      "favicon": "./assets/images/favicon.png",
      "description": "Description here",
      "tags": ["Tag1", "Tag2"],
      "pinned": false,
      "isArchived": false,
      "visitCount": 0,
      "createdAt": "2024-01-01T00:00:00Z",
      "lastVisited": "2024-01-02T00:00:00Z"
    }
  ]
}
```

## Troubleshooting

**Error: "failed to create bookmark"**
- Check for duplicate titles or URLs (both have unique constraints)
- Verify the user ID exists in `neon_auth.users_sync`

**Error: "Failed to connect to database"**
- Verify your `DATABASE_URL` in `.env`
- Ensure PostgreSQL is running
- Check database credentials

**Error: "--user flag is required"**
- You must provide a valid user ID: `--user=your-user-id`

## Integration with Main App

If you want to add a seed flag to your main application, add this to `cmd/api/main.go`:

```go
import "flag"

func main() {
    seed := flag.Bool("seed", false, "Seed the database")
    userID := flag.String("user", "", "User ID for seeding")
    flag.Parse()

    // ... existing config and database setup ...

    if *seed {
        if *userID == "" {
            panic("--user flag is required for seeding")
        }
        if err := database.SeedDatabase(*userID); err != nil {
            panic("Failed to seed database: " + err.Error())
        }
        fmt.Println("Database seeded successfully!")
        return
    }

    // ... rest of main.go ...
}
```

Then run: `go run cmd/api/main.go --seed --user=your-user-id`
