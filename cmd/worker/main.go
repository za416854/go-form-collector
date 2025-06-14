package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Prometheus 指標
var (
	insertSuccess = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_insert_total",
		Help: "Number of successful MongoDB inserts",
	})
	insertFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_failures_total",
		Help: "Number of failed MongoDB inserts",
	})
)

func initMetricsServer() {
	prometheus.MustRegister(insertSuccess)
	prometheus.MustRegister(insertFailures)

	go func() {
		log.Println("✅ Worker metrics available at :9091/metrics")
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9091", nil)
	}()
}

func main() {
	ctx := context.Background()

	// 啟動 Prometheus
	initMetricsServer()

	// 讀取環境變數
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("❌ REDIS_URL 環境變數未設定")
	}

	mongoURI := "mongodb+srv://z416854:Chris710!@clusterkris.pzdoz64.mongodb.net/"
	if mongoURI == "" {
		log.Fatal("❌ MONGODB_URI 環境變數未設定")
	}

	queueKey := "form:queue"

	// 初始化 Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("❌ Redis URL 解析錯誤: %v", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis 連線失敗: %v", err)
	}
	log.Println("✅ Redis 連線成功")

	// 初始化 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ MongoDB 連線失敗: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	log.Println("✅ MongoDB 連線成功")

	db := mongoClient.Database("form_collector")
	coll := db.Collection("submissions")

	// 循環處理佇列
	for {
		result, err := rdb.BRPop(ctx, 0*time.Second, queueKey).Result()
		if err != nil {
			log.Printf("❌ 從 Redis 取資料失敗: %v", err)
			continue
		}
		if len(result) < 2 {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result[1]), &data); err != nil {
			log.Printf("❌ JSON 解析失敗: %v", err)
			insertFailures.Inc()
			continue
		}

		if _, err = coll.InsertOne(ctx, data); err != nil {
			log.Printf("❌ MongoDB 寫入失敗: %v", err)
			insertFailures.Inc()
			continue
		}

		insertSuccess.Inc()
		log.Println("✅ 寫入成功:", data)
	}
}
