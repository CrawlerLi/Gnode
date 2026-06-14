package core

import (
	"bytes"
	"math"
	"math/big"
)

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-b.TargetBits))

	pow := &ProofOfWork{b, target}
	return pow

}

func (pow *ProofOfWork) PrepareData(nonce int64) []byte {
	predata := bytes.Join(
		[][]byte{
			pow.block.IntToByte(pow.block.Version),
			pow.block.PrevHash,
			pow.block.IntToByte(pow.block.TimeStamp),
			pow.block.IntToByte(int64(pow.block.TargetBits)),
			pow.block.IntToByte(nonce),
			pow.block.SerializeTranscations(),
		},
		[]byte{},
	)

	return predata
}

func (pow *ProofOfWork) Run() (int64, []byte) {

	var hashBytes []byte
	var hashInt big.Int
	var nonce int64

	for nonce = int64(0); nonce < math.MaxInt64; nonce++ {
		pow.block.Nonce = nonce
		hashBytes = pow.block.ComputeHash()
		hashInt.SetBytes(hashBytes)
		if hashInt.Cmp(pow.target) == -1 {
			break
		}
	}
	return nonce, hashBytes
}

func (pow *ProofOfWork) Check() bool {
	var hashInt big.Int
	hash := pow.block.ComputeHash()
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(pow.target) == -1
}
