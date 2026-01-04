#!/bin/sh
set -e

DB_PATH="${DB_PATH:-/data/lords.db}"

# Ensure data directory exists
mkdir -p "$(dirname "$DB_PATH")"

# Restore database from replica if it exists (and local DB doesn't)
# On first run, there's no backup - that's OK, we'll start fresh
echo "Restoring database from backup..."
if litestream restore -if-db-not-exists -config /app/litestream.yml "$DB_PATH" 2>&1; then
    echo "Database restored from backup"
else
    echo "No backup found - starting with fresh database"
fi

# Start the server with litestream replication
echo "Starting server with litestream replication..."
exec litestream replicate -exec "/app/server" -config /app/litestream.yml
