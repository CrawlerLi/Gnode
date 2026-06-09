package p2p

import (
	"context"
	"fmt"
	"log"

	pb "github.com/CrawlerLi/myMiniBitcoin/internal/p2p/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Conn        *grpc.ClientConn
	Gclient     pb.PeerServiceClient
	LocalNodeID string
}

func NewClient(peerAddr string, localNodeID string) (*Client, error) {
	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create grpc client to %s: %w", peerAddr, err)
	}

	return &Client{
		Conn:        conn,
		Gclient:     pb.NewPeerServiceClient(conn),
		LocalNodeID: localNodeID,
	}, nil
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

func (c *Client) Ping(ctx context.Context) (*pb.PingResponse, error) {
	resp, err := c.Gclient.Ping(ctx, &pb.PingRequest{NodeId: c.LocalNodeID})
	log.Printf("Get %s response from node %s", resp.Message, resp.NodeId)
	return resp, err
}
