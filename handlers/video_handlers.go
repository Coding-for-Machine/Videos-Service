// handlers/video_handlers.go
package handlers

import (
	"io"

	"github.com/Coding-for-Machine/Videos-Service/models"
	"github.com/Coding-for-Machine/Videos-Service/services"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

func UploadVideo(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Form ma'lumotlarini olish
		var req models.VideoUploadRequest
		if err := c.BodyParser(&req); err != nil {
			req.Title = c.FormValue("title")
			req.Description = c.FormValue("description")
			req.Username = c.FormValue("username", "Anonymous")
		}

		if req.Title == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Title kerak",
			})
		}

		// Video faylni olish
		file, err := c.FormFile("video")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Video fayl kerak",
			})
		}

		// Fayl turini tekshirish
		contentType := file.Header.Get("Content-Type")
		if contentType != "video/mp4" && contentType != "video/webm" &&
			contentType != "video/quicktime" && contentType != "video/x-msvideo" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Faqat video fayllar ruxsat etilgan",
			})
		}

		// Faylni ochish
		fileData, err := file.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Fayl ochilmadi",
			})
		}
		defer fileData.Close()

		// Video yuklash
		video, err := videoService.UploadVideo(
			c.Context(),
			req.Title,
			req.Description,
			req.Username,
			fileData,
			file.Size,
			file.Filename,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(201).JSON(fiber.Map{
			"message": "Video yuklandi va processing boshlandi",
			"video":   video,
		})
	}
}

func GetVideos(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 20)

		videos, err := videoService.GetVideos(c.Context(), limit)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"videos": videos,
			"total":  len(videos),
		})
	}
}

func GetVideo(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		videoID := c.Params("id")

		video, err := videoService.GetVideo(c.Context(), videoID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Video topilmadi",
			})
		}

		return c.JSON(video)
	}
}

func DeleteVideo(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		videoID := c.Params("id")

		if err := videoService.DeleteVideo(c.Context(), videoID); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "Video o'chirildi",
		})
	}
}

func IncrementView(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		videoID := c.Params("id")

		if err := videoService.IncrementView(c.Context(), videoID); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "View qo'shildi",
		})
	}
}

func StreamVideo(videoService *services.VideoService, minioClient *minio.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		videoID := c.Params("id")
		quality := c.Query("quality", "720p")

		video, err := videoService.GetVideo(c.Context(), videoID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Video topilmadi",
			})
		}

		// MinIOdan videoni stream qilish
		objectPath := video.VideoURL
		if quality != "" {
			objectPath = video.QualityVersions[quality]
		}

		object, err := minioClient.GetObject(c.Context(), "videos-processed", objectPath, minio.GetObjectOptions{})
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Video fayl topilmadi",
			})
		}
		defer object.Close()

		c.Set("Content-Type", "video/mp4")
		c.Set("Accept-Ranges", "bytes")

		_, err = io.Copy(c.Response().BodyWriter(), object)
		return err
	}
}

func SearchVideos(videoService *services.VideoService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		keyword := c.Query("q")
		limit := c.QueryInt("limit", 20)

		if keyword == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Search keyword kerak",
			})
		}

		videos, err := videoService.SearchVideos(c.Context(), keyword, limit)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"results": videos,
			"total":   len(videos),
		})
	}
}

func GetTrending(analyticsService *services.AnalyticsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 10)

		videos, err := analyticsService.GetTrendingVideos(c.Context(), limit)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"trending": videos,
			"total":    len(videos),
		})
	}
}

func GetVideoAnalytics(analyticsService *services.AnalyticsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		videoID := c.Params("id")
		days := c.QueryInt("days", 7)

		analytics, err := analyticsService.GetVideoAnalytics(c.Context(), videoID, days)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"analytics": analytics,
		})
	}
}
