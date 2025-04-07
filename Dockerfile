# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for static linking, especially with sqlite and alpine
# -ldflags="-w -s" strips debug information, reducing binary size
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /icecast-ripper ./cmd/icecast-ripper/main.go


# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /icecast-ripper /app/icecast-ripper

# Create directories for recordings, temp files, and database if needed inside the container
# These should ideally be mounted as volumes in production
RUN mkdir -p /app/recordings /app/temp

# Expose the server port
EXPOSE 8080

# Set default environment variables (can be overridden)
ENV RECORDINGS_PATH=/app/recordings
ENV TEMP_PATH=/app/temp
ENV SERVER_ADDRESS=:8080
ENV LOG_LEVEL=info
ENV CHECK_INTERVAL=1m
ENV RSS_FEED_URL=http://localhost:8080/rss
# ENV STREAM_URL= # Required: Must be set at runtime

# Command to run the application
ENTRYPOINT ["/app/icecast-ripper"]
