#!/bin/sh
set -e

# Clean vendor directory if it exists (ignore errors if mounted as volume)
rm -rf /app/vendor 2>/dev/null || true

# Run go mod tidy to ensure dependencies are correct
go mod tidy

# Execute the main command (Air)
exec "$@"
