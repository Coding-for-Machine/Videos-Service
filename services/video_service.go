// services/video_service.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Coding-for-Machine/Videos-Service/models"

	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type VideoService struct {
	cassandra *gocql.Session
	minio     *minio.Client
	redis     *redis.Client
}

func NewVideoService(cassandra *gocql.Session, minio *minio.Client, redis *redis.Client) *VideoService {
	return &VideoService{
		cassandra: cassandra,
		minio:     minio,
		redis:     redis,
	}
}

func (s *VideoService) UploadVideo(ctx context.Context, title, description, username string, file io.Reader, fileSize int64, fileName string) (*models.Video, error) {
	videoID := gocql.TimeUUID()
	userID := gocql.TimeUUID() // Haqiqiy user authentication kerak

	// MinIOga yuklash (raw bucket)
	objectName := fmt.Sprintf("raw/%s-%s", videoID.String(), fileName)
	_, err := s.minio.PutObject(ctx, "videos-raw", objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return nil, fmt.Errorf("MinIOga yuklash xatosi: %w", err)
	}

	// Video ma'lumotlarini Cassandraga saqlash
	video := &models.Video{
		ID:              videoID,
		Title:           title,
		Description:     description,
		UserID:          userID,
		Username:        username,
		FileName:        fileName,
		FileSize:        fileSize,
		Status:          "processing",
		QualityVersions: make(map[string]string),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	query := `INSERT INTO videos (id, title, description, user_id, username, file_name, 
		file_size, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err = s.cassandra.Query(query, video.ID, video.Title, video.Description,
		video.UserID, video.Username, video.FileName, video.FileSize,
		video.Status, video.CreatedAt, video.UpdatedAt).Exec()
	if err != nil {
		return nil, fmt.Errorf("Cassandraga saqlash xatosi: %w", err)
	}

	// Processing joblarni Redis queuega qo'shish
	jobs := []models.ProcessingJob{
		{
			JobID:     gocql.TimeUUID(),
			VideoID:   videoID,
			JobType:   "transcode",
			Status:    "pending",
			Priority:  1,
			CreatedAt: time.Now(),
		},
		{
			JobID:     gocql.TimeUUID(),
			VideoID:   videoID,
			JobType:   "thumbnail",
			Status:    "pending",
			Priority:  2,
			CreatedAt: time.Now(),
		},
	}

	for _, job := range jobs {
		jobData, _ := json.Marshal(job)
		s.redis.RPush(ctx, "processing_queue", jobData)
	}

	return video, nil
}

func (s *VideoService) GetVideos(ctx context.Context, limit int) ([]models.Video, error) {
	query := "SELECT id, title, description, username, thumbnail_url, video_url, duration, created_at FROM videos LIMIT ?"
	iter := s.cassandra.Query(query, limit).Iter()

	var videos []models.Video
	var video models.Video

	for iter.Scan(&video.ID, &video.Title, &video.Description, &video.Username,
		&video.ThumbnailURL, &video.VideoURL, &video.Duration, &video.CreatedAt) {

		// Views counterini olish
		viewQuery := "SELECT views FROM videos WHERE id = ?"
		s.cassandra.Query(viewQuery, video.ID).Scan(&video.Views)

		videos = append(videos, video)
		video = models.Video{}
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return videos, nil
}

func (s *VideoService) GetVideo(ctx context.Context, videoID string) (*models.Video, error) {
	id, err := gocql.ParseUUID(videoID)
	if err != nil {
		return nil, fmt.Errorf("noto'g'ri video ID: %w", err)
	}

	var video models.Video
	query := `SELECT id, title, description, username, file_name, file_size, 
		duration, thumbnail_url, video_url, status, created_at, updated_at 
		FROM videos WHERE id = ?`

	err = s.cassandra.Query(query, id).Scan(
		&video.ID, &video.Title, &video.Description, &video.Username,
		&video.FileName, &video.FileSize, &video.Duration,
		&video.ThumbnailURL, &video.VideoURL, &video.Status,
		&video.CreatedAt, &video.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("video topilmadi: %w", err)
	}

	// Views counterini olish
	viewQuery := "SELECT views FROM videos WHERE id = ?"
	s.cassandra.Query(viewQuery, video.ID).Scan(&video.Views)

	return &video, nil
}

func (s *VideoService) IncrementView(ctx context.Context, videoID string) error {
	id, err := gocql.ParseUUID(videoID)
	if err != nil {
		return err
	}

	// Redis queuega qo'shish (batch processing uchun)
	s.redis.RPush(ctx, "view_queue", videoID)

	// Cassandra counterini oshirish
	query := "UPDATE videos SET views = views + 1 WHERE id = ?"
	return s.cassandra.Query(query, id).Exec()
}

func (s *VideoService) DeleteVideo(ctx context.Context, videoID string) error {
	id, err := gocql.ParseUUID(videoID)
	if err != nil {
		return err
	}

	// Video ma'lumotlarini olish
	video, err := s.GetVideo(ctx, videoID)
	if err != nil {
		return err
	}

	// MinIOdan fayllarni o'chirish
	objectName := fmt.Sprintf("raw/%s-%s", video.ID.String(), video.FileName)
	s.minio.RemoveObject(ctx, "videos-raw", objectName, minio.RemoveObjectOptions{})

	// Cassandradan o'chirish
	query := "DELETE FROM videos WHERE id = ?"
	return s.cassandra.Query(query, id).Exec()
}

func (s *VideoService) UpdateVideoStatus(ctx context.Context, videoID gocql.UUID, status, videoURL, thumbnailURL string) error {
	query := `UPDATE videos SET status = ?, video_url = ?, thumbnail_url = ?, 
		updated_at = ? WHERE id = ?`
	return s.cassandra.Query(query, status, videoURL, thumbnailURL, time.Now(), videoID).Exec()
}

func (s *VideoService) SearchVideos(ctx context.Context, keyword string, limit int) ([]models.Video, error) {
	// Simple search (production uchun Elasticsearch kerak)
	query := "SELECT video_id, title, thumbnail_url FROM video_search WHERE keyword = ? LIMIT ?"
	iter := s.cassandra.Query(query, keyword, limit).Iter()

	var videos []models.Video
	var video models.Video

	for iter.Scan(&video.ID, &video.Title, &video.ThumbnailURL) {
		videos = append(videos, video)
		video = models.Video{}
	}

	return videos, iter.Close()
}
