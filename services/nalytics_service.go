// services/analytics_service.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Coding-for-Machine/Videos-Service/models"
	"github.com/gocql/gocql"
)

type AnalyticsService struct {
	cassandra *gocql.Session
}

func NewAnalyticsService(cassandra *gocql.Session) *AnalyticsService {
	return &AnalyticsService{cassandra: cassandra}
}

// Soatlik statistikani to'plash
func (s *AnalyticsService) AggregateHourlyStats(ctx context.Context) error {
	// now := time.Now()
	// date := now.Format("2006-01-02")
	// hour := now.Hour()

	// query := `INSERT INTO video_analytics (video_id, date, hour, views, watch_time, likes, shares)
	// 	VALUES (?, ?, ?, ?, ?, ?, ?)`

	// Bu yerda real ma'lumotlarni to'plash kerak
	// Misol uchun placeholder
	return nil
}

// Trending videolarni yangilash
func (s *AnalyticsService) UpdateTrendingVideos(ctx context.Context) error {
	now := time.Now()
	timeBucket := now.Format("2006-01-02-15") // Soatlik bucket

	// Oxirgi 24 soat ichidagi eng ko'p ko'rilgan videolarni olish
	query := `SELECT id, title, thumbnail_url FROM videos LIMIT 100`
	iter := s.cassandra.Query(query).Iter()

	type VideoWithViews struct {
		ID           gocql.UUID
		Title        string
		ThumbnailURL string
		Views        int64
	}

	var videos []VideoWithViews
	var video VideoWithViews

	for iter.Scan(&video.ID, &video.Title, &video.ThumbnailURL) {
		// Views counterini olish
		viewQuery := "SELECT views FROM videos WHERE id = ?"
		s.cassandra.Query(viewQuery, video.ID).Scan(&video.Views)

		videos = append(videos, video)
		video = VideoWithViews{}
	}

	if err := iter.Close(); err != nil {
		return err
	}

	// Trending jadvaliga yozish
	for _, v := range videos {
		insertQuery := `INSERT INTO trending_videos (time_bucket, video_id, title, 
			thumbnail_url, created_at) VALUES (?, ?, ?, ?, ?)`
		s.cassandra.Query(insertQuery, timeBucket, v.ID, v.Title,
			v.ThumbnailURL, time.Now()).Exec()

		// Counter update
		updateQuery := `UPDATE trending_videos SET views = views + ? 
			WHERE time_bucket = ? AND video_id = ?`
		s.cassandra.Query(updateQuery, v.Views, timeBucket, v.ID).Exec()
	}

	return nil
}

// Trending videolarni olish
func (s *AnalyticsService) GetTrendingVideos(ctx context.Context, limit int) ([]models.Video, error) {
	now := time.Now()
	timeBucket := now.Format("2006-01-02-15")

	query := `SELECT video_id, title, thumbnail_url, created_at 
		FROM trending_videos WHERE time_bucket = ? LIMIT ?`
	iter := s.cassandra.Query(query, timeBucket, limit).Iter()

	var videos []models.Video
	var video models.Video

	for iter.Scan(&video.ID, &video.Title, &video.ThumbnailURL, &video.CreatedAt) {
		// Views counterini olish
		viewQuery := "SELECT views FROM trending_videos WHERE time_bucket = ? AND video_id = ?"
		s.cassandra.Query(viewQuery, timeBucket, video.ID).Scan(&video.Views)

		videos = append(videos, video)
		video = models.Video{}
	}

	return videos, iter.Close()
}

// Video uchun analytics
func (s *AnalyticsService) GetVideoAnalytics(ctx context.Context, videoID string, days int) ([]models.VideoAnalytics, error) {
	id, err := gocql.ParseUUID(videoID)
	if err != nil {
		return nil, fmt.Errorf("noto'g'ri video ID: %w", err)
	}

	startDate := time.Now().AddDate(0, 0, -days)

	query := `SELECT video_id, date, hour, views, watch_time, likes, shares 
		FROM video_analytics WHERE video_id = ? AND date >= ?`
	iter := s.cassandra.Query(query, id, startDate).Iter()

	var analytics []models.VideoAnalytics
	var stat models.VideoAnalytics

	for iter.Scan(&stat.VideoID, &stat.Date, &stat.Hour,
		&stat.Views, &stat.WatchTime, &stat.Likes, &stat.Shares) {
		analytics = append(analytics, stat)
		stat = models.VideoAnalytics{}
	}

	return analytics, iter.Close()
}
