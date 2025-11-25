package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
}

// NewDB creates a new database connection and runs migrations
func NewDB(dbPath string, migrationPath string, logger *slog.Logger) (*DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Use WAL mode for better concurrent access
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1", dbPath)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Connection pool settings
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)

	db := &DB{
		conn:   conn,
		logger: logger,
	}

	// Run migrations
	if err := db.migrate(migrationPath); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("database initialized", "path", dbPath)

	return db, nil
}

// migrate runs SQL migration files in order
func (db *DB) migrate(migrationPath string) error {
	files, err := os.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("reading migration directory: %w", err)
	}

	// Sort files by name to ensure order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationPath, file.Name()))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", file.Name(), err)
		}

		if _, err := db.conn.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", file.Name(), err)
		}

		db.logger.Info("applied migration", "file", file.Name())
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Ping checks if the database connection is alive
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}
