package core

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

type BestState struct {
	BlockHeight int
	Hash        []byte
}

func newBestState(blockHeight int, hash []byte) *BestState {
	return &BestState{
		BlockHeight: blockHeight,
		Hash:        hash,
	}
}

func (bs *BestState) encodeBestState() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(bs)
	if err != nil {
		return nil, fmt.Errorf("encode best state: %w", err)
	}

	return buffer.Bytes(), nil
}

func decodeBestState(key []byte) (*BestState, error) {
	var bs BestState
	decoder := gob.NewDecoder(bytes.NewReader(key))
	err := decoder.Decode(&bs)
	if err != nil {
		return nil, fmt.Errorf("decode best state: %w", err)
	}

	return &bs, nil
}

func updateBestState(candidate *BestState, dbtx database.Tx) error {
	metaBucket := dbtx.Bucket("metaData")
	if metaBucket == nil {
		return fmt.Errorf("update best state: failed to find metaData bucket")
	}
	bestStateBytes, err := candidate.encodeBestState()
	if err != nil {
		return fmt.Errorf("update best state: encode best state: %w", err)
	}

	err = metaBucket.Put([]byte("bestState"), bestStateBytes)
	if err != nil {
		return fmt.Errorf("update best state: put best state into database: %w", err)
	}

	return nil
}

type BlockChain struct {
	DB        database.DB
	UTXO      *UTXOSet
	BestState *BestState
}

func InitBlockChain(pubkeyHash []byte, path string) (bc *BlockChain, err error) {
	db, err := database.InitDB(path)
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: initialize database: %w", err)
	}

	newBC := &BlockChain{DB: db, UTXO: &UTXOSet{db: db}, BestState: newBestState(0, []byte{})}

	err = newBC.DB.CreateBucket("blocks")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create blocks bucket: %w", err)
	}

	err = newBC.DB.CreateBucket("UTXOSet")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create UTXOSet bucket: %w", err)
	}

	err = newBC.DB.CreateBucket("metaData")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create metaData bucket: %w", err)
	}

	coinbase, err := NewCoinBase(pubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewGensisBlock : %w", err)
	}

	Genesisblock := NewGenesisBlock(coinbase)
	bestState := newBestState(1, Genesisblock.Hash)

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

		err = updateBestState(bestState, tx)
		if err != nil {
			return fmt.Errorf("failed to update best state: %w", err)
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

	newBC.BestState = bestState

	return newBC, nil
}

func OpenBlockChain(path string) (bc *BlockChain, err error) {
	db, err := database.OpenDB(path)
	if err != nil {
		return nil, fmt.Errorf("OpenBlockChain: open database: %w", err)
	}

	var bestStateSnapshot *BestState

	err = db.View(func(tx database.Tx) error {
		metaBucket := tx.Bucket("metaData")
		if metaBucket == nil {
			return fmt.Errorf("OpenBlockChain: failed to find metaData bucket: %w", ErrChainNotInitialized)
		}
		bestStateBytes := metaBucket.Get([]byte("bestState"))
		if bestStateBytes == nil {
			return fmt.Errorf("OpenBlockChain: failed to find bestState in database: %w", ErrChainNotInitialized)
		}
		bestState, err := decodeBestState(bestStateBytes)
		if err != nil {
			return fmt.Errorf("OpenBlockChain: decode best state: %w", err)
		}
		bestStateSnapshot = bestState
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	BC := &BlockChain{DB: db, UTXO: &UTXOSet{db: db}, BestState: bestStateSnapshot}
	return BC, nil
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) error {
	for _, tx := range transactions {

		err := bc.VerifyTransaction(tx)
		if err != nil {
			return fmt.Errorf("Add block: invalid transaction: %w", err)
		}
	}

	prevHash := bc.BestState.Hash
	NewBlock := NewBlock(transactions, prevHash)
	candidate := &BestState{
		BlockHeight: bc.BestState.BlockHeight + 1,
		Hash:        NewBlock.Hash,
	}

	err := bc.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("find blocks bucket")
		}

		bytesBlock, err := NewBlock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("serialize new block: %w", err)
		}
		err = blockBucket.Put(NewBlock.Hash, bytesBlock)
		if err != nil {
			return fmt.Errorf("update new block: %w", err)
		}

		err = bc.UTXO.UpdateUTXO(NewBlock, tx)
		if err != nil {
			return fmt.Errorf("update UTXO set: %w", err)
		}

		err = updateBestState(candidate, tx)
		if err != nil {
			return fmt.Errorf("update best state: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Add block: update new block into database: %w", err)
	}

	bc.BestState = candidate

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
