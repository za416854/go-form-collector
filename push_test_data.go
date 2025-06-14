package testtool

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // 根據你的設定
	})

	err := rdb.LPush(ctx, "form:queue", `{"name": "test user", "email": "test@example.com"}`).Err()
	if err != nil {
		log.Fatalf("❌ 推資料進 Redis 失敗: %v", err)
	}

	log.Println("✅ 成功推一筆資料進 Redis queue")
}
