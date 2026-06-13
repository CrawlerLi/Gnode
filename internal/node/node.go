package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CrawlerLi/Gnode/internal/p2p"
	"github.com/CrawlerLi/Gnode/internal/service"
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
	PeerAddr     string
	RemoteNodeID string
	Message      string
}

type PeerChainState struct {
	PeerAddr     string
	RemoteNodeID string
	Height       int
	LastHash     []byte
}

type PeerBlocks struct {
	PeerAddr     string
	RemoteNodeID string
	BlocksData   []service.SerializedChainBlock
}

func InitNode(appService *service.AppService, localNodeID string, localNodeAddr string, peersAddr []string) (*Node, error) {
	server, err := p2p.NewServer(localNodeAddr, localNodeID)
	if err != nil {
		return nil, fmt.Errorf("init node: %w", err)
	}

	server.ChainStateProvider = func() (int, []byte, error) {
		state, err := appService.ChainService.GetChainState()
		if err != nil {
			return 0, nil, err
		}

		return state.Height, state.LastHash, nil
	}

	server.ChainBlockPeerProvider = func(startHeight int, limit int) ([]p2p.BlockPayload, error) {
		blocks, err := appService.ChainService.GetSerializedBlocksFromHeight(startHeight, limit)
		if err != nil {
			return nil, fmt.Errorf("chain block peer provider: %w", err)
		}

		blockPayloads := make([]p2p.BlockPayload, 0, len(blocks))
		for _, block := range blocks {
			blockPayloads = append(blockPayloads, p2p.BlockPayload{
				Height: block.Height,
				Block:  block.Block,
			})
		}

		return blockPayloads, nil

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
		PeerAddr:     peerAddr,
		RemoteNodeID: resp.NodeId,
		Message:      resp.Message,
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

func (n *Node) GetPeerBlocksFromHeight(localBestHeight int, limit int, peerAddr string) (*PeerBlocks, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	peer, ok := n.Peers[peerAddr]
	if !ok || peer == nil {
		return nil, fmt.Errorf("peer %s not connected", peerAddr)
	}

	resp, err := peer.GetBlocksFromHeight(ctx, localBestHeight, limit)
	if err != nil {
		return nil, fmt.Errorf("get peer blocks from height: %w", err)
	}

	blocksData := make([]service.SerializedChainBlock, 0, len(resp.Blocks))
	for _, blockData := range resp.Blocks {
		if blockData == nil {
			return nil, fmt.Errorf("get peer blocks from height: nil block data")
		}

		blocksData = append(blocksData, service.SerializedChainBlock{
			Height: int(blockData.Height),
			Block:  append([]byte(nil), blockData.Block...),
		})
	}

	return &PeerBlocks{
		PeerAddr:     peerAddr,
		RemoteNodeID: resp.NodeId,
		BlocksData:   blocksData,
	}, nil
}

func (n *Node) Start() error {
	go func() {
		log.Printf("node %s listening on %s\n", n.ID, n.Addr)
		if err := n.Server.Start(); err != nil {
			n.errCh <- err
		}
	}()

	for peerAddr := range n.Peers {
		log.Printf("connected peer: %s\n", peerAddr)
		//for ping test
		resp, err := n.PingPeer(peerAddr)
		if err != nil {
			log.Printf("failed to ping peer %s: %v", peerAddr, err)
			//Poll to check connection status
			continue
		}
		log.Printf("Received ping response [%s] from %s", resp.Message, peerAddr)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	peerChainStateTicker := time.NewTicker(2 * time.Second)
	defer peerChainStateTicker.Stop()

	for {
		select {
		case <-interrupt:
			log.Println("node shutting down")
			return nil
		case err := <-n.Errch():
			if err != nil {
				return fmt.Errorf("run node: node stopped unexpectedly: %w", err)
			}
		case <-peerChainStateTicker.C:
			peerBestHeight := 0
			peerBestStateAddress := ""
			for peerAddr := range n.Peers {
				peerState, err := n.GetPeerChainState(peerAddr)
				if err != nil {
					log.Printf("failed to get peer %s chainstate: %v", peerAddr, err)
					continue
				}
				log.Printf("peer chain state: peer=%s, node=%s height=%d, besthash=%x\n",
					peerState.PeerAddr,
					peerState.RemoteNodeID,
					peerState.Height,
					peerState.LastHash)
				if peerBestHeight < peerState.Height {
					peerBestHeight = peerState.Height
					peerBestStateAddress = peerAddr
				}
			}
			localBestState, err := n.AppService.ChainService.GetChainState()
			if err != nil {
				return fmt.Errorf("run node : %w", err)
			}

			if localBestState.Height < peerBestHeight {
				log.Printf("get best state from peer %s, peer block height is %d, local block height is %d\n",
					peerBestStateAddress,
					peerBestHeight,
					localBestState.Height)

				log.Printf("start getting blocks from peer %s\n", peerBestStateAddress)
				peerBlocks, err := n.GetPeerBlocksFromHeight(localBestState.Height, 20, peerBestStateAddress)
				if err != nil {
					return fmt.Errorf("run Node: get peer %s blocks from height at %d: %w",
						peerBestStateAddress,
						peerBestHeight,
						err)
				}
				if peerBlocks == nil {
					return fmt.Errorf("run Node: get peer %s blocks from height at %d: peerBlocks is nil",
						peerBestStateAddress,
						peerBestHeight)
				}

				peerLastBlockHeight := peerBlocks.BlocksData[len(peerBlocks.BlocksData)-1].Height
				log.Printf("start accepting blocks from peer %s\n, at Height %d to %d",
					peerBestStateAddress,
					localBestState.Height+1,
					peerLastBlockHeight)
				err = n.AppService.ChainService.SyncChainBlocks(peerBlocks.BlocksData)
				if err != nil {
					return fmt.Errorf("run Node: %w", err)
				}
			}

		}

	}
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
