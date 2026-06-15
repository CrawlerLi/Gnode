package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math/big"

	"github.com/CrawlerLi/Gnode/internal/core"
	"github.com/CrawlerLi/Gnode/pkg/crypto"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Publickey  []byte
	Address    []byte
	Role       string
}

type serializedWallet struct {
	PrivateD  []byte
	PublicX   []byte
	PublicY   []byte
	Publickey []byte
	Address   []byte
	Role      string
}

func NewWallet(role string) (*Wallet, error) {

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
		Role:       role,
	}

	return wallet, nil
}

func (w *Wallet) SerializeWallet() ([]byte, error) {
	if w == nil || w.PrivateKey == nil {
		return nil, fmt.Errorf("encode wallet: empty wallet private key")
	}

	//there may be some probelms
	sw := serializedWallet{
		PrivateD:  w.PrivateKey.D.Bytes(),
		PublicX:   w.PrivateKey.PublicKey.X.Bytes(),
		PublicY:   w.PrivateKey.PublicKey.Y.Bytes(),
		Publickey: w.Publickey,
		Address:   w.Address,
		Role:      w.Role,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(sw)
	if err != nil {
		return nil, fmt.Errorf("encode walllet: %w", err)
	}
	return buffer.Bytes(), nil
}

func DeserializedWallet(key []byte) (*Wallet, error) {
	var sw serializedWallet
	decoder := gob.NewDecoder(bytes.NewReader(key))
	err := decoder.Decode(&sw)
	if err != nil {
		return nil, fmt.Errorf("decode wallet: %w", err)
	}

	privateKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(sw.PublicX),
			Y:     new(big.Int).SetBytes(sw.PublicY),
		},
		D: new(big.Int).SetBytes(sw.PrivateD),
	}

	w := Wallet{
		PrivateKey: privateKey,
		Publickey:  sw.Publickey,
		Address:    sw.Address,
		Role:       sw.Role,
	}
	return &w, nil
}

func (w *Wallet) Sign(tx *core.Transaction, prevOutputs map[string]core.TxOutput) error {
	if core.IsCoinBase(tx) {
		return nil
	}

	for index, input := range tx.Vin {
		txCopy := tx.TrimTx()

		prevOutput, ok := prevOutputs[core.OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()]
		if !ok {
			return fmt.Errorf("sign transaction: previous output not found")
		}
		txCopy.Vin[index].Pubkey = prevOutput.ScriptPubkey

		txID, err := txCopy.Hash()
		if err != nil {
			return fmt.Errorf("sign transaction: hash pending transaction: %w", err)
		}

		signHash := sha256.Sum256(txID)

		r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, signHash[:])
		if err != nil {
			return fmt.Errorf("sign transaction: ecdsa sign: %w", err)
		}

		tx.Vin[index].Signature = append(r.Bytes(), s.Bytes()...)
	}

	return nil

}
