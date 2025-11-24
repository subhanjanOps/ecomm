package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	orderspb "ecomm/orders-service/gen/orderspb"
)

type resp map[string]any

type ordersServer struct {
	orderspb.UnimplementedOrdersServiceServer
}

func (s *ordersServer) ListOrders(ctx context.Context, req *orderspb.ListOrdersRequest) (*orderspb.ListOrdersResponse, error) {
	return &orderspb.ListOrdersResponse{Orders: []*orderspb.Order{}}, nil
}

func main() {
	httpPort := getenv("PORT", "8083")
	grpcPort := getenv("GRPC_PORT", "9093")

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp{"status": "ok"})
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp{"status": "ready"})
		})
		srv := &http.Server{Addr: ":" + httpPort, Handler: mux}
		log.Printf("orders-service http listening on :%s", httpPort)
		log.Println(srv.ListenAndServe())
	}()

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	gs := grpc.NewServer()
	orderspb.RegisterOrdersServiceServer(gs, &ordersServer{})
	reflection.Register(gs)
	log.Printf("orders-service grpc listening on :%s", grpcPort)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
