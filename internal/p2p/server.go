package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/CrawlerLi/myMiniBitcoin/internal/p2p/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedPeerServiceServer
}

func (s *server) Ping(_ context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Received ping from node %s", in.GetNodeId())

	return &pb.PingResponse{
		NodeId:  "Server",
		Message: "Pong",
	}, nil

}

func NewGRPCServer() *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterPeerServiceServer(s, &server{})
	return s
}

func StartServer(port string) error {
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", port, err)
	}
	grpcServer := NewGRPCServer()
	log.Printf("server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve on port %s: %w", port, err)
	}
	return nil
}
