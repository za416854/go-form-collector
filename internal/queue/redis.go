package queue

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	QueueKey    = "form:queue"
	ctx         = context.Background()
)

func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatalf("❌ REDIS_URL 環境變數未設定")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("❌ Redis 連線字串解析失敗: %v", err)
	}

	RedisClient = redis.NewClient(opt)

	_, err = RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Redis 連線失敗: %v", err)
	}

	log.Println("✅ Redis 連線成功")
}
