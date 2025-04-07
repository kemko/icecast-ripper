# Icecast Ripper

A Go application that monitors and records Icecast audio streams. It detects when streams go live, automatically records the audio content, and generates an RSS feed of recordings.

## Features

- **Automatic Stream Monitoring**: Periodically checks if a stream is active
- **Intelligent Recording**: Records audio streams when they become active
- **RSS Feed Generation**: Provides an RSS feed of recorded streams
- **Web Interface**: Simple HTTP server for accessing recordings and RSS feed
- **Docker Support**: Run easily in containers with Docker and Docker Compose
- **Configurable**: Set recording paths, check intervals, and more via environment variables

## Installation

### Binary Installation

1. Download the latest release from the [GitHub releases page](https://github.com/kemko/icecast-ripper/releases)
2. Extract the binary to a location in your PATH
3. Run the binary with the required configuration (see Configuration section)

### Docker Installation

Pull the Docker image:

```bash
docker pull ghcr.io/kemko/icecast-ripper:master
```

Or use Docker Compose (see the Docker Compose section below).

### Building From Source

Requires Go 1.24 or higher.

```bash
git clone https://github.com/kemko/icecast-ripper.git
cd icecast-ripper
go build -o icecast-ripper ./cmd/icecast-ripper/main.go
```

## Configuration

Icecast Ripper is configured through environment variables:

| Environment Variable | Description | Default | Required |
|---------------------|-------------|---------|----------|
| `STREAM_URL` | URL of the Icecast stream to monitor | - | Yes |
| `CHECK_INTERVAL` | Interval between stream checks (e.g., 1m, 30s) | 1m | No |
| `RECORDINGS_PATH` | Path where recordings are stored | ./recordings | No |
| `TEMP_PATH` | Path for temporary files | ./temp | No |
| `SERVER_ADDRESS` | Address and port for the HTTP server | :8080 | No |
| `RSS_FEED_URL` | Public URL for the RSS feed | <http://localhost:8080/rss> | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info | No |

## Docker Compose

Create a `docker-compose.yml` file:

```yaml
---
services:
  icecast-ripper:
    image: ghcr.io/kemko/icecast-ripper:master
    ports:
      - "8080:8080"
    environment:
      - STREAM_URL=http://example.com/stream
      - CHECK_INTERVAL=1m
      - RECORDINGS_PATH=/records
      - TEMP_PATH=/app/temp
      - SERVER_ADDRESS=:8080
      - RSS_FEED_URL=http://localhost:8080/rss
      - LOG_LEVEL=info
    volumes:
      - ./records:/records
      - ./temp:/app/temp
      - ./data:/app/data
```

Run with:

```bash
docker-compose up -d
```

## Usage

1. Start the application with the required configuration
2. The application will monitor the stream at the specified interval
3. When the stream becomes active, recording starts automatically
4. Access the RSS feed at `http://localhost:8080/rss` (or the configured URL)
5. Access the recordings directly via the web interface

## API Endpoints

- `GET /` - Lists all recordings
- `GET /rss` - RSS feed of recordings
- `GET /recordings/{filename}` - Download a specific recording

## License

This project is licensed under the MIT License - see the LICENSE file for details.
