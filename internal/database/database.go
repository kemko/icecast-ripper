package database

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

// FileStore represents an in-memory store with optional file persistence
type FileStore struct {
	files     map[string]*RecordedFile // Map of filename -> file data
	storePath string                   // Path to persistence file, if empty no persistence
	mu        sync.RWMutex             // Mutex for thread safety
}

// RecordedFile represents information about a recorded file.
type RecordedFile struct {
	ID         int64         `json:"id"`
	Filename   string        `json:"filename"`
	Hash       string        `json:"hash"` // GUID for the file
	FileSize   int64         `json:"fileSize"`
	Duration   time.Duration `json:"duration"`
	RecordedAt time.Time     `json:"recordedAt"`
}

// InitDB initializes a new FileStore.
// If dataSourceName is provided, it will be used as the path to persist the store.
func InitDB(dataSourceName string) (*FileStore, error) {
	slog.Info("Initializing file store", "path", dataSourceName)

	fs := &FileStore{
		files:     make(map[string]*RecordedFile),
		storePath: dataSourceName,
	}

	// If a data source is provided, try to load existing data
	if dataSourceName != "" {
		if err := fs.loadFromFile(); err != nil {
			// If file doesn't exist, that's fine - we'll create it when we save
			if !os.IsNotExist(err) {
				slog.Error("Failed to load file store data", "error", err)
			}
		}
	}

	slog.Info("File store initialized successfully")
	return fs, nil
}

// loadFromFile loads the file store data from the persistence file.
func (fs *FileStore) loadFromFile() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.storePath == "" {
		return nil // No persistence path set
	}

	data, err := os.ReadFile(fs.storePath)
	if err != nil {
		return err
	}

	var files []*RecordedFile
	if err := json.Unmarshal(data, &files); err != nil {
		return fmt.Errorf("failed to parse file store data: %w", err)
	}

	// Reset the map and repopulate it
	fs.files = make(map[string]*RecordedFile, len(files))
	for _, file := range files {
		fs.files[file.Filename] = file
	}

	slog.Info("Loaded file store data", "count", len(fs.files))
	return nil
}

// saveToFile persists the file store data to disk.
func (fs *FileStore) saveToFile() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.storePath == "" {
		return nil // No persistence path set
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fs.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for file store: %w", err)
	}

	// Convert map to slice
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

// AddRecordedFile adds a file to the store.
func (fs *FileStore) AddRecordedFile(filename, hash string, fileSize int64, duration time.Duration, recordedAt time.Time) (int64, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Generate a unique ID (just use timestamp as we don't need real DB IDs)
	id := time.Now().UnixNano()

	// Store the file
	fs.files[filename] = &RecordedFile{
		ID:         id,
		Filename:   filename,
		Hash:       hash,
		FileSize:   fileSize,
		Duration:   duration,
		RecordedAt: recordedAt,
	}

	// Persist changes
	if err := fs.saveToFile(); err != nil {
		slog.Error("Failed to save file store data", "error", err)
	}

	slog.Debug("Added recorded file to store", "id", id, "filename", filename, "hash", hash)
	return id, nil
}

// GetRecordedFiles retrieves all recorded files, ordered by recording date descending.
func (fs *FileStore) GetRecordedFiles(limit int) ([]RecordedFile, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Convert map to slice for sorting
	files := make([]RecordedFile, 0, len(fs.files))
	for _, file := range fs.files {
		files = append(files, *file)
	}

	// Sort by recorded_at descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].RecordedAt.After(files[j].RecordedAt)
	})

	// Apply limit if provided
	if limit > 0 && limit < len(files) {
		files = files[:limit]
	}

	return files, nil
}

// Close closes the file store and ensures all data is persisted.
func (fs *FileStore) Close() error {
	slog.Info("Closing file store")
	return fs.saveToFile()
}
