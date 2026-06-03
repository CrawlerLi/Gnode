package service

import (
	"errors"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

type AppService struct {
	ChainService  *BlockchainService
	WalletService *WalletService

	chainDB  database.DB
	walletDB database.DB
}

func InitApp(minerAddress string, chainDBFile string, walletDBFile string) (*AppService, error) {
	//here init wallet storage and wallet service, it may be reviesed later.
	walletDB, err := database.OpenDB(walletDBFile)
	if err != nil {
		return nil, fmt.Errorf("init chain: open wallet db: %w", err)
	}

	if err := walletDB.CreateBucket("Wallet"); err != nil {
		walletDB.Close()
		return nil, fmt.Errorf("init chain: create wallet bucket: %w", err)
	}

	walletStorage := wallet.NewWalletStorage(walletDB)
	walletService := NewWalletService(walletStorage, nil)

	if minerAddress == "" {
		workerWalletInfo, err := walletService.GetWorkerWallet()
		if err != nil {
			if errors.Is(err, ErrWorkerWalletNotFound) {
				if err := walletService.CreateWallet("worker", "miner"); err != nil {
					walletDB.Close()
					return nil, fmt.Errorf("init chain: create worker wallet: %w", err)
				}

				workerWalletInfo, err = walletService.GetWorkerWallet()
				if err != nil {
					walletDB.Close()
					return nil, fmt.Errorf("init chain: reload worker wallet: %w", err)
				}
			} else {
				walletDB.Close()
				return nil, fmt.Errorf("init chain: get worker wallet: %w", err)
			}
		}

		minerAddress = workerWalletInfo.Address
	}

	minerPubkeyHash, err := crypto.AddressToPubkeyHash([]byte(minerAddress))
	if err != nil {
		walletDB.Close()
		return nil, fmt.Errorf("init chain: parse miner address: %w", err)
	}

	bc, err := core.InitBlockChain(minerPubkeyHash, chainDBFile)
	if err != nil {
		walletDB.Close()
		return nil, fmt.Errorf("init chain: initialize blockchain: %w", err)
	}
	walletService.bc = bc

	chainService := &BlockchainService{
		chain:       bc,
		walletStore: walletStorage,
	}

	appService := &AppService{
		ChainService:  chainService,
		WalletService: walletService,
		chainDB:       chainService.chain.DB,
		walletDB:      walletService.store.DB,
	}

	return appService, nil
}

func OpenServices(chainDBFile string, walletDBFile string) (*AppService, error) {
	bc, err := core.OpenBlockChain(chainDBFile)
	if err != nil {
		return nil, fmt.Errorf("open services: open blockchain: %w", err)
	}

	walletDB, err := database.OpenDB(walletDBFile)
	if err != nil {
		bc.DB.Close()
		return nil, fmt.Errorf("open services: open wallet db: %w", err)
	}

	if err := walletDB.CreateBucket("Wallet"); err != nil {
		bc.DB.Close()
		walletDB.Close()
		return nil, fmt.Errorf("open services: create wallet bucket: %w", err)
	}

	walletStorage := wallet.NewWalletStorage(walletDB)
	walletService := NewWalletService(walletStorage, bc)
	chainService := &BlockchainService{
		chain:       bc,
		walletStore: walletStorage,
	}

	return &AppService{
		ChainService:  chainService,
		WalletService: walletService,
		chainDB:       bc.DB,
		walletDB:      walletDB,
	}, nil
}

func (app *AppService) Close() error {
	var err error

	if app.walletDB != nil {
		if closeErr := app.walletDB.Close(); closeErr != nil {
			err = fmt.Errorf("close wallet db: %w", closeErr)
		}
	}

	if app.chainDB != nil {
		if closeErr := app.chainDB.Close(); closeErr != nil {
			if err != nil {
				return fmt.Errorf("%v; close chain db: %w", err, closeErr)
			}
			err = fmt.Errorf("close chain db: %w", closeErr)
		}
	}

	return err
}
