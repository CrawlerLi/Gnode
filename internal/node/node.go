package node

import (
	"context"
	"fmt"
	"time"

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

type PingResponse struct {
	PeerAddr    string
	RemotNodeID string
	Messgae     string
}

type PeerChainState struct {
	PeerAddr     string
	RemoteNodeID string
	Height       int
	LastHash     []byte
}

func InitNode(appService *service.AppService, localNodeID string, localNodeAddr string, peersAddr []string) (*Node, error) {
	server, err := p2p.NewServer(localNodeAddr, localNodeID)
	if err != nil {
		return nil, fmt.Errorf("init node: %w", err)
	}

	server.ChainStateProvider = func() (int, []byte, error) {
		state, err := appService.ChainService.RequireChainState()
		if err != nil {
			return 0, nil, err
		}

		return state.Height, state.LastHash, nil
	}

	n := &Node{
		AppService: appService,
		Server:     server,
		ID:         localNodeID,
		Addr:       server.Addr,
		errCh:      make(chan error, 1),
		Peers:      make(map[string]*p2p.Client),
	}

	if len(peersAddr) > 0 {
		if err = n.ConnectPeers(peersAddr); err != nil {
			n.Stop()
			return nil, fmt.Errorf("init node: connect peers: %w", err)
		}

	}

	return n, nil

}

func (n *Node) ConnectPeers(peersAddr []string) error {
	for _, addr := range peersAddr {
		err := n.ConnectPeer(addr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) ConnectPeer(peerAddr string) error {
	client, err := p2p.NewClient(peerAddr, n.ID)
	if err != nil {
		return fmt.Errorf("connect peer %s : %w", peerAddr, err)
	}

	n.Peers[peerAddr] = client
	return nil
}

func (n *Node) PingPeer(peerAddr string) (*PingResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	peer, ok := n.Peers[peerAddr]
	if !ok || peer == nil {
		return nil, fmt.Errorf("peer %s not connected", peerAddr)
	}

	resp, err := peer.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("ping peer %s: %w", peerAddr, err)
	}
	return &PingResponse{
		PeerAddr:    peerAddr,
		RemotNodeID: resp.NodeId,
		Messgae:     resp.Message,
	}, nil
}

func (n *Node) GetPeerChainState(peerAddr string) (*PeerChainState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	peer, ok := n.Peers[peerAddr]
	if !ok || peer == nil {
		return nil, fmt.Errorf("peer %s not connected", peerAddr)
	}

	resp, err := peer.GetChainState(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peer chain state: %w", err)
	}
	return &PeerChainState{
		PeerAddr:     peerAddr,
		RemoteNodeID: resp.NodeId,
		Height:       int(resp.Height),
		LastHash:     resp.BestHash,
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
