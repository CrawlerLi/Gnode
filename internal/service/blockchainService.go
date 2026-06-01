package service

import (
	"errors"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

type BlockchainService struct {
	chain       *core.BlockChain
	walletStore *wallet.WalletStorage
}

type ChainInfo struct {
	Height int
	Blocks []*core.Block
}

func InitChain(minerAddress string, chainDBFile string, walletDBFile string) (*ChainInfo, error) {
	if minerAddress == "" {

		//here init wallet storage and wallet service, it may be reviesed later.
		walletDB, err := database.OpenDB(walletDBFile)
		if err != nil {
			return nil, fmt.Errorf("init chain: open wallet db: %w", err)
		}
		defer walletDB.Close()

		if err := walletDB.CreateBucket("Wallet"); err != nil {
			return nil, fmt.Errorf("init chain: create wallet bucket: %w", err)
		}

		walletStorage := wallet.NewWalletStorage(walletDB)
		walletService := NewWalletService(walletStorage, nil)

		workerWalletInfo, err := walletService.GetWorkerWallet()
		if err != nil {
			if errors.Is(err, ErrWorkerWalletNotFound) {
				if err := walletService.CreateWallet("worker", "miner"); err != nil {
					return nil, fmt.Errorf("init chain: create worker wallet: %w", err)
				}

				workerWalletInfo, err = walletService.GetWorkerWallet()
				if err != nil {
					return nil, fmt.Errorf("init chain: reload worker wallet: %w", err)
				}
			} else {
				return nil, fmt.Errorf("init chain: get worker wallet: %w", err)
			}
		}

		minerAddress = workerWalletInfo.Address
	}

	minerPubkeyHash, err := crypto.AddressToPubkeyHash([]byte(minerAddress))
	if err != nil {
		return nil, fmt.Errorf("init chain: parse miner address: %w", err)
	}

	bc, err := core.InitBlockChain(minerPubkeyHash, chainDBFile)
	if err != nil {
		return nil, fmt.Errorf("init chain: initialize blockchain: %w", err)
	}
	defer bc.DB.Close()

	return &ChainInfo{Height: bc.BestState.BlockHeight}, nil
}
