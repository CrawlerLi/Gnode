package service

import (
	"fmt"

	"github.com/CrawlerLi/Gnode/internal/core"
	"github.com/CrawlerLi/Gnode/internal/infra/database"
	"github.com/CrawlerLi/Gnode/internal/wallet"
	"github.com/CrawlerLi/Gnode/pkg/utils"
)

type BlockchainService struct {
	chain       *core.BlockChain
	walletStore *wallet.WalletStorage
}

type ChainInfo struct {
	Height   int
	LastHash []byte
	Blocks   []BlockInfo
}

type BlockInfo struct {
	Height   int
	Hash     []byte
	PrevHash []byte
}

type ChainState struct {
	Height   int
	LastHash []byte
}

type SerializedChainBlock struct {
	Height int
	Block  []byte
}

func (bcs *BlockchainService) GetChainInfo() (*ChainInfo, error) {
	state := bcs.chain.BestSnapshot()
	if state == nil {
		return nil, fmt.Errorf("get chain info: best state is nil")
	}

	BlocksInfo := make([]BlockInfo, 0, state.BlockHeight)

	err := bcs.chain.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		heightIdxBucket := tx.Bucket("HeightIdx")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}
		if heightIdxBucket == nil {
			return fmt.Errorf("failed to find HeightIdx bucket")
		}

		for height := 1; height <= state.BlockHeight; height++ {
			heightBytes, err := utils.IntToBytes(height)
			if err != nil {
				return fmt.Errorf("convert height %d to bytes: %w", height, err)
			}

			hash := heightIdxBucket.Get(heightBytes)
			if hash == nil {
				return fmt.Errorf("get block hash by height %d", height)
			}

			blockBytes := blockBucket.Get(hash)
			if blockBytes == nil {
				return fmt.Errorf("get block by hash %x at height %d", hash, height)
			}

			block, err := core.DeserializedBlock(blockBytes)
			if err != nil {
				return fmt.Errorf("deserialize block at height %d: %w", height, err)
			}

			BlocksInfo = append(BlocksInfo, BlockInfo{
				Height:   height,
				Hash:     block.Hash,
				PrevHash: block.PrevHash,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get blockchain info: view in db: %w", err)
	}

	return &ChainInfo{
		Height:   state.BlockHeight,
		LastHash: state.Hash,
		Blocks:   BlocksInfo,
	}, nil
}

func (bcs *BlockchainService) GetChainState() (*ChainState, error) {
	state := bcs.chain.BestSnapshot()
	if state == nil {
		return nil, fmt.Errorf("get chain state: best state is nil")
	}

	return &ChainState{
		Height:   state.BlockHeight,
		LastHash: state.Hash,
	}, nil
}

func (bcs *BlockchainService) GetSerializedBlocksFromHeight(startHeight int, limit int) ([]SerializedChainBlock, error) {
	chainBlocks, err := bcs.chain.GetBlocksFromHeight(startHeight, limit)
	if err != nil {
		return nil, fmt.Errorf("get serialized blocks from height: %w", err)
	}

	blocks := make([]SerializedChainBlock, 0, len(chainBlocks))
	for _, chainBlock := range chainBlocks {
		if chainBlock.Block == nil {
			return nil, fmt.Errorf("get serialized blocks from height: block at height %d is nil", chainBlock.Height)
		}

		blockBytes, err := chainBlock.Block.SerializeBlock()
		if err != nil {
			return nil, fmt.Errorf("get serialized blocks from height: serialize block at height %d: %w", chainBlock.Height, err)
		}

		blocks = append(blocks, SerializedChainBlock{
			Height: chainBlock.Height,
			Block:  blockBytes,
		})
	}

	return blocks, nil
}

func (bcs *BlockchainService) SyncChainBlocks(chainblocks []core.ChainBlock) error {
	for _, chainblock := range chainblocks {
		err := bcs.chain.AcceptChainBlock(chainblock)
		if err != nil {
			return fmt.Errorf("sync chainblock at height %d: %w", chainblock.Height, err)
		}
	}
	return nil
}
