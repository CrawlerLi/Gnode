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
	if err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	log.Printf("Get %s response from node %s", resp.Message, resp.NodeId)
	return resp, nil
}

func (c *Client) GetChainState(ctx context.Context) (*pb.ChainStateResponse, error) {
	resp, err := c.Gclient.GetChainState(ctx, &pb.ChainStateRequest{NodeId: c.LocalNodeID})
	if err != nil {
		return nil, fmt.Errorf("get chain state: %w", err)
	}
	return resp, nil
}
