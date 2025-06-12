package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
	"github.com/za416854/go-form-collector/internal/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pb "github.com/za416854/go-form-collector/internal/proto/formpb"
)

// 當一個「gRPC伺服器」
// 意思是「我要實作一個 gRPC server，方法會對應到 proto 裡的 FormCollector」。
type formCollectorServer struct {
	pb.UnimplementedFormCollectorServer
}
// 當一個「REST伺服器」
var (
	apiQPS = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "api_requests_total",
		Help: "Total number of API requests",
	})
)
// 如果別人用 gRPC 把資料傳進來，我就回一句 “我收到了”
func (s *formCollectorServer) Submit(ctx context.Context, data *pb.FormData) (*pb.SubmitReply, error) {
	apiQPS.Inc() // 增加 API QPS 計數器
	err := queue.EnqueueForm(data)
	if err != nil {
		log.Printf("gRPC 佇列失敗: %v\n", err)
		return nil, err
	}
	log.Printf("gRPC received: %+v\n", data)
	return &pb.SubmitReply{Message: "Queued!"}, nil
}

// 如果有網站打 POST /submit，我就回一句 “我收到了”
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	apiQPS.Inc() // 增加 API QPS 計數器
	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)
	err := queue.EnqueueForm(data)
	if err != nil {
		log.Println("佇列失敗:", err)
		http.Error(w, "queue failed", 500)
		return
	}
	log.Println("REST received:", data)
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("queued"))
}

func main() {
	prometheus.MustRegister(apiQPS) // 註冊 API QPS 的計數器
	queue.InitRedis() // 初始化 Redis 連線
	// 「啟動 gRPC 伺服器」的那一段 
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterFormCollectorServer(grpcServer, &formCollectorServer{})
		reflection.Register(grpcServer) // 這行是可選的，讓 gRPC 伺服器可以被工具（如 grpcurl）探索
		log.Println("gRPC server started at :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 「啟動 REST 伺服器」的那一段
	r := chi.NewRouter()
	r.Post("/submit", handleSubmit)
	r.Handle("/metrics", promhttp.Handler())
	log.Println("REST server started at :8080")
	http.ListenAndServe(":8080", r)
}
