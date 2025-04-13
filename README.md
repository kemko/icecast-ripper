# Icecast Ripper

A lightweight Go application that automatically monitors Icecast audio streams, records them when they go live, and serves recordings via an RSS feed for podcast clients.

## Features

- **Smart Stream Detection**: Monitors Icecast streams and detects when they go live
- **Automatic Recording**: Records live streams to MP3 files with timestamps
- **Podcast-Ready RSS Feed**: Generates an RSS feed compatible with podcast clients
- **Web Server**: Built-in HTTP server for accessing recordings and RSS feed
- **Containerized**: Ready to run with Docker and Docker Compose
- **Configurable**: Easy configuration via environment variables

## Quick Start

### Using Docker

```bash
docker run -d \
  --name icecast-ripper \
  -p 8080:8080 \
  -e STREAM_URL=http://example.com:8000/stream \
  -v ./recordings:/recordings \
  ghcr.io/kemko/icecast-ripper:latest
```

### Using Docker Compose

```yaml
services:
  icecast-ripper:
    image: ghcr.io/kemko/icecast-ripper:latest
    ports:
      - "8080:8080"
    environment:
      - STREAM_URL=http://example.com:8000/stream
      - PUBLIC_URL=https://your-domain.com  # For RSS feed links
    volumes:
      - ./recordings:/recordings
```

### Running the Binary

Download the latest release and run:

```bash
export STREAM_URL=http://example.com:8000/stream
./icecast-ripper
```

## Configuration

Configure Icecast Ripper with these environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STREAM_URL` | URL of the Icecast stream to monitor | - | Yes |
| `CHECK_INTERVAL` | How often to check if the stream is live | 1m | No |
| `RECORDINGS_PATH` | Where to store recordings | ./recordings | No |
| `TEMP_PATH` | Where to store temporary files | /tmp | No |
| `BIND_ADDRESS` | HTTP server address:port | :8080 | No |
| `PUBLIC_URL` | Public URL for RSS feed links | <http://localhost:8080> | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info | No |

## Endpoints

- `GET /rss` - RSS feed of recordings (for podcast apps)
- `GET /recordings/` - Direct access to stored recordings

## Building From Source

Requires Go 1.22 or higher:

```bash
git clone https://github.com/kemko/icecast-ripper.git
cd icecast-ripper
make build
```

## How It Works

1. The application checks if the specified Icecast stream is live
2. When the stream is detected as live, recording begins
3. Recording continues until the stream ends or is interrupted
4. Recordings are saved with timestamps in the configured directory
5. The RSS feed is automatically updated with new recordings

## License

This project is licensed under the MIT License - see the LICENSE file for details.
