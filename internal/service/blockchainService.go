package service

import (
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
)

type BlockchainService struct {
	chain       *core.BlockChain
	walletStore *wallet.WalletStorage
}

type ChainInfo struct {
	Height   int
	LastHash []byte
	Blocks   []*core.Block
}

type ChainState struct {
	Height   int
	LastHash []byte
}

func (bcs *BlockchainService) RequireChainInfo() (*ChainInfo, error) {
	var blocks []*core.Block

	err := bcs.chain.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("find blocks bucket")
		}

		LastBlockHash := bcs.chain.BestState.Hash
		if LastBlockHash == nil {
			return fmt.Errorf("blocks print: check LastBlockHash")
		}

		hashPointer := LastBlockHash
		for {
			blockBytes := blockBucket.Get(hashPointer)
			if blockBytes == nil {
				return fmt.Errorf("blocks print: get block by hash: %x", hashPointer)
			}
			block, err := core.DeserializedBlock(blockBytes)
			if err != nil {
				return fmt.Errorf("blocks print: deserialize block: %w", err)
			}

			blocks = append(blocks, block)
			if len(block.PrevHash) == 0 {
				break
			}

			hashPointer = block.PrevHash

		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("check blockchain info:view in db: %w", err)
	}

	return &ChainInfo{
		Height:   bcs.chain.BestState.BlockHeight,
		LastHash: bcs.chain.BestState.Hash,
		Blocks:   blocks,
	}, nil
}

func (bcs *BlockchainService) RequireChainState() (*ChainState, error) {
	state := bcs.chain.BestSnapshot()
	if state == nil {
		return nil, fmt.Errorf("require chain state: best state is nil")
	}

	return &ChainState{
		Height:   state.BlockHeight,
		LastHash: state.Hash,
	}, nil
}
