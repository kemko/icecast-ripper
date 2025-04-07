package rss

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/gorilla/feeds"
	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/database"
)

// Generator creates RSS feeds.
type Generator struct {
	fileStore      *database.FileStore
	feedBaseURL    string // Base URL for links in the feed (e.g., http://server.com/recordings/)
	recordingsPath string // Local path to recordings (needed for file info, maybe not directly used in feed)
	feedTitle      string
	feedDesc       string
}

// New creates a new RSS Generator instance.
func New(fileStore *database.FileStore, cfg *config.Config, title, description string) *Generator {
	// Ensure the base URL for recordings ends with a slash
	baseURL := cfg.RSSFeedURL // This should be the URL base for *serving* files
	if baseURL == "" {
		slog.Warn("RSS_FEED_URL not set, RSS links might be incomplete. Using placeholder.")
		baseURL = "http://localhost:8080/recordings/" // Placeholder
	}
	if baseURL[len(baseURL)-1:] != "/" {
		baseURL += "/"
	}

	return &Generator{
		fileStore:      fileStore,
		feedBaseURL:    baseURL,
		recordingsPath: cfg.RecordingsPath,
		feedTitle:      title,
		feedDesc:       description,
	}
}

// GenerateFeed fetches recordings and produces the RSS feed XML as a byte slice.
func (g *Generator) GenerateFeed(maxItems int) ([]byte, error) {
	slog.Debug("Generating RSS feed", "maxItems", maxItems)
	recordings, err := g.fileStore.GetRecordedFiles(maxItems)
	if err != nil {
		return nil, fmt.Errorf("failed to get recorded files for RSS feed: %w", err)
	}

	now := time.Now()
	feed := &feeds.Feed{
		Title:       g.feedTitle,
		Link:        &feeds.Link{Href: g.feedBaseURL},
		Description: g.feedDesc,
		Created:     now,
	}

	for _, rec := range recordings {
		fileURL, err := url.JoinPath(g.feedBaseURL, rec.Filename)
		if err != nil {
			slog.Error("Failed to create file URL for RSS item", "filename", rec.Filename, "error", err)
			continue // Skip this item if URL creation fails
		}

		item := &feeds.Item{
			Title:       fmt.Sprintf("Recording %s", rec.RecordedAt.Format("2006-01-02 15:04")),
			Link:        &feeds.Link{Href: fileURL},
			Description: fmt.Sprintf("Icecast stream recording from %s. Duration: %s.", rec.RecordedAt.Format(time.RFC1123), rec.Duration.String()),
			Created:     rec.RecordedAt,
			Id:          rec.Hash,
			Enclosure:   &feeds.Enclosure{Url: fileURL, Length: rec.FileSize, Type: "audio/mpeg"},
		}
		feed.Items = append(feed.Items, item)
	}

	// Create RSS 2.0 feed
	rssFeed, err := feed.ToRss()
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	slog.Debug("RSS feed generated successfully", "item_count", len(feed.Items))
	return []byte(rssFeed), nil
}
