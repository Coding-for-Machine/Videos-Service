// workers/video_processing_worker.go
package workers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Coding-for-Machine/Videos-Service/models"
	"github.com/Coding-for-Machine/Videos-Service/services"
	"github.com/redis/go-redis/v9"
)

// Video processing worker - videolarni transcoding qiladi
func VideoProcessingWorker(ctx context.Context, redis *redis.Client, processingService *services.ProcessingService, videoService *services.VideoService) {
	log.Println("Video Processing Worker ishga tushdi")

	for {
		// Redis queuedan jobni olish
		result, err := redis.BLPop(ctx, 0, "processing_queue").Result()
		if err != nil {
			log.Printf("Queue xatosi: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var job models.ProcessingJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("Job parse xatosi: %v", err)
			continue
		}

		// Jobni processing statusga o'zgartirish
		job.Status = "processing"
		job.UpdatedAt = time.Now()

		log.Printf("Job boshlandi: %s (type: %s, video: %s)", job.JobID, job.JobType, job.VideoID)

		// Job turini tekshirish
		switch job.JobType {
		case "transcode":
			go processTranscodeJob(ctx, job, processingService, videoService)
		case "thumbnail":
			go processThumbnailJob(ctx, job, processingService, videoService)
		default:
			log.Printf("Noma'lum job turi: %s", job.JobType)
		}
	}
}

func processTranscodeJob(ctx context.Context, job models.ProcessingJob, processingService *services.ProcessingService, videoService *services.VideoService) {
	// Videoni olish
	video, err := videoService.GetVideo(ctx, job.VideoID.String())
	if err != nil {
		log.Printf("Video topilmadi: %v", err)
		return
	}

	// Transcoding
	qualityVersions, err := processingService.TranscodeVideo(ctx, job.VideoID, video.FileName)
	if err != nil {
		log.Printf("Transcoding xatosi: %v", err)
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		return
	}

	// Video URLni yangilash
	videoURL := qualityVersions["720p"] // Default quality
	if videoURL == "" {
		videoURL = qualityVersions["480p"]
	}

	// Statusni yangilash
	err = videoService.UpdateVideoStatus(ctx, job.VideoID, "ready", videoURL, video.ThumbnailURL)
	if err != nil {
		log.Printf("Status yangilash xatosi: %v", err)
		return
	}

	job.Status = "completed"
	job.UpdatedAt = time.Now()
	log.Printf("Transcoding tugadi: %s", job.VideoID)
}

func processThumbnailJob(ctx context.Context, job models.ProcessingJob, processingService *services.ProcessingService, videoService *services.VideoService) {
	video, err := videoService.GetVideo(ctx, job.VideoID.String())
	if err != nil {
		log.Printf("Video topilmadi: %v", err)
		return
	}

	// Thumbnail yaratish
	thumbnailURL, err := processingService.GenerateThumbnail(ctx, job.VideoID, video.FileName)
	if err != nil {
		log.Printf("Thumbnail xatosi: %v", err)
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		return
	}

	// Video davomiyligini olish
	duration, _ := processingService.GetVideoDuration(ctx, job.VideoID, video.FileName)

	// Ma'lumotlarni yangilash
	err = videoService.UpdateVideoStatus(ctx, job.VideoID, video.Status, video.VideoURL, thumbnailURL)
	if err != nil {
		log.Printf("Status yangilash xatosi: %v", err)
		return
	}

	job.Status = "completed"
	job.UpdatedAt = time.Now()
	log.Printf("Thumbnail yaratildi: %s (duration: %d)", job.VideoID, duration)
}

// Thumbnail generator worker
func ThumbnailGeneratorWorker(ctx context.Context, redis *redis.Client, processingService *services.ProcessingService, videoService *services.VideoService) {
	log.Println("Thumbnail Generator Worker ishga tushdi")
	// VideoProcessingWorker bilan bir xil mantiq
}

// View counter worker - viewlarni batch rejimida yangilaydi
func ViewCounterWorker(ctx context.Context, redis *redis.Client, videoService *services.VideoService) {
	log.Println("View Counter Worker ishga tushdi")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// View queuedan barcha viewlarni olish
			views := make(map[string]int)

			for {
				result, err := redis.LPop(ctx, "view_queue").Result()
				if err != nil {
					break // Queue bo'sh
				}
				views[result]++
			}

			// Batch rejimida yangilash
			for videoID, count := range views {
				for i := 0; i < count; i++ {
					videoService.IncrementView(ctx, videoID)
				}
				log.Printf("Views yangilandi: %s (+%d)", videoID, count)
			}
		case <-ctx.Done():
			return
		}
	}
}
