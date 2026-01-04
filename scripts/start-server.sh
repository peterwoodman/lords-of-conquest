#!/bin/bash
set -e

DB_PATH="${DB_PATH:-/data/lords.db}"

# Ensure data directory exists
mkdir -p "$(dirname "$DB_PATH")"

# Restore database from replica if it exists (and local DB doesn't)
echo "Restoring database from backup..."
litestream restore -if-db-not-exists -config /app/litestream.yml "$DB_PATH"

# Start the server with litestream replication
echo "Starting server with litestream replication..."
exec litestream replicate -exec "/app/server" -config /app/litestream.yml
