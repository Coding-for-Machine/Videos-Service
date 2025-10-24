FROM golang:1.24-alpine AS builder

# FFmpeg o'rnatish (video processing uchun)
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o youtube-clone ./cfm/main.go

# Final stage
FROM alpine:latest

# FFmpeg o'rnatish
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Binary va static fayllar
COPY --from=builder /app/youtube-clone .
COPY --from=builder /app/public ./public

# Papkalar yaratish
RUN mkdir -p uploads/videos uploads/thumbnails

EXPOSE 3000

CMD ["./youtube-clone"]