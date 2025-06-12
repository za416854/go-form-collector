package main

import (
	"context"
	"encoding/json"
	// "fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	redisAddr     = "localhost:6379"
	redisQueueKey = "form:queue"
	mongoURI      = "mongodb+srv://z416854:Chris710!@clusterkris.pzdoz64.mongodb.net/" // ← 👈 改成你貼的 URI
	mongoDB       = "form_collector"
	mongoColl     = "submissions"
)

func main() {
	ctx := context.Background()

	// 連接 Redis
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis 連線失敗: %v", err)
	}
	log.Println("✅ Redis 連線成功")

	// 連接 MongoDB Atlas
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB 連線失敗: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	log.Println("✅ MongoDB Atlas 連線成功")

	collection := mongoClient.Database(mongoDB).Collection(mongoColl)

	// 開始從 Redis 拿資料
	for {
		result, err := rdb.BRPop(ctx, 0*time.Second, redisQueueKey).Result()
		if err != nil {
			log.Printf("從 Redis 取資料失敗: %v", err)
			continue
		}

		if len(result) < 2 {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result[1]), &data); err != nil {
			log.Printf("JSON 解析失敗: %v", err)
			continue
		}

		_, err = collection.InsertOne(ctx, data)
		if err != nil {
			log.Printf("MongoDB 寫入失敗: %v", err)
			continue
		}

		log.Println("✅ 成功寫入 MongoDB:", data)
	}
}
