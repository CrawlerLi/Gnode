package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/CrawlerLi/Gnode/internal/p2p/proto"
	"google.golang.org/grpc"
)

type Server struct {
	GrpcServer *grpc.Server
	listener   net.Listener
	NodeID     string
	Addr       string

	ChainStateProvider func() (int, []byte, error)
	ChainBlockProvider func(startHeight int, limits int) ([]BlockPayload, error)

	pb.UnimplementedPeerServiceServer
}

type BlockPayload struct {
	Height int
	Block  []byte
}

func (s *Server) Ping(_ context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Received ping from node %s", in.GetNodeId())

	return &pb.PingResponse{
		NodeId:  s.NodeID,
		Message: "Pong",
	}, nil

}

func (s *Server) GetChainState(_ context.Context, in *pb.ChainStateRequest) (*pb.ChainStateResponse, error) {
	log.Printf("Received get chianstate request from node %s", in.GetNodeId())

	height, bestHash, err := s.ChainStateProvider()
	if err != nil {
		return nil, fmt.Errorf("Get chain state: call chain state provider: %w", err)
	}

	return &pb.ChainStateResponse{
		NodeId:   s.NodeID,
		Height:   int32(height),
		BestHash: bestHash,
	}, nil
}

func (s *Server) GetBlocksFromHeight(_ context.Context, in *pb.GetBlocksFromHeightRequest) (resp *pb.GetBlocksFromHeightResponse, err error) {
	log.Printf("Received get blocks request from node %s, peer block height is %d", in.GetNodeId(), in.GetStartHeight())
	BlockPayloads, err := s.ChainBlockProvider(int(in.GetStartHeight()), int(in.GetLimit()))
	if err != nil {
		return nil, fmt.Errorf("Get chain state: call blocks from height provider: %w", err)
	}

	for _, block := range BlockPayloads {
		resp.Blocks = append(resp.Blocks, &pb.BlockData{
			Height: int32(block.Height),
			Block:  block.Block,
		})
	}

	return resp, nil

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

	s := &Server{
		GrpcServer: grpc.NewServer(),
		listener:   listener,
		NodeID:     nodeId,
		Addr:       listener.Addr().String(),
	}

	pb.RegisterPeerServiceServer(s.GrpcServer, s)
	return s, nil
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
