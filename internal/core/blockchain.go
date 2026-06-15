package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/CrawlerLi/Gnode/internal/infra/database"
	"github.com/CrawlerLi/Gnode/pkg/utils"
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

type ChainBlock struct {
	Height int
	Block  *Block
}

type BlockChain struct {
	DB        database.DB
	UTXO      *UTXOSet
	BestState *BestState

	mu sync.RWMutex
}

func InitBlockChain(path string) (bc *BlockChain, err error) {
	db, err := database.InitDB(path)
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

	err = newBC.DB.CreateBucket("metaData")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create metaData bucket: %w", err)
	}

	err = newBC.DB.CreateBucket("HeightIdx")
	if err != nil {
		return nil, fmt.Errorf("newBlockchain: create HeightIdx bucket: %w", err)
	}

	candidateState := &BestState{
		BlockHeight: 0,
		Hash:        []byte{},
	}

	err = newBC.DB.Update(func(tx database.Tx) error {
		err = updateBestState(candidateState, tx)
		if err != nil {
			return fmt.Errorf("failed to update best state: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("init block chain: failed to update best state: %w", err)
	}

	newBC.BestState = candidateState

	return newBC, nil
}

func (bc *BlockChain) CommitGenesisBlock(genesisBlock *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	candidateState := &BestState{
		BlockHeight: 1,
		Hash:        genesisBlock.Hash,
	}

	err := bc.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}

		heightIdxBucket := tx.Bucket("HeightIdx")
		if heightIdxBucket == nil {
			return fmt.Errorf("failed to find HeightIdx bucket")
		}

		byteBlock, err := genesisBlock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("failed to serialize genesis block: %w", err)
		}

		err = blockBucket.Put(genesisBlock.Hash, byteBlock)
		if err != nil {
			return fmt.Errorf("failed to put genesis block into bucket: %w", err)
		}

		err = updateBestState(candidateState, tx)
		if err != nil {
			return fmt.Errorf("failed to update best state: %w", err)
		}

		err = bc.UTXO.UpdateUTXO(genesisBlock, tx)
		if err != nil {
			return fmt.Errorf("failed to update UTXO set: %w", err)
		}

		heightBytes, err := utils.IntToBytes(candidateState.BlockHeight)
		if err != nil {
			return fmt.Errorf("convert genesis height: %w", err)
		}
		if err := heightIdxBucket.Put(heightBytes, genesisBlock.Hash); err != nil {
			return fmt.Errorf("put genesis height index: %w", err)
		}

		return nil

	})
	if err != nil {
		return fmt.Errorf("commit genesis block: %w", err)
	}

	bc.BestState = candidateState

	return nil
}

func OpenBlockChain(path string) (bc *BlockChain, err error) {
	db, err := database.OpenDB(path)
	if err != nil {
		return nil, fmt.Errorf("open blockchain: open database: %w", err)
	}

	var bestStateSnapshot *BestState

	err = db.View(func(tx database.Tx) error {
		metaBucket := tx.Bucket("metaData")
		if metaBucket == nil {
			return fmt.Errorf("open blockchain: failed to find metaData bucket: %w", ErrChainNotInitialized)
		}
		bestStateBytes := metaBucket.Get([]byte("bestState"))
		if bestStateBytes == nil {
			return fmt.Errorf("open blockchain: failed to find bestState in database: %w", ErrChainNotInitialized)
		}
		bestState, err := decodeBestState(bestStateBytes)
		if err != nil {
			return fmt.Errorf("open blockchain: decode best state: %w", err)
		}
		bestStateSnapshot = bestState
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	bc = &BlockChain{DB: db, UTXO: &UTXOSet{db: db}, BestState: bestStateSnapshot}
	return bc, nil
}

func (bc *BlockChain) BestSnapshot() *BestState {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if bc.BestState == nil {
		return nil
	}

	return &BestState{
		BlockHeight: bc.BestState.BlockHeight,
		Hash:        append([]byte(nil), bc.BestState.Hash...),
	}
}

func (bc *BlockChain) GetBlocksFromHeight(startHeight int, limit int) ([]ChainBlock, error) {
	//并发安全 Need to consider concurrency safety here
	state := bc.BestSnapshot()
	if state == nil {
		return nil, fmt.Errorf("blocks from height: best state is nil")
	}

	if startHeight >= state.BlockHeight {
		return nil, nil
	}

	endHeight := state.BlockHeight
	if limit > 0 && startHeight+limit < endHeight {
		endHeight = startHeight + limit
	}

	chainBlocks := []ChainBlock{}

	err := bc.DB.View(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		heightIdxBucket := tx.Bucket("HeightIdx")
		if heightIdxBucket == nil {
			return fmt.Errorf("height index bucket not found")
		}
		if blockBucket == nil {
			return fmt.Errorf("blocks bucket not found")
		}

		for h := startHeight + 1; h <= endHeight; h++ {
			hByte, err := utils.IntToBytes(h)
			if err != nil {
				return fmt.Errorf("failed to convert height from int to byte")
			}

			hashBytes := heightIdxBucket.Get(hByte)
			if hashBytes == nil {
				return fmt.Errorf("failed to find hash by height")
			}

			blockByte := blockBucket.Get(hashBytes)
			if blockByte == nil {
				return fmt.Errorf("failed to find block by hash")
			}

			block, err := DeserializedBlock(blockByte)
			if err != nil {
				return fmt.Errorf("failed to deserialize block")
			}

			chainBlocks = append(chainBlocks, ChainBlock{
				Height: h,
				Block:  block})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get blocks from height: view in db: %w", err)
	}
	return chainBlocks, nil
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for _, tx := range transactions {

		err := bc.VerifyTransaction(tx)
		if err != nil {
			return fmt.Errorf("add block: invalid transaction: %w", err)
		}
	}

	prevHash := bc.BestState.Hash
	newBlock := NewBlock(transactions, prevHash)

	//避免入库失败，保证数据一致性
	//Avoid database insertion failures and ensure data consistency
	candidate := &BestState{
		BlockHeight: bc.BestState.BlockHeight + 1,
		Hash:        newBlock.Hash,
	}

	err := bc.DB.Update(func(tx database.Tx) error {
		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}

		heightIdxBucket := tx.Bucket("HeightIdx")
		if heightIdxBucket == nil {
			return fmt.Errorf("failed to find HeightIdx bucket")
		}

		bytesBlock, err := newBlock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("serialize new block: %w", err)
		}

		err = blockBucket.Put(newBlock.Hash, bytesBlock)
		if err != nil {
			return fmt.Errorf("update new block: %w", err)
		}

		blockHeightBytes, err := utils.IntToBytes(candidate.BlockHeight)
		if err != nil {
			return fmt.Errorf("convert block height to bytes: %w", err)
		}

		err = heightIdxBucket.Put(blockHeightBytes, newBlock.Hash)
		if err != nil {
			return fmt.Errorf("update HeightIdx: %w", err)
		}

		err = bc.UTXO.UpdateUTXO(newBlock, tx)
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
		return fmt.Errorf("add block: update new block into database: %w", err)
	}

	bc.BestState = candidate

	return nil
}

func (bc *BlockChain) AcceptChainBlock(chainBlock ChainBlock) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if chainBlock.Block == nil {
		return fmt.Errorf("accept chain block: block is empty")
	}

	expectedHeight := bc.BestState.BlockHeight + 1
	if chainBlock.Height != expectedHeight {
		return fmt.Errorf("accept chain block: unexpected height got %d want %d",
			chainBlock.Height, expectedHeight)
	}

	if !bytes.Equal(chainBlock.Block.PrevHash, bc.BestState.Hash) {
		return fmt.Errorf("accept chain block: prev hash mismatch")
	}

	if !bytes.Equal(chainBlock.Block.ComputeHash(), chainBlock.Block.Hash) {
		return fmt.Errorf("accept chain block: invalid block hash, computeHash is %x, block hash is %x",
			chainBlock.Block.ComputeHash(),
			chainBlock.Block.Hash)
	}

	//未验POW，后续添加
	// pow is not verified， and it will be adder later
	pow := NewProofOfWork(chainBlock.Block)
	if !pow.Check() {
		return fmt.Errorf("accept chain block: invalid proof of work")
	}

	txs := chainBlock.Block.Transactions
	for _, tx := range txs {

		err := bc.VerifyTransaction(tx)
		if err != nil {
			return fmt.Errorf("accept chain block: invalid transaction: %w", err)
		}
	}

	//避免入库失败，保证数据一致性
	//Avoid database insertion failures and ensure data consistency
	candidate := &BestState{
		BlockHeight: bc.BestState.BlockHeight + 1,
		Hash:        chainBlock.Block.ComputeHash(),
	}
	newBlock := chainBlock.Block
	err := bc.DB.Update(func(tx database.Tx) error {

		blockBucket := tx.Bucket("blocks")
		if blockBucket == nil {
			return fmt.Errorf("failed to find blocks bucket")
		}

		HeightIdxBucket := tx.Bucket("HeightIdx")
		if HeightIdxBucket == nil {
			return fmt.Errorf("failed to find HeightIdx bucket")
		}

		bytesBlock, err := newBlock.SerializeBlock()
		if err != nil {
			return fmt.Errorf("serialize new block: %w", err)
		}
		err = blockBucket.Put(newBlock.Hash, bytesBlock)
		if err != nil {
			return fmt.Errorf("update new block: %w", err)
		}

		blockHeightBytes, err := utils.IntToBytes(candidate.BlockHeight)
		if err != nil {
			return fmt.Errorf("convert block height to bytes: %w", err)
		}

		err = HeightIdxBucket.Put(blockHeightBytes, newBlock.Hash)
		if err != nil {
			return fmt.Errorf("update HeightIdx: %w", err)
		}

		err = bc.UTXO.UpdateUTXO(newBlock, tx)
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
		return fmt.Errorf("accept chain block: update new block into database: %w", err)
	}

	bc.BestState = candidate

	return nil
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) error {
	fmt.Printf("VerifyTransaction called txid=%x coinbase=%v\n", tx.ID, IsCoinBase(tx))
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
