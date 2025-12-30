// Package database provides SQLite persistence for game state.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// New creates a new database connection.
// If the database file doesn't exist, it will be created.
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Limit concurrent connections to avoid lock contention
	conn.SetMaxOpenConns(1)

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	// Run migrations
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate runs all database migrations.
func (db *DB) migrate() error {
	// Create migrations table if not exists
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Run each migration
	for _, m := range migrations {
		applied, err := db.isMigrationApplied(m.id)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := db.runMigration(m); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.id, m.name, err)
		}
	}

	return nil
}

func (db *DB) isMigrationApplied(id int) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM migrations WHERE id = ?", id).Scan(&count)
	return count > 0, err
}

func (db *DB) runMigration(m migration) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.sql); err != nil {
		return err
	}

	if _, err := tx.Exec("INSERT INTO migrations (id, name) VALUES (?, ?)", m.id, m.name); err != nil {
		return err
	}

	return tx.Commit()
}

