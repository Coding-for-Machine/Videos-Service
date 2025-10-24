// models/video.go
package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Video struct {
	ID              gocql.UUID        `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	UserID          gocql.UUID        `json:"user_id"`
	Username        string            `json:"username"`
	FileName        string            `json:"file_name"`
	FileSize        int64             `json:"file_size"`
	Duration        int               `json:"duration"`
	ThumbnailURL    string            `json:"thumbnail_url"`
	VideoURL        string            `json:"video_url"`
	Status          string            `json:"status"` // uploading, processing, ready, failed
	QualityVersions map[string]string `json:"quality_versions"`
	Views           int64             `json:"views"`
	Likes           int64             `json:"likes"`
	Dislikes        int64             `json:"dislikes"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type VideoUploadRequest struct {
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
	UserID      string `json:"user_id" form:"user_id"`
	Username    string `json:"username" form:"username"`
}

type ProcessingJob struct {
	JobID        gocql.UUID `json:"job_id"`
	VideoID      gocql.UUID `json:"video_id"`
	JobType      string     `json:"job_type"` // transcode, thumbnail, extract_audio
	Status       string     `json:"status"`   // pending, processing, completed, failed
	Priority     int        `json:"priority"`
	RetryCount   int        `json:"retry_count"`
	ErrorMessage string     `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Comment struct {
	VideoID   gocql.UUID `json:"video_id"`
	CommentID gocql.UUID `json:"comment_id"`
	UserID    gocql.UUID `json:"user_id"`
	Username  string     `json:"username"`
	Text      string     `json:"text"`
	Likes     int64      `json:"likes"`
	CreatedAt time.Time  `json:"created_at"`
}

type VideoAnalytics struct {
	VideoID   gocql.UUID `json:"video_id"`
	Date      time.Time  `json:"date"`
	Hour      int        `json:"hour"`
	Views     int64      `json:"views"`
	WatchTime int64      `json:"watch_time"` // seconds
	Likes     int64      `json:"likes"`
	Shares    int64      `json:"shares"`
}
