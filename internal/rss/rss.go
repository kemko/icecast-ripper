package rss

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/database"
)

// Structs for RSS 2.0 feed generation
// Based on https://validator.w3.org/feed/docs/rss2.html

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	XMLName       xml.Name `xml:"channel"`
	Title         string   `xml:"title"`
	Link          string   `xml:"link"` // Link to the website/source
	Description   string   `xml:"description"`
	LastBuildDate string   `xml:"lastBuildDate,omitempty"` // RFC1123Z format
	Items         []Item   `xml:"item"`
}

type Item struct {
	XMLName     xml.Name  `xml:"item"`
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`        // Link to the specific recording file
	Description string    `xml:"description"` // Can include duration, size etc.
	PubDate     string    `xml:"pubDate"`     // RFC1123Z format of recording time
	GUID        GUID      `xml:"guid"`        // Unique identifier (using metadata-based GUID)
	Enclosure   Enclosure `xml:"enclosure"`   // Describes the media file
}

// GUID needs IsPermaLink attribute
type GUID struct {
	XMLName     xml.Name `xml:"guid"`
	IsPermaLink bool     `xml:"isPermaLink,attr"`
	Value       string   `xml:",chardata"`
}

// Enclosure describes the media file
type Enclosure struct {
	XMLName xml.Name `xml:"enclosure"`
	URL     string   `xml:"url,attr"`
	Length  int64    `xml:"length,attr"`
	Type    string   `xml:"type,attr"` // MIME type (e.g., "audio/mpeg")
}

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

	items := make([]Item, 0, len(recordings))
	for _, rec := range recordings {
		fileURL, err := url.JoinPath(g.feedBaseURL, rec.Filename)
		if err != nil {
			slog.Error("Failed to create file URL for RSS item", "filename", rec.Filename, "error", err)
			continue // Skip this item if URL creation fails
		}

		item := Item{
			Title:       fmt.Sprintf("Recording %s", rec.RecordedAt.Format("2006-01-02 15:04")), // Example title
			Link:        fileURL,
			Description: fmt.Sprintf("Icecast stream recording from %s. Duration: %s.", rec.RecordedAt.Format(time.RFC1123), rec.Duration.String()),
			PubDate:     rec.RecordedAt.Format(time.RFC1123Z), // Use RFC1123Z for pubDate
			GUID: GUID{
				IsPermaLink: false, // The guid itself is not a permalink URL
				Value:       rec.Hash,
			},
			Enclosure: Enclosure{
				URL:    fileURL,
				Length: rec.FileSize,
				Type:   "audio/mpeg", // Assuming MP3, adjust if format varies or is detectable
			},
		}
		items = append(items, item)
	}

	feed := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:         g.feedTitle,
			Link:          g.feedBaseURL, // Link to the base URL or a relevant page
			Description:   g.feedDesc,
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	xmlBytes, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RSS feed to XML: %w", err)
	}

	// Add the standard XML header
	output := append([]byte(xml.Header), xmlBytes...)

	slog.Debug("RSS feed generated successfully", "item_count", len(items))
	return output, nil
}
