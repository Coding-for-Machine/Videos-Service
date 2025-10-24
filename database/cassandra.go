// database/cassandra.go
package database

import (
	"log"
	"time"

	"github.com/gocql/gocql"
)

// NewCassandraDB Cassandra bilan ulanishni yaratadi,
// agar keyspace mavjud bo‘lmasa — uni yaratadi va table’larni tayyorlaydi.
func NewCassandraDB(hosts []string) (*gocql.Session, error) {
	// 1️⃣ Avval default cluster (keyspace’siz) orqali ulanamiz
	cluster := gocql.NewCluster(hosts...)
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = 10 * time.Second
	cluster.Timeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// 2️⃣ Keyspace yaratish (agar mavjud bo‘lmasa)
	if err := createKeyspace(session); err != nil {
		return nil, err
	}
	log.Println("✅ Keyspace 'youtube_clone' mavjud yoki yaratildi")

	// 3️⃣ Endi yangi cluster — youtube_clone keyspace bilan
	cluster.Keyspace = "youtube_clone"
	keyspaceSession, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	// 4️⃣ Table’larni yaratish
	if err := createTables(keyspaceSession); err != nil {
		return nil, err
	}
	log.Println("✅ Cassandra table’lar yaratildi")

	log.Println("🚀 Cassandra ulanish muvaffaqiyatli o‘rnatildi")
	return keyspaceSession, nil
}

// createKeyspace — youtube_clone keyspace’ni yaratadi
func createKeyspace(session *gocql.Session) error {
	query := `
	CREATE KEYSPACE IF NOT EXISTS youtube_clone
	WITH replication = {
		'class': 'SimpleStrategy',
		'replication_factor': 1
	}`
	return session.Query(query).Exec()
}

// createTables — barcha zarur jadval (table)larni yaratadi
func createTables(session *gocql.Session) error {
	tables := []string{
		// Videos table
		`CREATE TABLE IF NOT EXISTS videos (
			id UUID PRIMARY KEY,
			title TEXT,
			description TEXT,
			user_id UUID,
			username TEXT,
			file_name TEXT,
			file_size BIGINT,
			duration INT,
			thumbnail_url TEXT,
			video_url TEXT,
			status TEXT,
			quality_versions MAP<TEXT, TEXT>,
			views COUNTER,
			likes COUNTER,
			dislikes COUNTER,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,

		// Videos by user
		`CREATE TABLE IF NOT EXISTS videos_by_user (
			user_id UUID,
			created_at TIMESTAMP,
			video_id UUID,
			title TEXT,
			thumbnail_url TEXT,
			views COUNTER,
			PRIMARY KEY (user_id, created_at, video_id)
		) WITH CLUSTERING ORDER BY (created_at DESC)`,

		// Trending videos
		`CREATE TABLE IF NOT EXISTS trending_videos (
			time_bucket TEXT,
			views COUNTER,
			video_id UUID,
			title TEXT,
			thumbnail_url TEXT,
			created_at TIMESTAMP,
			PRIMARY KEY (time_bucket, views, video_id)
		) WITH CLUSTERING ORDER BY (views DESC)`,

		// Video analytics
		`CREATE TABLE IF NOT EXISTS video_analytics (
			video_id UUID,
			date DATE,
			hour INT,
			views COUNTER,
			watch_time COUNTER,
			likes COUNTER,
			shares COUNTER,
			PRIMARY KEY ((video_id, date), hour)
		)`,

		// Search index (simplified)
		`CREATE TABLE IF NOT EXISTS video_search (
			keyword TEXT,
			video_id UUID,
			title TEXT,
			thumbnail_url TEXT,
			views COUNTER,
			created_at TIMESTAMP,
			PRIMARY KEY (keyword, views, video_id)
		) WITH CLUSTERING ORDER BY (views DESC)`,

		// Comments
		`CREATE TABLE IF NOT EXISTS comments (
			video_id UUID,
			created_at TIMESTAMP,
			comment_id UUID,
			user_id UUID,
			username TEXT,
			text TEXT,
			likes COUNTER,
			PRIMARY KEY (video_id, created_at, comment_id)
		) WITH CLUSTERING ORDER BY (created_at DESC)`,

		// Processing queue
		`CREATE TABLE IF NOT EXISTS processing_jobs (
			job_id UUID PRIMARY KEY,
			video_id UUID,
			job_type TEXT,
			status TEXT,
			priority INT,
			retry_count INT,
			error_message TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
	}

	for _, query := range tables {
		if err := session.Query(query).Exec(); err != nil {
			return err
		}
	}

	return nil
}
