package main

import (
	"database/sql"
	"fmt"
)

// initSchema creates all database tables and indexes if they don't exist.
// This function contains the FINAL schema state - no migration history or versioning.
// Safe to call multiple times (idempotent) due to "IF NOT EXISTS" clauses.
func initSchema(db *sql.DB) error {
	// Create all tables
	tables := []string{
		// Users table - authentication and authorization
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			readonly BOOLEAN DEFAULT FALSE,
			is_superuser BOOLEAN DEFAULT FALSE,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Hosts table - Wake-on-LAN target devices
		`CREATE TABLE IF NOT EXISTS hosts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			mac TEXT NOT NULL,
			broadcast TEXT NOT NULL,
			interface TEXT,
			static_ip TEXT,
			use_as_fallback BOOLEAN DEFAULT FALSE,
			user_id TEXT,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Sessions table - user authentication sessions
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires DATETIME NOT NULL,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range tables {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_hosts_user_id ON hosts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires)`,
	}

	for _, query := range indexes {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
