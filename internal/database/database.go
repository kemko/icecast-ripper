package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// DB represents the database connection.
type DB struct {
	*sql.DB
}

// RecordedFile represents a row in the recorded_files table.
type RecordedFile struct {
	ID         int64
	Filename   string
	Hash       string
	FileSize   int64         // Size in bytes
	Duration   time.Duration // Duration of the recording
	RecordedAt time.Time
}

// InitDB initializes the SQLite database connection and creates tables if they don't exist.
func InitDB(dataSourceName string) (*DB, error) {
	slog.Info("Initializing database connection", "path", dataSourceName)
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create table if not exists
	query := `
	CREATE TABLE IF NOT EXISTS recorded_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL UNIQUE,
		hash TEXT NOT NULL UNIQUE,
		filesize INTEGER NOT NULL,
		duration INTEGER NOT NULL, -- Store duration in seconds or milliseconds
		recorded_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_recorded_at ON recorded_files (recorded_at);
	`
	if _, err = db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	slog.Info("Database initialized successfully")
	return &DB{db}, nil
}

// AddRecordedFile inserts a new record into the database.
func (db *DB) AddRecordedFile(filename, hash string, fileSize int64, duration time.Duration, recordedAt time.Time) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO recorded_files (filename, hash, filesize, duration, recorded_at) VALUES (?, ?, ?, ?, ?)",
		filename, hash, fileSize, int64(duration.Seconds()), recordedAt, // Store duration as seconds
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert recorded file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	slog.Debug("Added recorded file to database", "id", id, "filename", filename, "hash", hash)
	return id, nil
}

// GetRecordedFiles retrieves all recorded files, ordered by recording date descending.
func (db *DB) GetRecordedFiles(limit int) ([]RecordedFile, error) {
	query := "SELECT id, filename, hash, filesize, duration, recorded_at FROM recorded_files ORDER BY recorded_at DESC"
	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recorded files: %w", err)
	}
	defer rows.Close()

	var files []RecordedFile
	for rows.Next() {
		var rf RecordedFile
		var durationSeconds int64
		if err := rows.Scan(&rf.ID, &rf.Filename, &rf.Hash, &rf.FileSize, &durationSeconds, &rf.RecordedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recorded file row: %w", err)
		}
		rf.Duration = time.Duration(durationSeconds) * time.Second
		files = append(files, rf)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over recorded file rows: %w", err)
	}

	return files, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	slog.Info("Closing database connection")
	return db.DB.Close()
}
