package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Publickey  []byte
	Address    []byte
}

func NewWallet() (*Wallet, error) {

	var wallet *Wallet
	private, pubkey, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("new wallet: generate key pair: %w", err)
	}

	address, err := crypto.PublicKeyToAddress(pubkey)
	if err != nil {
		return nil, fmt.Errorf("new wallet: convert public key to address: %w", err)
	}

	wallet = &Wallet{
		PrivateKey: private,
		Publickey:  pubkey,
		Address:    address,
	}

	return wallet, nil
}

func (w *Wallet) SerializeWallet() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(w)
	if err != nil {
		return nil, fmt.Errorf("encode walllet: %w", err)
	}
	return buffer.Bytes(), nil
}

func DeserializedBlock(key []byte) (*Wallet, error) {
	var w *Wallet
	decoder := gob.NewDecoder(bytes.NewReader(key))
	err := decoder.Decode(w)
	if err != nil {
		return nil, fmt.Errorf("decode wallet: %w:", err)
	}
	return w, nil
}

func (w *Wallet) Sign(tx *core.Transaction, prevOutputs map[string]core.TxOutput) error {
	if core.IsCoinBase(tx) {
		return nil
	}

	txCopy := tx.TrimTx()

	for index, input := range txCopy.Vin {
		prevOutput, ok := prevOutputs[core.OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()]
		if !ok {
			return fmt.Errorf("sign transaction: previous output not found")
		}
		txCopy.Vin[index].Pubkey = prevOutput.ScriptPubkey
		txID, err := txCopy.Hash()
		if err != nil {
			return fmt.Errorf("sign transaction: hash pending transaction: %w", err)
		}
		txCopy.ID = txID
		sighHash := sha256.Sum256(txCopy.ID)

		r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, sighHash[:])
		if err != nil {
			return fmt.Errorf("sign transaction: ecdsa sign: %w", err)
		}

		tx.Vin[index].Signature = append(r.Bytes(), s.Bytes()...)
	}

	return nil

}
