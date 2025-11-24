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

	userpb "ecomm/user-service/gen/userpb"
)

type resp map[string]any

type userServer struct {
	userpb.UnimplementedUserServiceServer
}

func (s *userServer) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	return &userpb.ListUsersResponse{Users: []*userpb.User{}}, nil
}

func main() {
	httpPort := getenv("PORT", "8081")
	grpcPort := getenv("GRPC_PORT", "9091")

	// HTTP health server
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
		log.Printf("user-service http listening on :%s", httpPort)
		log.Println(srv.ListenAndServe())
	}()

	// gRPC server
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	gs := grpc.NewServer()
	userpb.RegisterUserServiceServer(gs, &userServer{})
	reflection.Register(gs)
	log.Printf("user-service grpc listening on :%s", grpcPort)
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
