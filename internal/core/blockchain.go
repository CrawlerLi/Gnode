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

	bc.db.Update(func(tx database.Transaction) error {

	}
	)

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
