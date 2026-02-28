FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION}" -o /icecast-ripper ./cmd/icecast-ripper

FROM alpine:latest

WORKDIR /app
COPY --from=builder /icecast-ripper /app/icecast-ripper

RUN mkdir -p /app/recordings /app/temp

EXPOSE 8080

ENV RECORDINGS_PATH=/app/recordings
ENV TEMP_PATH=/app/temp
ENV BIND_ADDRESS=:8080
ENV LOG_LEVEL=info
ENV CHECK_INTERVAL=1m
ENV PUBLIC_URL=http://localhost:8080
ENV RETENTION_DAYS=90

ENTRYPOINT ["/app/icecast-ripper"]
