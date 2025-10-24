// database/redis.go
package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis ulanmadi:", err)
	}

	log.Println("Redis ulanish muvaffaqiyatli")
	return client
}
