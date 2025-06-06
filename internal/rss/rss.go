package rss

import (
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/hash"
	"github.com/kemko/icecast-ripper/internal/mp3util"
)

// RecordingInfo contains metadata about a recording
type RecordingInfo struct {
	Filename   string
	Hash       string
	FileSize   int64
	Duration   time.Duration
	RecordedAt time.Time
}

// Generator creates RSS feeds for recorded streams
type Generator struct {
	baseURL        string
	recordingsPath string
	feedTitle      string
	feedDesc       string
	streamName     string
}

// New creates a new RSS Generator instance
func New(cfg *config.Config, title, description, streamName string) *Generator {
	baseURL := cfg.PublicURL

	// Ensure base URL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &Generator{
		baseURL:        baseURL,
		recordingsPath: cfg.RecordingsPath,
		feedTitle:      title,
		feedDesc:       description,
		streamName:     streamName,
	}
}

// Pattern to extract timestamp from recording filename (stream.somesite.com_20240907_195622.mp3)
var recordingPattern = regexp.MustCompile(`([^_]+)_(\d{8}_\d{6})\.mp3$`)

// GenerateFeed produces the RSS feed XML
func (g *Generator) GenerateFeed(maxItems int) ([]byte, error) {
	recordings, err := g.scanRecordings(maxItems)
	if err != nil {
		return nil, fmt.Errorf("failed to scan recordings: %w", err)
	}

	feed := &feeds.Feed{
		Title:       g.feedTitle,
		Link:        &feeds.Link{Href: g.baseURL},
		Description: g.feedDesc,
		Created:     time.Now(),
	}

	feed.Items = g.createFeedItems(recordings)

	rssFeed, err := feed.ToRss()
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	slog.Debug("RSS feed generated", "itemCount", len(feed.Items))
	return []byte(rssFeed), nil
}

// createFeedItems converts recording info to RSS feed items
func (g *Generator) createFeedItems(recordings []RecordingInfo) []*feeds.Item {
	items := make([]*feeds.Item, 0, len(recordings))

	baseURL := strings.TrimSuffix(g.baseURL, "/")

	for _, rec := range recordings {
		fileURL := fmt.Sprintf("%s/recordings/%s", baseURL, rec.Filename)

		// Round duration to the nearest second
		roundedDuration := time.Duration(math.Round(float64(rec.Duration)/float64(time.Second))) * time.Second

		item := &feeds.Item{
			Title: fmt.Sprintf("Recording %s", rec.RecordedAt.Format("2006-01-02 15:04")),
			Link:  &feeds.Link{Href: fileURL},
			Description: fmt.Sprintf("Icecast stream recording from %s. Duration: %s",
				rec.RecordedAt.Format(time.RFC1123), roundedDuration.String()),
			Created: rec.RecordedAt,
			Id:      rec.Hash,
			Enclosure: &feeds.Enclosure{
				Url:    fileURL,
				Length: fmt.Sprintf("%d", rec.FileSize),
				Type:   "audio/mpeg",
			},
		}
		items = append(items, item)
	}

	return items
}

// scanRecordings scans the recordings directory and returns metadata
func (g *Generator) scanRecordings(maxItems int) ([]RecordingInfo, error) {
	var recordings []RecordingInfo

	err := filepath.WalkDir(g.recordingsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".mp3") {
			return err
		}

		// Extract timestamp from filename
		matches := recordingPattern.FindStringSubmatch(d.Name())
		if len(matches) < 3 {
			slog.Debug("Skipping non-conforming filename", "filename", d.Name())
			return nil
		}

		// Parse the timestamp from the filename
		timestamp, err := time.Parse("20060102_150405", matches[2])
		if err != nil {
			slog.Warn("Failed to parse timestamp from filename", "filename", d.Name(), "error", err)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			slog.Warn("Failed to get file info", "filename", d.Name(), "error", err)
			return nil
		}

		// Get the actual duration from the MP3 file
		duration, err := mp3util.GetDuration(path)
		if err != nil {
			slog.Warn("Failed to get MP3 duration, estimating", "filename", d.Name(), "error", err)
			// Estimate: ~128kbps MP3 bitrate = 16KB per second
			duration = time.Duration(info.Size()/16000) * time.Second
		}

		// Generate a stable hash for the recording
		filename := filepath.Base(path)
		fileHash := hash.GenerateGUID(g.streamName, timestamp, filename)

		recordings = append(recordings, RecordingInfo{
			Filename:   filename,
			Hash:       fileHash,
			FileSize:   info.Size(),
			Duration:   duration,
			RecordedAt: timestamp,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk recordings directory: %w", err)
	}

	// Sort recordings by timestamp (newest first)
	sort.Slice(recordings, func(i, j int) bool {
		return recordings[i].RecordedAt.After(recordings[j].RecordedAt)
	})

	// Limit number of items if specified
	if maxItems > 0 && maxItems < len(recordings) {
		recordings = recordings[:maxItems]
	}

	return recordings, nil
}
