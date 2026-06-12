package p2p

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := NewGRPCServer()
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()

	client, err := NewClient(lis.Addr().String(), "TestNode")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	response, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("client failed to ping server: %v", err)
	}
	t.Logf("Get response: %s", response.Message)
}
