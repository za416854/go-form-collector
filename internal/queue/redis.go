package queue

import (
	"context"
	"encoding/json"
	"log"

	// "time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	QueueKey    = "form:queue"
	ctx         = context.Background()
)

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Codespace Redis 預設 port
		Password: "",               // 沒密碼
		DB:       0,
	})
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis 連線失敗: %v", err)
	}
	log.Println("✅ Redis 連線成功")
}

func EnqueueForm(data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return RedisClient.LPush(ctx, QueueKey, bytes).Err()
}
