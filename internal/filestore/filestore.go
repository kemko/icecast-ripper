// Package filestore provides functionality for storing and retrieving metadata about recorded files
package filestore

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RecordedFile represents information about a recorded file
type RecordedFile struct {
	ID         int64         `json:"id"`
	Filename   string        `json:"filename"`
	Hash       string        `json:"hash"`
	FileSize   int64         `json:"fileSize"`
	Duration   time.Duration `json:"duration"`
	RecordedAt time.Time     `json:"recordedAt"`
}

// Store represents an in-memory store with optional file persistence
type Store struct {
	files     map[string]*RecordedFile
	storePath string
	mu        sync.RWMutex
}

// Init initializes a new Store
func Init(dataSourceName string) (*Store, error) {
	slog.Info("Initializing file store", "path", dataSourceName)

	fs := &Store{
		files:     make(map[string]*RecordedFile),
		storePath: dataSourceName,
	}

	if dataSourceName != "" {
		if err := fs.loadFromFile(); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to load file store data", "error", err)
		}
	}

	return fs, nil
}

func (fs *Store) loadFromFile() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.storePath == "" {
		return nil
	}

	data, err := os.ReadFile(fs.storePath)
	if err != nil {
		return err
	}

	var files []*RecordedFile
	if err := json.Unmarshal(data, &files); err != nil {
		return fmt.Errorf("failed to parse file store data: %w", err)
	}

	fs.files = make(map[string]*RecordedFile, len(files))
	for _, file := range files {
		fs.files[file.Filename] = file
	}

	slog.Info("Loaded file store data", "count", len(fs.files))
	return nil
}

func (fs *Store) saveToFile() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.storePath == "" {
		return nil
	}

	dir := filepath.Dir(fs.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for file store: %w", err)
	}

	files := make([]*RecordedFile, 0, len(fs.files))
	for _, file := range fs.files {
		files = append(files, file)
	}

	data, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file store data: %w", err)
	}

	if err := os.WriteFile(fs.storePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file store data: %w", err)
	}

	slog.Debug("Saved file store data", "count", len(files))
	return nil
}

// AddRecordedFile adds a file to the store
func (fs *Store) AddRecordedFile(filename, hash string, fileSize int64, duration time.Duration, recordedAt time.Time) (int64, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	id := time.Now().UnixNano()

	fs.files[filename] = &RecordedFile{
		ID:         id,
		Filename:   filename,
		Hash:       hash,
		FileSize:   fileSize,
		Duration:   duration,
		RecordedAt: recordedAt,
	}

	if err := fs.saveToFile(); err != nil {
		slog.Error("Failed to save file store data", "error", err)
	}

	return id, nil
}

// GetRecordedFiles retrieves all recorded files, ordered by recording date descending
func (fs *Store) GetRecordedFiles(limit int) ([]RecordedFile, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	files := make([]RecordedFile, 0, len(fs.files))
	for _, file := range fs.files {
		files = append(files, *file)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RecordedAt.After(files[j].RecordedAt)
	})

	if limit > 0 && limit < len(files) {
		files = files[:limit]
	}

	return files, nil
}

// Close ensures all data is persisted
func (fs *Store) Close() error {
	return fs.saveToFile()
}
