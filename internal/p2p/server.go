package p2p

import (
	pb "github.com/CrawlerLi/myMiniBitcoin/internal/p2p/proto"
)

type server struct {
	pb.UnimplementedPeerServiceServer
}


