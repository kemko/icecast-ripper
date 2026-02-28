package rss

import (
	"testing"
	"time"
)

func TestGenerateGUID(t *testing.T) {
	ts := time.Date(2024, 9, 7, 19, 56, 22, 0, time.UTC)

	guid1 := generateGUID("stream.example.com", ts, "stream.example.com_20240907_195622.mp3")
	guid2 := generateGUID("stream.example.com", ts, "stream.example.com_20240907_195622.mp3")

	if guid1 != guid2 {
		t.Errorf("same inputs produced different GUIDs: %q vs %q", guid1, guid2)
	}

	guid3 := generateGUID("other.stream", ts, "stream.example.com_20240907_195622.mp3")
	if guid1 == guid3 {
		t.Error("different stream names should produce different GUIDs")
	}

	if len(guid1) != 64 {
		t.Errorf("GUID length = %d, want 64 (sha256 hex)", len(guid1))
	}
}
