package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"time"
)

type Block struct {
	Version    int64
	PrevHash   []byte
	TimeStamp  int64
	TargetBits int
	Nonce      int64
	Hash       []byte

	Transactions []*Transaction
}

func (b *Block) ComputeHash() []byte {
	data := bytes.Join(
		[][]byte{
			b.IntToByte(b.Version),
			b.PrevHash[:],
			b.IntToByte(b.TimeStamp),
			b.IntToByte(int64(b.TargetBits)),
			b.IntToByte(b.Nonce),
			b.SerializeTranscations(),
		},
		[]byte{},
	)

	hash := sha256.Sum256(data)
	hash = sha256.Sum256(hash[:])
	return hash[:]
}

// IntToByte converts an int64 to a byte slice.
// This function should must excute successfully,
// no need to return error
func (b *Block) IntToByte(num int64) []byte {
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, num)
	return buffer.Bytes()
}

func (b *Block) SerializeTranscations() []byte {
	var buf bytes.Buffer
	for _, tx := range b.Transactions {
		buf.Write(tx.ID)
	}
	return buf.Bytes()
}

func NewBlock(transcations []*Transaction, prevHash []byte) *Block {
	var nblock = &Block{
		Version:      1,
		PrevHash:     prevHash,
		TimeStamp:    time.Now().Unix(),
		TargetBits:   16,
		Nonce:        0,
		Transactions: transcations,
	}

	pow := NewProofOfWork(nblock)

	nonce, hash := pow.Run()

	nblock.Hash = hash
	nblock.Nonce = nonce

	return nblock
}

func (b *Block) SerializeBlock() ([]byte, error) {
	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(b)
	if err != nil {
		return nil, fmt.Errorf("failed to encode block: %w", err)
	}
	return buf.Bytes(), nil
}

func DeserializedBlock(key []byte) (*Block, error) {
	var b Block

	decoder := gob.NewDecoder(bytes.NewReader(key))

	err := decoder.Decode(&b)
	if err != nil {
		return nil, fmt.Errorf("failed to decode block: %w", err)
	}

	return &b, nil
}
