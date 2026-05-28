package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"

	"fmt"
	"math/big"
)

type TxInput struct {
	Txid      []byte
	OutIndex  int
	Signature []byte
	Pubkey    []byte
}

type TxOutput struct {
	Value        int
	ScriptPubkey []byte
}

type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

const (
	coinbaseNonceSize = 8
	coinbaseOutIndex  = -1
	coinbaseReward    = 50
	pubKeyCoordLen    = 32
	pubKeyLen         = pubKeyCoordLen * 2
)

func (tx *Transaction) Hash() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("encode transaction: %w", err)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hash[:], nil
}

func (tx *Transaction) SerializeTxOutput(outindex int) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx.Vout[outindex])
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction output: %w", err)
	}

	return buf.Bytes(), nil
}

func DeserializeTxOutput(bytesOutput []byte) (TxOutput, error) {
	var txo TxOutput
	dec := gob.NewDecoder(bytes.NewReader(bytesOutput))
	err := dec.Decode(&txo)
	if err != nil {
		return TxOutput{}, err
	}

	return txo, nil
}

func NewCoinBase(pubkeyHash []byte) (*Transaction, error) {
	nonce := make([]byte, coinbaseNonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("new coinbase tx: generate nonce: %w", err)
	}

	txin := TxInput{
		Txid:      []byte{},
		OutIndex:  coinbaseOutIndex,
		Signature: nonce,
		Pubkey:    nil,
	}

	txout := TxOutput{
		Value:        coinbaseReward,
		ScriptPubkey: pubkeyHash,
	}

	tx := &Transaction{
		Vin:  []TxInput{txin},
		Vout: []TxOutput{txout},
	}

	txID, err := tx.Hash()
	tx.ID = txID

	if err != nil {
		return nil, fmt.Errorf("new coinbase tx: get coinbase tx hash  %w", err)
	}

	return tx, nil
}

func (tx *Transaction) Verify(prevOutputs map[string]TxOutput) error {
	if IsCoinBase(tx) {
		return nil
	}

	txCopy := tx.TrimTx()
	for idx, input := range tx.Vin {

		prevOutput, ok := prevOutputs[OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()]
		if !ok {
			return fmt.Errorf("verify transaction: previous output not found")
		}
		txCopy.Vin[idx].Pubkey = prevOutput.ScriptPubkey
		txID, err := txCopy.Hash()
		if err != nil {
			return fmt.Errorf("failed to verify transaction: %w", err)
		}
		txCopy.ID = txID

		verifyHash := sha256.Sum256(txCopy.ID)

		if len(input.Signature) == 0 || len(input.Signature)%2 != 0 {
			return fmt.Errorf("verify transaction: invalid signature length")
		}
		if len(input.Pubkey) != pubKeyLen {
			return fmt.Errorf("verify transaction: invalid public key length")
		}

		r, s := &big.Int{}, &big.Int{}
		siglen := len(input.Signature) / 2
		r.SetBytes(input.Signature[:siglen])
		s.SetBytes(input.Signature[siglen:])

		x := new(big.Int).SetBytes(input.Pubkey[:pubKeyCoordLen])
		y := new(big.Int).SetBytes(input.Pubkey[pubKeyCoordLen:])

		pubKey := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}

		if !ecdsa.Verify(pubKey, verifyHash[:], r, s) {
			return fmt.Errorf("verify transaction: ecdsa verification failed")
		}

	}

	return nil

}

func (tx *Transaction) TrimTx() (txcopy *Transaction) {
	var copyVin []TxInput
	var copyVout []TxOutput
	for _, txi := range tx.Vin {
		copyVin = append(copyVin, TxInput{txi.Txid, txi.OutIndex, nil, nil})
	}

	for _, txo := range tx.Vout {
		copyVout = append(copyVout, TxOutput{txo.Value, txo.ScriptPubkey})
	}

	return &Transaction{nil, copyVin, copyVout}
}

func IsCoinBase(tx *Transaction) bool {
	return len(tx.Vin[0].Txid) == 0 && tx.Vin[0].OutIndex == coinbaseOutIndex
}
