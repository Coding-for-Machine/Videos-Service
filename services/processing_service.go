// services/processing_service.go
package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
)

type ProcessingService struct {
	minio *minio.Client
}

func NewProcessingService(minio *minio.Client) *ProcessingService {
	return &ProcessingService{minio: minio}
}

// Video transcoding - turli sifatda
func (s *ProcessingService) TranscodeVideo(ctx context.Context, videoID gocql.UUID, fileName string) (map[string]string, error) {
	log.Printf("Video transcoding boshlandi: %s", videoID)

	// MinIOdan raw videoni yuklab olish
	objectName := fmt.Sprintf("raw/%s-%s", videoID.String(), fileName)
	object, err := s.minio.GetObject(ctx, "videos-raw", objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("MinIOdan yuklash xatosi: %w", err)
	}
	defer object.Close()

	// Vaqtinchalik faylga saqlash
	tempDir := os.TempDir()
	inputPath := filepath.Join(tempDir, fmt.Sprintf("%s-input.mp4", videoID))
	outputFile, err := os.Create(inputPath)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(outputFile, object)
	outputFile.Close()
	if err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)

	// Turli sifatlarda transcode qilish
	qualities := map[string]string{
		"360p":  "640x360",
		"480p":  "854x480",
		"720p":  "1280x720",
		"1080p": "1920x1080",
	}

	qualityVersions := make(map[string]string)

	for quality, resolution := range qualities {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("%s-%s.mp4", videoID, quality))

		// FFmpeg command
		cmd := exec.Command("ffmpeg",
			"-i", inputPath,
			"-vf", fmt.Sprintf("scale=%s", resolution),
			"-c:v", "libx264",
			"-crf", "23",
			"-preset", "medium",
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			outputPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg xatosi (%s): %s", quality, output)
			continue
		}

		// Processed videoni MinIOga yuklash
		minioPath := fmt.Sprintf("processed/%s/%s-%s.mp4", videoID, videoID, quality)
		file, _ := os.Open(outputPath)
		fileInfo, _ := file.Stat()

		_, err = s.minio.PutObject(ctx, "videos-processed", minioPath, file, fileInfo.Size(), minio.PutObjectOptions{
			ContentType: "video/mp4",
		})
		file.Close()
		os.Remove(outputPath)

		if err == nil {
			qualityVersions[quality] = fmt.Sprintf("/videos/%s/%s", videoID, quality)
		}
	}

	log.Printf("Video transcoding tugadi: %s", videoID)
	return qualityVersions, nil
}

// Thumbnail yaratish
func (s *ProcessingService) GenerateThumbnail(ctx context.Context, videoID gocql.UUID, fileName string) (string, error) {
	log.Printf("Thumbnail yaratish boshlandi: %s", videoID)

	// MinIOdan raw videoni yuklab olish
	objectName := fmt.Sprintf("raw/%s-%s", videoID.String(), fileName)
	object, err := s.minio.GetObject(ctx, "videos-raw", objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("MinIOdan yuklash xatosi: %w", err)
	}
	defer object.Close()

	// Vaqtinchalik faylga saqlash
	tempDir := os.TempDir()
	inputPath := filepath.Join(tempDir, fmt.Sprintf("%s-input.mp4", videoID))
	outputPath := filepath.Join(tempDir, fmt.Sprintf("%s-thumb.jpg", videoID))

	outputFile, err := os.Create(inputPath)
	if err != nil {
		return "", err
	}
	io.Copy(outputFile, object)
	outputFile.Close()
	defer os.Remove(inputPath)

	// FFmpeg bilan thumbnail yaratish (5-soniyada)
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ss", "00:00:05",
		"-vframes", "1",
		"-vf", "scale=1280:720",
		"-q:v", "2",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("thumbnail yaratish xatosi: %w", err)
	}
	defer os.Remove(outputPath)

	// Thumbnailni MinIOga yuklash
	file, err := os.Open(outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	minioPath := fmt.Sprintf("%s/thumbnail.jpg", videoID)

	_, err = s.minio.PutObject(ctx, "thumbnails", minioPath, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", err
	}

	thumbnailURL := fmt.Sprintf("/thumbnails/%s/thumbnail.jpg", videoID)
	log.Printf("Thumbnail yaratildi: %s", videoID)
	return thumbnailURL, nil
}

// Video davomiyligini olish
func (s *ProcessingService) GetVideoDuration(ctx context.Context, videoID gocql.UUID, fileName string) (int, error) {
	objectName := fmt.Sprintf("raw/%s-%s", videoID.String(), fileName)
	object, err := s.minio.GetObject(ctx, "videos-raw", objectName, minio.GetObjectOptions{})
	if err != nil {
		return 0, err
	}
	defer object.Close()

	tempDir := os.TempDir()
	inputPath := filepath.Join(tempDir, fmt.Sprintf("%s-input.mp4", videoID))

	outputFile, err := os.Create(inputPath)
	if err != nil {
		return 0, err
	}
	io.Copy(outputFile, object)
	outputFile.Close()
	defer os.Remove(inputPath)

	// FFprobe bilan davomiylikni olish
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var duration float64
	fmt.Sscanf(string(output), "%f", &duration)
	return int(duration), nil
}
