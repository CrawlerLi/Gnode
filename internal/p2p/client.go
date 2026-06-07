package p2p

import (
	"context"
	"fmt"

	pb "github.com/CrawlerLi/myMiniBitcoin/internal/p2p/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Conn    *grpc.ClientConn
	Gclient pb.PeerServiceClient
	NodeID  string
}

func NewClient(serverAddr string, nodeID string) (*Client, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		Conn:    conn,
		Gclient: pb.NewPeerServiceClient(conn),
		NodeID:  nodeID,
	}, nil
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

func (c *Client) Ping(ctx context.Context, nodeID string) (*pb.PingResponse, error) {
	return c.Gclient.Ping(ctx, &pb.PingRequest{NodeId: nodeID})
}
