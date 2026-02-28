package main

import "testing"

func TestExtractStreamName(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"http://stream.example.com/live", "stream.example.com_live"},
		{"http://radio.fm:8000/stream.mp3", "radio.fm_stream.mp3"},
		{"http://example.com/", "example.com"},
		{"http://example.com", "example.com"},
		{"http://example.com/path/to/stream", "example.com_path_to_stream"},
		{"://invalid", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractStreamName(tt.url)
			if got != tt.want {
				t.Errorf("extractStreamName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
