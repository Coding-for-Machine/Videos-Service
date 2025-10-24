// main.go
package main

import (
	"context"
	"log"

	"github.com/Coding-for-Machine/Videos-Service/config"
	"github.com/Coding-for-Machine/Videos-Service/database"
	"github.com/Coding-for-Machine/Videos-Service/handlers"
	"github.com/Coding-for-Machine/Videos-Service/middleware"
	"github.com/Coding-for-Machine/Videos-Service/services"
	"github.com/Coding-for-Machine/Videos-Service/workers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Konfiguratsiya yuklash
	cfg := config.Load()

	// Cassandra ulanish
	cassandraSession, err := database.NewCassandraDB(cfg.CassandraHosts)
	if err != nil {
		log.Fatal("Cassandra ulanmadi:", err)
	}
	defer cassandraSession.Close()

	// MinIO ulanish
	minioClient, err := database.NewMinIOClient(cfg.MinIO)
	if err != nil {
		log.Fatal("MinIO ulanmadi:", err)
	}

	// Redis ulanish (queue uchun)
	redisClient := database.NewRedisClient(cfg.RedisAddr)
	defer redisClient.Close()

	// Services
	videoService := services.NewVideoService(cassandraSession, minioClient, redisClient)
	processingService := services.NewProcessingService(minioClient)
	analyticsService := services.NewAnalyticsService(cassandraSession)

	// Background workers ishga tushirish
	ctx := context.Background()

	// Video processing worker
	go workers.VideoProcessingWorker(ctx, redisClient, processingService, videoService)

	// Thumbnail generator worker
	go workers.ThumbnailGeneratorWorker(ctx, redisClient, processingService, videoService)

	// Analytics aggregator worker
	go workers.AnalyticsAggregatorWorker(ctx, analyticsService)

	// View counter worker
	go workers.ViewCounterWorker(ctx, redisClient, videoService)

	// Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 2 * 1024 * 1024 * 1024, // 2GB
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Routes
	api := app.Group("/api")

	// Video routes
	videos := api.Group("/videos")
	videos.Post("/", middleware.RateLimit(), handlers.UploadVideo(videoService))
	videos.Get("/", handlers.GetVideos(videoService))
	videos.Get("/:id", handlers.GetVideo(videoService))
	videos.Delete("/:id", handlers.DeleteVideo(videoService))
	videos.Post("/:id/view", handlers.IncrementView(videoService))
	videos.Get("/:id/stream", handlers.StreamVideo(videoService, minioClient))

	// Analytics routes
	analytics := api.Group("/analytics")
	analytics.Get("/trending", handlers.GetTrending(analyticsService))
	analytics.Get("/video/:id", handlers.GetVideoAnalytics(analyticsService))

	// Search routes
	search := api.Group("/search")
	search.Get("/", handlers.SearchVideos(videoService))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Static files
	app.Static("/", "./public")

	log.Printf("Server ishga tushdi: http://localhost:%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
