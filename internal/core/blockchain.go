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

		err := bc.VerifyTransaction(tx)
		if err != nil {
			return fmt.Errorf("Add block: invalid transaction: %w", err)
		}
	}

	var prevHash []byte

	err := bc.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("Add block: failed to find blocks bucket")
		}
		prevHash = blockBucket.Get([]byte("tip"))
		return nil
	})

	if err != nil {
		return fmt.Errorf("Add block: get previous block hash: %w", err)
	}

	NewBlock := NewBlock(transactions, prevHash)

	err = bc.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}

		bytesBlock, err := NewBlock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("failed to serialize new block: %w", err)
		}
		err = blockBucket.Put(NewBlock.Hash, bytesBlock)
		if err != nil {
			return fmt.Errorf("Add block: update new block: %w", err)
		}

		err = blockBucket.Put([]byte("tip"), NewBlock.Hash)
		if err != nil {
			return fmt.Errorf("Add block: update tip: %w", err)
		}

		err = bc.UTXO.UpdateUTXO(NewBlock, tx)
		if err != nil {
			return fmt.Errorf("Add block:update UTXO set: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Add block: update new block into database: %w", err)
	}

	return nil

}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) error {
	if IsCoinBase(tx) {
		return nil
	}

	prevTxOutputs := make(map[string]TxOutput)
	for _, input := range tx.Vin {
		prevOutput, err := bc.UTXO.FindTxOutput(input.Txid, input.OutIndex)
		if err != nil {
			return fmt.Errorf("verify transaction: find previous output: %w", err)
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
		return nil, fmt.Errorf("newBlockchain: initialize database: %w", err)
	}

	newBC := &BlockChain{DB: db, UTXO: &UTXOSet{db: db}}

	err = newBC.DB.CreateBucket("blocks")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create blocks bucket: %w", err)
	}

	err = newBC.DB.CreateBucket("UTXOSet")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create UTXOSet bucket: %w", err)
	}

	coinbase, err := NewCoinBase(pubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewGensisBlock : %w", err)
	}
	Genesisblock := NewGenesisBlock(coinbase)
	err = newBC.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}

		byteBlock, err := Genesisblock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("failed to serialize genesis block: %w", err)
		}

		err = blockBucket.Put(Genesisblock.Hash, byteBlock)
		if err != nil {
			return fmt.Errorf("failed to put genesis block into bucket: %w", err)
		}

		err = blockBucket.Put([]byte("tip"), Genesisblock.Hash)
		if err != nil {
			return fmt.Errorf("failed to update tip: %w", err)
		}

		err = newBC.UTXO.UpdateUTXO(Genesisblock, tx)
		if err != nil {
			return fmt.Errorf("failed to update UTXO set: %w", err)
		}

		return nil

	})
	if err != nil {
		return nil, fmt.Errorf("failed to add genesis block: %w", err)
	}

	return newBC, nil
}

func (bc *BlockChain) Print() error {
	var blocks []*Block

	err := bc.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf(" find blocks bucket")
		}

		LastBlockHash := blockBucket.Get([]byte("tip"))
		if LastBlockHash == nil {
			return fmt.Errorf("blocks print: check LastBlockHash")
		}

		hashPointer := LastBlockHash
		for {
			blockBytes := blockBucket.Get(hashPointer)
			if blockBytes == nil {
				return fmt.Errorf("blocks print: get block by hash: %x", hashPointer)
			}
			block, err := DeserializedBlock(blockBytes)
			if err != nil {
				return fmt.Errorf("blocks print: deserialize block: %w", err)
			}

			blocks = append(blocks, block)
			if len(block.PrevHash) == 0 {
				break
			}

			hashPointer = block.PrevHash

		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("print blockchain:view in db: %w", err)
	}

	for i, block := range blocks {
		fmt.Printf("========= 区块 %d =========\n", len(blocks)-i-1)
		fmt.Printf("当前区块哈希: %x\n", block.Hash)
		fmt.Printf("上一个区块哈希: %x\n", block.PrevHash)
		fmt.Println()
	}
	return nil
}
