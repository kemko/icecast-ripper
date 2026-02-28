package recorder

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"stream.example.com_20240907_195622.mp3", "stream.example.com_20240907_195622.mp3"},
		{"stream with spaces.mp3", "stream_with_spaces.mp3"},
		{"file:name.mp3", "file-name.mp3"},
		{"file<>|name.mp3", "file---name.mp3"},
		{"  _leading_trailing_.  ", "leading_trailing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func newTestRecorder(t *testing.T) (*Recorder, string) {
	t.Helper()
	dir := t.TempDir()
	return &Recorder{
		recordingsPath: dir,
		log:            slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}, dir
}

func createFile(t *testing.T, path string, age time.Duration) {
	t.Helper()
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if age > 0 {
		modTime := time.Now().Add(-age)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCleanOldRecordings(t *testing.T) {
	maxAge := 90 * 24 * time.Hour

	t.Run("deletes old mp3, keeps new mp3 and non-mp3", func(t *testing.T) {
		r, dir := newTestRecorder(t)

		createFile(t, filepath.Join(dir, "old.mp3"), 100*24*time.Hour)
		createFile(t, filepath.Join(dir, "new.mp3"), 0)
		createFile(t, filepath.Join(dir, "notes.txt"), 200*24*time.Hour)

		deleted, err := r.CleanOldRecordings(maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 1 {
			t.Errorf("deleted = %d, want 1", deleted)
		}

		assertFileNotExists(t, filepath.Join(dir, "old.mp3"))
		assertFileExists(t, filepath.Join(dir, "new.mp3"))
		assertFileExists(t, filepath.Join(dir, "notes.txt"))
	})

	t.Run("empty directory", func(t *testing.T) {
		r, _ := newTestRecorder(t)

		deleted, err := r.CleanOldRecordings(maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("deleted = %d, want 0", deleted)
		}
	})

	t.Run("all files old", func(t *testing.T) {
		r, dir := newTestRecorder(t)

		createFile(t, filepath.Join(dir, "a.mp3"), 100*24*time.Hour)
		createFile(t, filepath.Join(dir, "b.mp3"), 200*24*time.Hour)
		createFile(t, filepath.Join(dir, "c.mp3"), 365*24*time.Hour)

		deleted, err := r.CleanOldRecordings(maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 3 {
			t.Errorf("deleted = %d, want 3", deleted)
		}
	})

	t.Run("no files old enough", func(t *testing.T) {
		r, dir := newTestRecorder(t)

		createFile(t, filepath.Join(dir, "recent1.mp3"), 10*24*time.Hour)
		createFile(t, filepath.Join(dir, "recent2.mp3"), 89*24*time.Hour)

		deleted, err := r.CleanOldRecordings(maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("deleted = %d, want 0", deleted)
		}

		assertFileExists(t, filepath.Join(dir, "recent1.mp3"))
		assertFileExists(t, filepath.Join(dir, "recent2.mp3"))
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		r, dir := newTestRecorder(t)

		subDir := filepath.Join(dir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		createFile(t, filepath.Join(dir, "old.mp3"), 100*24*time.Hour)

		deleted, err := r.CleanOldRecordings(maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 1 {
			t.Errorf("deleted = %d, want 1", deleted)
		}

		if _, err := os.Stat(subDir); err != nil {
			t.Error("subdirectory should still exist")
		}
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		r := &Recorder{
			recordingsPath: "/nonexistent/path/to/recordings",
			log:            slog.New(slog.NewTextHandler(os.Stderr, nil)),
		}

		_, err := r.CleanOldRecordings(maxAge)
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %s", path)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted: %s", path)
	}
}
