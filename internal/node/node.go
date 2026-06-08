package node

import (
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/p2p"
	"github.com/CrawlerLi/myMiniBitcoin/internal/service"
)

type Node struct {
	AppService *service.AppService
	Server     *p2p.Server
	ID         string
	Addr       string
	errCh      chan error

	Peers map[string]*p2p.Client
}

func InitNode(appService *service.AppService, ID string, addr string) (*Node, error) {
	server, err := p2p.NewServer(addr, ID)
	if err != nil {
		return nil, fmt.Errorf("init node: %w", err)
	}

	return &Node{
		AppService: appService,
		Server:     server,
		ID:         ID,
		Addr:       server.Addr,
		errCh:      make(chan error, 1),
		Peers:      make(map[string]*p2p.Client),
	}, nil
}

func (n *Node) Start() {
	go func() {
		if err := n.Server.Start(); err != nil {
			n.errCh <- err
		}
	}()

}

func (n *Node) Errch() <-chan error {
	return n.errCh
}

func (n *Node) Stop() {
	if n.Server != nil {
		n.Server.Stop()
	}

	for _, client := range n.Peers {
		if client != nil {
			client.Close()
		}
	}

}
