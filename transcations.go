package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
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

func NewTrasaction(wallet *Wallet, to string, amount int, bc *BlockChain) *Transaction {
	var tx *Transaction

	pubkeyHash := HashPubkey(wallet.Publickey)

	payable, acc := bc.FindSpendableUTXOS(amount, pubkeyHash)
	if acc < amount {
		fmt.Println("balance of the address is not enough")
	}

	var Vin []TxInput

	for txIDHex, idxs := range payable {
		for _, idx := range idxs {
			txID, _ := hex.DecodeString(txIDHex)
			txin := TxInput{
				Txid:     txID,
				OutIndex: idx,
				Pubkey:   wallet.Publickey,
			}
			Vin = append(Vin, txin)
		}
	}

	TopubkeyHash := Base58decode([]byte(to))
	TopubkeyHash = TopubkeyHash[1 : len(TopubkeyHash)-4]

	txout := TxOutput{
		Value:        amount,
		ScriptPubkey: TopubkeyHash,
	}

	Vout := []TxOutput{txout}

	if acc > amount {
		Vout = append(Vout, TxOutput{acc - amount, pubkeyHash})
	}

	tx = &Transaction{
		Vin:  Vin,
		Vout: Vout,
	}

	tx.ID = tx.Hash()

	prevTxs := make(map[string]Transaction)
	for _, in := range Vin {
		prevTx, err := bc.FindTransaction(in.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTxs[string(in.Txid)] = prevTx
	}

	tx.Sign(prevTxs, wallet.PrivateKey)

	return tx
}

func (tx *Transaction) Sign(prevTXs map[string]Transaction, privateKey *ecdsa.PrivateKey) {
	if IsCoinBase(tx) {
		return
	}

	txCopy := tx.TrimTx()

	for index, input := range txCopy.Vin {
		txCopy.Vin[index].Pubkey = prevTXs[string(input.Txid)].Vout[input.OutIndex].ScriptPubkey
		txCopy.ID = txCopy.Hash()
		sighHash := sha256.Sum256(txCopy.ID)

		r, s, err := ecdsa.Sign(rand.Reader, privateKey, sighHash[:])
		if err != nil {
			log.Panic(err)
		}

		tx.Vin[index].Signature = append(r.Bytes(), s.Bytes()...)
	}

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

func HashPubkey(pubkey []byte) []byte {
	sha256Pubkey := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(sha256Pubkey[:])
	if err != nil {
		log.Panic(err)
	}

	return ripemd160Hasher.Sum(nil)

}

func IsCoinBase(tx *Transaction) bool {
	return len(tx.Vin[0].Txid) == 0 && tx.Vin[0].OutIndex == -1
}
