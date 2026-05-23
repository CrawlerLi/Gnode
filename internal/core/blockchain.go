package core

import (
	"bytes"
	"fmt"
	"log"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

type BlockChain struct {
	db database.DB
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) {
	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("signature verify failure")
		}
	}

	prevHash, err := bc.db.Get("Blocks", []byte("tips"))
	if err != nil {
		log.Panic("failed to get previous block hash")
	}
	newBlock := NewBlock(transactions, prevHash)
	bc.db.Put("Blocks", []byte("tips"), newBlock.Hash)
	bc.db.Put("Blocks", newBlock.ComputeHash(), newBlock.SerializeBlock())

}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if IsCoinBase(tx) {
		return true
	}

	prevTxs := make(map[string]Transaction)
	for _, input := range tx.Vin {
		prevTx, _ := bc.FindTransaction(input.Txid)
		prevTxs[string(input.Txid)] = prevTx

	}
	return tx.Verify(prevTxs)
}

func (bc *BlockChain) FindTransaction(txID []byte) (Transaction, error) {
	for _, b := range bc.blocks {
		for _, tx := range b.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return *tx, nil
			}

		}
	}
	return Transaction{}, fmt.Errorf("Transaction does not exist")
}

func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func NewBlockChain(pubkeyHash []byte, path string) (bs *BlockChain) {
	db, err := database.NewDB(path)
	if err != nil {
		panic("failed to initialize database")
	}

	db.CreateBucket("blocks")
	db.CreateBucket("UTXO")
	coinbase := NewCoinBase(pubkeyHash)
	Genesisblock := NewGenesisBlock(coinbase)
	err = db.Put("blocks", Genesisblock.ComputeHash(), Genesisblock.SerializeBlock())
	if err != nil {
		panic("failed to add genesis block")
	}

	return &BlockChain{db: db}
}

func (bc *BlockChain) Print() {
	for i, block := range bc.blocks {
		fmt.Printf("========= 区块 %d =========\n", i)
		fmt.Printf("上一个区块哈希: %x\n", block.PrevHash)
		fmt.Printf("当前区块哈希: %x\n", block.Hash)
		fmt.Println()
	}
}
