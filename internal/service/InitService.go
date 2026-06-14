package service

import (
	"errors"
	"fmt"

	"github.com/CrawlerLi/Gnode/internal/core"
	"github.com/CrawlerLi/Gnode/internal/infra/database"
	"github.com/CrawlerLi/Gnode/internal/wallet"
	"github.com/CrawlerLi/Gnode/pkg/crypto"
)

type AppService struct {
	ChainService  *BlockchainService
	WalletService *WalletService

	chainDB  database.DB
	walletDB database.DB
}

func InitLocalApp(minerAddress string, chainDBFile string, walletDBFile string) (*AppService, error) {
	walletService, minerAddress, err := InitWalletSerivce(minerAddress, walletDBFile)
	if err != nil {
		return nil, fmt.Errorf("init local app: %w", err)
	}

	minerPubkeyHash, err := crypto.AddressToPubkeyHash([]byte(minerAddress))
	if err != nil {
		return nil, fmt.Errorf("init chain: parse miner address: %w", err)
	}

	bc, err := core.InitBlockChain(chainDBFile)

	coinbase, err := core.NewCoinBase(minerPubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("init chain: create genesis coinbase tx: %w", err)
	}

	genesisBlock := core.NewGenesisBlock(coinbase)
	err = bc.CommitGenesisBlock(genesisBlock)
	if err != nil {
		return nil, fmt.Errorf("init chain: %w", err)
	}

	walletService.bc = bc

	chainService := &BlockchainService{
		chain:       bc,
		walletStore: walletService.store,
	}

	appService := &AppService{
		ChainService:  chainService,
		WalletService: walletService,
		chainDB:       chainService.chain.DB,
		walletDB:      walletService.store.DB,
	}

	return appService, nil
}

func SyncInit(minerAddress string, chainDBFile string, walletDBFile string) (*AppService, error) {
	walletService, _, err := InitWalletSerivce(minerAddress, walletDBFile)
	if err != nil {
		return nil, fmt.Errorf("init local app: %w", err)
	}

	bc, err := core.InitBlockChain(chainDBFile)

	walletService.bc = bc

	chainService := &BlockchainService{
		chain:       bc,
		walletStore: walletService.store,
	}

	appService := &AppService{
		ChainService:  chainService,
		WalletService: walletService,
		chainDB:       chainService.chain.DB,
		walletDB:      walletService.store.DB,
	}

	return appService, nil
}

func IsChainInitialized(chainDBFile string) (bool, error) {
	bc, err := core.OpenBlockChain(chainDBFile)
	if err != nil {
		if errors.Is(err, core.ErrChainNotInitialized) {
			return false, nil
		}
		return false, fmt.Errorf("check chain initialized: %w", err)
	}
	defer bc.DB.Close()

	return true, nil
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

func OpenWalletService(walletDBFile string) (*WalletService, func() error, error) {
	db, err := database.OpenDB(walletDBFile)
	if err != nil {
		return nil, nil, fmt.Errorf("open wallet db: %w", err)
	}

	if err := db.CreateBucket("Wallet"); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("create wallet bucket: %w", err)
	}

	ws := NewWalletService(wallet.NewWalletStorage(db), nil)
	return ws, db.Close, nil
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
