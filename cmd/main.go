package main

import (
	"fmt"
	"log"
	"os"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

const testDBFile = "test_temp.db"

func main() {
	if err := run(); err != nil {
		log.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run() error {
	Alice, err := crypto.NewWallet()
	if err != nil {
		return fmt.Errorf("failed to create Alice wallet: %w", err)
	}
	Bob, err := crypto.NewWallet()
	if err != nil {
		return fmt.Errorf("failed to create Bob wallet: %w", err)
	}

	fmt.Println("The wallets have been constructed Successfully")
	fmt.Println("The address of Alice is ", string(Alice.Address))
	fmt.Println("The address of Bob is ", string(Bob.Address))
	fmt.Println()

	bc, err := core.NewBlockChain(crypto.HashPubkey(Alice.Publickey), testDBFile)

	if err != nil {
		return fmt.Errorf("failed to create block chain: %w", err)
	}

	defer func() {
		if err := bc.DB.Close(); err != nil {
			fmt.Printf("failed to close database: %s", err)
		}
		if err := os.Remove(testDBFile); err != nil {
			fmt.Printf("failed to remove database file: %s", err)
		}
	}()

	fmt.Println("The gensis block has been created!")
	err = bc.Print()
	if err != nil {
		return fmt.Errorf("failed to print blockchain: %w", err)
	}

	coinbaseTx, err := core.NewCoinBase(crypto.HashPubkey(Alice.Publickey))
	if err != nil {
		return fmt.Errorf("failed to create coinbase transaction: %w", err)
	}
	err = bc.AddBlock([]*core.Transaction{coinbaseTx})
	if err != nil {
		return fmt.Errorf("failed to add block: %w", err)
	}
	fmt.Println("The second block has been created")
	err = bc.Print()
	if err != nil {
		return fmt.Errorf("failed to print blockchain: %w", err)
	}

	fmt.Println("======初始余额======")
	banlanceA, err := wallet.GetBalance(bc, string(Alice.Address))
	if err != nil {
		return fmt.Errorf("failed to get Alice's balance: %w", err)
	}
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB, err := wallet.GetBalance(bc, string(Bob.Address))
	if err != nil {
		return fmt.Errorf("failed to get Bob's balance: %w", err)
	}
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	//打包一笔交易

	NewTx, err := wallet.NewTrasaction(Alice, string(Bob.Address), 30, bc)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	coinbaseTx2, err := core.NewCoinBase(crypto.HashPubkey(Alice.Publickey))
	if err != nil {
		return fmt.Errorf("failed to create coinbase transaction: %w", err)
	}
	err = bc.AddBlock([]*core.Transaction{NewTx, coinbaseTx2})
	if err != nil {
		return fmt.Errorf("failed to add block: %w", err)
	}
	fmt.Println("The third block has been created, all transactions have benn verified!")
	err = bc.Print()
	if err != nil {
		return fmt.Errorf("failed to print blockchain: %w", err)
	}

	fmt.Println("======最终余额======")
	banlanceA, err = wallet.GetBalance(bc, string(Alice.Address))
	if err != nil {
		return fmt.Errorf("failed to get Alice's balance: %w", err)
	}
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB, err = wallet.GetBalance(bc, string(Bob.Address))
	if err != nil {
		return fmt.Errorf("failed to get Bob's balance: %w", err)
	}
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	fmt.Println("You are good, man!")

	return nil
}
