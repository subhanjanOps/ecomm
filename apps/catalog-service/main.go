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

	catalogpb "ecomm/catalog-service/gen/catalogpb"
)

type resp map[string]any

type catalogServer struct {
	catalogpb.UnimplementedCatalogServiceServer
}

func (s *catalogServer) ListProducts(ctx context.Context, req *catalogpb.ListProductsRequest) (*catalogpb.ListProductsResponse, error) {
	return &catalogpb.ListProductsResponse{Products: []*catalogpb.Product{}}, nil
}

func main() {
	httpPort := getenv("PORT", "8082")
	grpcPort := getenv("GRPC_PORT", "9092")

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
		log.Printf("catalog-service http listening on :%s", httpPort)
		log.Println(srv.ListenAndServe())
	}()

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	gs := grpc.NewServer()
	catalogpb.RegisterCatalogServiceServer(gs, &catalogServer{})
	reflection.Register(gs)
	log.Printf("catalog-service grpc listening on :%s", grpcPort)
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
