package node

import (
	"context"
	"testing"
	"time"
)

func TestTwoNodesPing(t *testing.T) {
	node1, err := InitNode(nil, "node1", "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("init node1: %v", err)
	}
	defer node1.Stop()

	node1.Start()

	node2, err := InitNode(nil, "node2", "127.0.0.1:0", []string{node1.Addr})
	if err != nil {
		t.Fatalf("init node2: %v", err)
	}
	defer node2.Stop()

	node2.Start()

	peer := node2.Peers[node1.Addr]
	if peer == nil {
		t.Fatalf("node2 did not connect node1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := peer.Ping(ctx)
	if err != nil {
		t.Fatalf("ping node1: %v", err)
	}

	if resp.NodeId != "node1" {
		t.Fatalf("expected node1, got %s", resp.NodeId)
	}

	if resp.Message != "Pong" {
		t.Fatalf("expected Pong, got %s", resp.Message)
	}
}
