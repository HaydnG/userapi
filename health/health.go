package health

import (
	"context"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	// GRPCAddress is the address that the gRPC server is listening on
	GRPCAddress  = ":9090" // Update this if necessary or use an environment variable
	healthClient healthpb.HealthClient
)

func init() {
	var err error

	conn, err := grpc.NewClient(GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}

	healthClient = healthpb.NewHealthClient(conn)
}

// CheckHandler verifies the health of HTTP and gRPC
func CheckHandler(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil || resp.Status != healthpb.HealthCheckResponse_SERVING {
		http.Error(w, "gRPC health check failed", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
