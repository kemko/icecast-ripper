package rss

import (
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/kemko/icecast-ripper/internal/mp3util"
)

type RecordingInfo struct {
	Filename   string
	Hash       string
	FileSize   int64
	Duration   time.Duration
	RecordedAt time.Time
}

type Generator struct {
	baseURL        string
	recordingsPath string
	feedTitle      string
	feedDesc       string
	streamName     string
	log            *slog.Logger
}

func New(publicURL, recordingsPath, title, description, streamName string, log *slog.Logger) *Generator {
	baseURL := publicURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &Generator{
		baseURL:        baseURL,
		recordingsPath: recordingsPath,
		feedTitle:      title,
		feedDesc:       description,
		streamName:     streamName,
		log:            log,
	}
}

var recordingPattern = regexp.MustCompile(`([^_]+)_(\d{8}_\d{6})\.mp3$`)

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

	g.log.Debug("RSS feed generated", "itemCount", len(feed.Items))
	return []byte(rssFeed), nil
}

func (g *Generator) createFeedItems(recordings []RecordingInfo) []*feeds.Item {
	items := make([]*feeds.Item, 0, len(recordings))
	baseURL := strings.TrimSuffix(g.baseURL, "/")

	for _, rec := range recordings {
		fileURL := fmt.Sprintf("%s/recordings/%s", baseURL, rec.Filename)
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

func (g *Generator) scanRecordings(maxItems int) ([]RecordingInfo, error) {
	var recordings []RecordingInfo

	err := filepath.WalkDir(g.recordingsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".mp3") {
			return err
		}

		matches := recordingPattern.FindStringSubmatch(d.Name())
		if len(matches) < 3 {
			g.log.Debug("Skipping non-conforming filename", "filename", d.Name())
			return nil
		}

		timestamp, err := time.Parse("20060102_150405", matches[2])
		if err != nil {
			g.log.Warn("Failed to parse timestamp from filename", "filename", d.Name(), "error", err)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			g.log.Warn("Failed to get file info", "filename", d.Name(), "error", err)
			return nil
		}

		duration, err := mp3util.GetDuration(path)
		if err != nil {
			g.log.Warn("Failed to get MP3 duration, estimating", "filename", d.Name(), "error", err)
			// ~128kbps = 16KB per second
			duration = time.Duration(info.Size()/16000) * time.Second
		}

		filename := filepath.Base(path)
		fileHash := generateGUID(g.streamName, timestamp, filename)

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

	sort.Slice(recordings, func(i, j int) bool {
		return recordings[i].RecordedAt.After(recordings[j].RecordedAt)
	})

	if maxItems > 0 && maxItems < len(recordings) {
		recordings = recordings[:maxItems]
	}

	return recordings, nil
}

func generateGUID(streamName string, recordedAt time.Time, filePath string) string {
	filename := filepath.Base(filePath)
	input := fmt.Sprintf("%s:%s:%s", streamName, recordedAt.UTC().Format(time.RFC3339), filename)
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
