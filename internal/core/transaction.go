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

func (tx *Transaction) Hash() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil {
		fmt.Println(err)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

func (tx *Transaction) SerializeTxOutput() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil {
		fmt.Println(err)
	}

	return buf.Bytes()
}

func DeserializeTxOutput(bytesOutput []byte) *TxOutput {
	var txo TxOutput
	dec := gob.NewDecoder(bytes.NewReader(bytesOutput))
	err := dec.Decode(&txo)
	if err != nil {
		fmt.Println(err)
	}

	return &txo
}

func NewCoinBase(pubkeyHash []byte) *Transaction {
	nonce := make([]byte, 8)
	rand.Read(nonce)

	txin := TxInput{
		Txid:      []byte{},
		OutIndex:  -1,
		Signature: nonce,
		Pubkey:    nil,
	}

	txout := TxOutput{
		Value:        50,
		ScriptPubkey: pubkeyHash,
	}

	tx := &Transaction{
		Vin:  []TxInput{txin},
		Vout: []TxOutput{txout},
	}

	tx.ID = tx.Hash()

	return tx
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if IsCoinBase(tx) {
		return true
	}

	txCopy := tx.TrimTx()
	for idx, input := range tx.Vin {
		txCopy.Vin[idx].Pubkey = prevTxs[string(input.Txid)].Vout[input.OutIndex].ScriptPubkey
		txCopy.ID = txCopy.Hash()
		verifyHash := sha256.Sum256(txCopy.ID)

		r, s := &big.Int{}, &big.Int{}
		siglen := len(input.Signature) / 2
		r.SetBytes(input.Signature[:siglen])
		s.SetBytes(input.Signature[siglen:])

		x := new(big.Int).SetBytes(input.Pubkey[:32])
		y := new(big.Int).SetBytes(input.Pubkey[32:])

		pubKey := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}

		if !ecdsa.Verify(pubKey, verifyHash[:], r, s) {
			return false
		}

	}

	return true

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
	return len(tx.Vin[0].Txid) == 0 && tx.Vin[0].OutIndex == -1
}
