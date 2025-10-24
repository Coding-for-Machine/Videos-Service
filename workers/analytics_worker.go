// workers/analytics_worker.go
package workers

import (
	"context"
	"log"
	"time"

	"github.com/Coding-for-Machine/Videos-Service/services"
)

// Analytics aggregator - har soatda analytics ma'lumotlarini to'playdi
func AnalyticsAggregatorWorker(ctx context.Context, analyticsService *services.AnalyticsService) {
	log.Println("Analytics Aggregator Worker ishga tushdi")

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Analytics aggregation boshlandi")

			// Soatlik ma'lumotlarni to'plash
			if err := analyticsService.AggregateHourlyStats(ctx); err != nil {
				log.Printf("Analytics aggregation xatosi: %v", err)
			}

			// Trending videolarni yangilash
			if err := analyticsService.UpdateTrendingVideos(ctx); err != nil {
				log.Printf("Trending update xatosi: %v", err)
			}

			log.Println("Analytics aggregation tugadi")

		case <-ctx.Done():
			return
		}
	}
}
