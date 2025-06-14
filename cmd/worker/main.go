package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Redis + MongoDB 連線設定
const (
	redisAddr     = "localhost:6379"
	redisQueueKey = "form:queue"

	mongoURI  = "mongodb+srv://z416854:Chris710!@clusterkris.pzdoz64.mongodb.net/"
	mongoDB   = "form_collector"
	mongoColl = "submissions"
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

	// 啟動 Prometheus metrics
	initMetricsServer()

	// 連線 Redis
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis 連線失敗: %v", err)
	}
	log.Println("✅ Redis 連線成功")

	// 連線 MongoDB Atlas
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB 連線失敗: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	log.Println("✅ MongoDB Atlas 連線成功")

	collection := mongoClient.Database(mongoDB).Collection(mongoColl)

	// 無限迴圈：從 Redis 拿資料 ➜ 寫入 Mongo
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
			insertFailures.Inc()
			continue
		}

		_, err = collection.InsertOne(ctx, data)
		if err != nil {
			log.Printf("MongoDB 寫入失敗: %v", err)
			insertFailures.Inc()
			continue
		}

		insertSuccess.Inc()
		log.Println("✅ 寫入成功:", data)
	}
}
