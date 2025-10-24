// config/config.go
package config

import (
	"os"
)

type Config struct {
	Port           string
	CassandraHosts []string
	MinIO          MinIOConfig
	RedisAddr      string
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "3000"),
		CassandraHosts: []string{
			getEnv("CASSANDRA_HOST", "127.0.0.1:9042"),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:          false,
			BucketName:      getEnv("MINIO_BUCKET", "videos"),
		},
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
