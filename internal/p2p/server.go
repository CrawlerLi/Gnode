package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/CrawlerLi/myMiniBitcoin/internal/p2p/proto"
	"google.golang.org/grpc"
)

type Server struct {
	GrpcServer *grpc.Server
	listener   net.Listener
	NodeID     string
	Addr       string

	pb.UnimplementedPeerServiceServer
}

func (s *Server) Ping(_ context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Received ping from node %s", in.GetNodeId())

	return &pb.PingResponse{
		NodeId:  "Server",
		Message: "Pong",
	}, nil

}

func NewGRPCServer() *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterPeerServiceServer(s, &Server{})
	return s
}

func NewServer(addr string, nodeId string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on addr %s: %w", addr, err)
	}
	grpcServer := NewGRPCServer()
	return &Server{
		GrpcServer: grpcServer,
		listener:   listener,
		NodeID:     nodeId,
		Addr:       listener.Addr().String(),
	}, nil
}

func (s *Server) Start() error {
	log.Printf("server listening at %v", s.listener.Addr())

	err := s.GrpcServer.Serve(s.listener)
	if err != nil {
		return fmt.Errorf("start server: bind listerner : %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.GrpcServer.GracefulStop()
}
