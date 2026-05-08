package main

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
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

func (bc *BlockChain) FindAllUTXO() map[string]TxOutput {
	utxo := make(map[string]TxOutput)
	spentTxos := make(map[string]bool)

	for _, block := range bc.blocks {
		for _, tx := range block.Transactions {
			txid := tx.ID
			txidHex := fmt.Sprintf("%x", txid)
			for i, txo := range tx.Vout {
				key := fmt.Sprintf("%s:%d", txidHex, i)
				if !spentTxos[key] {
					utxo[key] = txo
				}
			}

			if !IsCoinBase(tx) {
				for _, txi := range tx.Vin {
					txidHex := fmt.Sprintf("%x", txi.Txid)
					key := fmt.Sprintf("%s:%d", txidHex, txi.OutIndex)
					spentTxos[key] = true
					delete(utxo, key)
				}

			}

		}
	}

	return utxo
}

func (bc *BlockChain) GetBalance(address string) int {
	var balance int
	pubkeyHash := Base58decode([]byte(address))
	pubkeyHash = pubkeyHash[1 : len(pubkeyHash)-4]

	utxos := bc.FindAllUTXO()
	for _, utxo := range utxos {
		if string(utxo.ScriptPubkey) == string(pubkeyHash) {
			balance += utxo.Value
		}
	}

	return balance
}

func (bc *BlockChain) FindSpendableUTXOS(amount int, pubkeyHash []byte) (map[string][]int, int) {

	payable := make(map[string][]int)
	acc := 0

	utxos := bc.FindAllUTXO()
	for key, output := range utxos {
		if bytes.Equal(pubkeyHash, output.ScriptPubkey) {

			parts := strings.Split(key, ":")

			txid := parts[0]
			outidx := parts[1]

			acc += output.Value
			outidxInt, _ := strconv.Atoi(outidx)
			payable[txid] = append(payable[txid], outidxInt)

			if acc >= amount {
				break
			}
		}
	}

	return payable, acc

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
