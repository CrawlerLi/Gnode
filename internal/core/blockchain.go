package core

import (
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

type BlockChain struct {
	DB   database.DB
	UTXO *UTXOSet
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) error {
	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			return fmt.Errorf("invalid transaction: %s", tx.ID)
		}
	}

	var prevHash []byte

	err := bc.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}
		prevHash = blockBucket.Get([]byte("tip"))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to get previous block hash: %s", err)
	}

	NewBlock := NewBlock(transactions, prevHash)

	err = bc.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}
		err = blockBucket.Put(NewBlock.ComputeHash(), NewBlock.SerializeBlock())
		if err != nil {
			return fmt.Errorf("failed to update new block: %s", err)
		}

		err = blockBucket.Put([]byte("tip"), NewBlock.Hash)
		if err != nil {
			return fmt.Errorf("failed to update tip: %s", err)
		}

		err = bc.UTXO.UpdateUTXO(NewBlock, tx)
		if err != nil {
			return fmt.Errorf("failed to update UTXO set: %s", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to add new block: %s", err)
	}

	return nil

}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if IsCoinBase(tx) {
		return true
	}

	prevTxOutputs := make(map[string]TxOutput)
	for _, input := range tx.Vin {
		prevOutput, err := bc.UTXO.FindTxOutput(input.Txid, input.OutIndex)
		if err != nil {
			return false
		}
		prevTxOutputs[OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()] = prevOutput
	}
	return tx.Verify(prevTxOutputs)
}

func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func NewBlockChain(pubkeyHash []byte, path string) (bc *BlockChain, err error) {
	db, err := database.NewDB(path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %s", err)
	}

	newBC := &BlockChain{DB: db, UTXO: &UTXOSet{db: db}}

	newBC.DB.CreateBucket("blocks")
	newBC.DB.CreateBucket("UTXOSet")
	coinbase := NewCoinBase(pubkeyHash)
	Genesisblock := NewGenesisBlock(coinbase)
	err = newBC.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}
		err := blockBucket.Put(Genesisblock.ComputeHash(), Genesisblock.SerializeBlock())
		if err != nil {
			return fmt.Errorf("failed to add genesis block: %s", err)
		}

		err = blockBucket.Put([]byte("tip"), Genesisblock.Hash)
		if err != nil {
			return fmt.Errorf("failed to update tip: %s", err)
		}

		err = newBC.UTXO.UpdateUTXO(Genesisblock, tx)
		if err != nil {
			return fmt.Errorf("failed to update UTXO set: %s", err)
		}

		return nil

	})
	if err != nil {
		return nil, fmt.Errorf("failed to add genesis block: %s", err)
	}

	return newBC, nil
}

func (bc *BlockChain) Print() {
	var blocks []*Block

	err := bc.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}
		cursor := blockBucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if string(k) == "tip" {
				continue
			}
			block, err := DeserializedBlock(v)
			if err != nil {
				return fmt.Errorf("failed to deserialize block: %s", err)
			}
			blocks = append(blocks, block)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("failed to print blockchain: %s", err)
		return
	}

	for i, block := range blocks {
		fmt.Printf("========= 区块 %d =========\n", i)
		fmt.Printf("上一个区块哈希: %x\n", block.PrevHash)
		fmt.Printf("当前区块哈希: %x\n", block.Hash)
		fmt.Println()
	}
}
