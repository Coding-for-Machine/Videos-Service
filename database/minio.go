// database/minio.go
package database

import (
	"context"
	"log"

	"github.com/Coding-for-Machine/Videos-Service/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinIOClient(cfg config.MinIOConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Bucket yaratish
	ctx := context.Background()
	buckets := []string{
		cfg.BucketName,                // videos
		cfg.BucketName + "-raw",       // raw videos
		cfg.BucketName + "-processed", // processed videos
		"thumbnails",                  // thumbnails
	}

	for _, bucketName := range buckets {
		exists, err := client.BucketExists(ctx, bucketName)
		if err != nil {
			return nil, err
		}

		if !exists {
			err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
			if err != nil {
				return nil, err
			}
			log.Printf("MinIO bucket yaratildi: %s", bucketName)
		}
	}

	// Bucket policy o'rnatish (public read)
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + cfg.BucketName + `/*"]
			}
		]
	}`

	err = client.SetBucketPolicy(ctx, cfg.BucketName, policy)
	if err != nil {
		log.Printf("Policy o'rnatilmadi: %v", err)
	}

	log.Println("MinIO ulanish muvaffaqiyatli")
	return client, nil
}
