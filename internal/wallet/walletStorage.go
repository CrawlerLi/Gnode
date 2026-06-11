package wallet

import (
	"fmt"

	"github.com/CrawlerLi/Gnode/internal/infra/database"
)

type WalletStorage struct {
	DB database.DB
}

func NewWalletStorage(db database.DB) *WalletStorage {
	return &WalletStorage{DB: db}
}

func (s *WalletStorage) Save(username string, w *Wallet) error {

	err := s.DB.Update(func(tx database.Tx) error {
		walletBucket := tx.Bucket("Wallet")
		bytesWallet, err := w.SerializeWallet()
		if err != nil {
			return fmt.Errorf("serialize wallet: %w", err)
		}
		err = walletBucket.Put([]byte(username), bytesWallet)
		if err != nil {
			return fmt.Errorf("store wallet into database: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Store wallet: %w", err)
	}
	return nil
}

func (s *WalletStorage) Get(username string) (*Wallet, error) {

	var w *Wallet
	err := s.DB.View(func(tx database.Tx) error {
		walletBucket := tx.Bucket("Wallet")
		bytesWallet := walletBucket.Get([]byte(username))
		if bytesWallet == nil {
			return fmt.Errorf("failed to find wallet")
		}

		decode, err := DeserializedWallet(bytesWallet)

		if err != nil {
			return fmt.Errorf("Deserialize wallet: %w", err)
		}

		w = decode
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Get wallet: %w", err)
	}
	return w, nil
}

func (s *WalletStorage) List() (map[string]*Wallet, error) {
	wallets := make(map[string]*Wallet)

	err := s.DB.View(func(tx database.Tx) error {
		walletBucket := tx.Bucket("Wallet")
		cursor := walletBucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			decode, err := DeserializedWallet(v)
			if err != nil {
				return fmt.Errorf("Deserialize wallet: %w", err)
			}
			wallets[string(k)] = decode
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("List wallet: %w", err)
	}
	return wallets, nil
}

func (s *WalletStorage) Delete(username string) error {

	err := s.DB.Update(func(tx database.Tx) error {
		walletBucket := tx.Bucket("Wallet")
		err := walletBucket.Delete([]byte(username))
		if err != nil {
			return fmt.Errorf("delete wallet from database: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Delete wallet: %w", err)
	}
	return nil
}
