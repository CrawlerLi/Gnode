package core

import (
	"bytes"
	"fmt"
	"log"
)

type BlockChain struct {
	blocks []*Block
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) {
	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("signature verify failure")
		}
	}

	prevHash := bc.blocks[len(bc.blocks)-1].Hash
	newBlock := NewBlock(transactions, prevHash)
	bc.blocks = append(bc.blocks, newBlock)
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

func NewBlockChain(pubkeyHash []byte) (bs *BlockChain) {
	coinbase := NewCoinBase(pubkeyHash)
	Genesisblock := NewGenesisBlock(coinbase)
	return &BlockChain{[]*Block{Genesisblock}}

}

func (bc *BlockChain) Print() {
	for i, block := range bc.blocks {
		fmt.Printf("========= 区块 %d =========\n", i)
		fmt.Printf("上一个区块哈希: %x\n", block.PrevHash)
		fmt.Printf("当前区块哈希: %x\n", block.Hash)
		fmt.Println()
	}
}
