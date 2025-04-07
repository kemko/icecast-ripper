package rss

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/database"
)

// Generator creates RSS feeds
type Generator struct {
	fileStore      *database.FileStore
	feedBaseURL    string
	recordingsPath string
	feedTitle      string
	feedDesc       string
}

// New creates a new RSS Generator instance
func New(fileStore *database.FileStore, cfg *config.Config, title, description string) *Generator {
	baseURL := cfg.RSSFeedURL
	if baseURL == "" {
		slog.Warn("RSS_FEED_URL not set, using default")
		baseURL = "http://localhost:8080/recordings/"
	}

	// Ensure base URL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
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

// GenerateFeed produces the RSS feed XML as a byte slice
func (g *Generator) GenerateFeed(maxItems int) ([]byte, error) {
	recordings, err := g.fileStore.GetRecordedFiles(maxItems)
	if err != nil {
		return nil, fmt.Errorf("failed to get recorded files: %w", err)
	}

	feed := &feeds.Feed{
		Title:       g.feedTitle,
		Link:        &feeds.Link{Href: g.feedBaseURL},
		Description: g.feedDesc,
		Created:     time.Now(),
	}

	feed.Items = make([]*feeds.Item, 0, len(recordings))

	for _, rec := range recordings {
		fileURL, err := url.JoinPath(g.feedBaseURL, rec.Filename)
		if err != nil {
			slog.Error("Failed to create file URL", "filename", rec.Filename, "error", err)
			continue
		}

		item := &feeds.Item{
			Title:       fmt.Sprintf("Recording %s", rec.RecordedAt.Format("2006-01-02 15:04")),
			Link:        &feeds.Link{Href: fileURL},
			Description: fmt.Sprintf("Icecast stream recording from %s. Duration: %s",
			              rec.RecordedAt.Format(time.RFC1123), rec.Duration.String()),
			Created:     rec.RecordedAt,
			Id:          rec.Hash,
			Enclosure:   &feeds.Enclosure{
				Url:    fileURL,
				Length: fmt.Sprintf("%d", rec.FileSize), // Convert int64 to string
				Type:   "audio/mpeg",
			},
		}
		feed.Items = append(feed.Items, item)
	}

	rssFeed, err := feed.ToRss()
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	slog.Debug("RSS feed generated", "itemCount", len(feed.Items))
	return []byte(rssFeed), nil
}
