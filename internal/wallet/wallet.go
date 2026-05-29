package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Publickey  []byte
	Address    []byte
}

const (
	addressVersionLen  = 1
	addressChecksumLen = 4
	pubKeyHashLen      = 20
)

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

func NewTrasaction(wallet *Wallet, to string, amount int, bc *core.BlockChain) (*core.Transaction, error) {
	var tx *core.Transaction

	pubkeyHash := crypto.HashPubkey(wallet.Publickey)

	payable, acc, err := bc.UTXO.FindSpendableUTXOS(amount, pubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("Failed to find payable UTXO: %w", err)
	}

	if acc < amount {
		return nil, fmt.Errorf("banlance do not enough")
	}

	var Vin []core.TxInput
	prevOutputs := make(map[string]core.TxOutput)

	for _, spendableUTXO := range payable {

		txID, idx := spendableUTXO.OutPoint.TxID, spendableUTXO.OutPoint.OutIndex
		txin := core.TxInput{
			Txid:     txID,
			OutIndex: idx,
			Pubkey:   wallet.Publickey,
		}

		Vin = append(Vin, txin)
		prevOutputs[spendableUTXO.OutPoint.String()] = spendableUTXO.Output

	}

	TopubkeyHash, err := crypto.AddressToPubkeyHash([]byte(to))
	if err != nil {
		return nil, fmt.Errorf("create new tx: convert recipient address to pubkey hash: %w", err)
	}
	if len(TopubkeyHash) != pubKeyHashLen {
		return nil, fmt.Errorf("create new tx: invalid recipient pubkey hash length")
	}

	txout := core.TxOutput{
		Value:        amount,
		ScriptPubkey: TopubkeyHash,
	}

	Vout := []core.TxOutput{txout}

	if acc > amount {
		Vout = append(Vout, core.TxOutput{Value: acc - amount, ScriptPubkey: pubkeyHash})
	}

	tx = &core.Transaction{
		Vin:  Vin,
		Vout: Vout,
	}

	txID, err := tx.Hash()
	if err != nil {
		return nil, fmt.Errorf("create new tx: hash new tx: %w", err)
	}
	tx.ID = txID

	err = Sign(tx, prevOutputs, wallet.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("create new tx: sign new tx: %w", err)
	}

	return tx, nil
}

func Sign(tx *core.Transaction, prevOutputs map[string]core.TxOutput, privateKey *ecdsa.PrivateKey) error {
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

		r, s, err := ecdsa.Sign(rand.Reader, privateKey, sighHash[:])
		if err != nil {
			return fmt.Errorf("sign transaction: ecdsa sign: %w", err)
		}

		tx.Vin[index].Signature = append(r.Bytes(), s.Bytes()...)
	}

	return nil

}

func GetBalance(bc *core.BlockChain, address string) (int, error) {
	var balance int
	pubkeyHash, err := crypto.AddressToPubkeyHash([]byte(address))
	if err != nil {
		return 0, fmt.Errorf("get balance: convert address to pubkey hash: %w", err)
	}

	utxos, err := bc.UTXO.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("fail to snapshot UTXO: %w", err)
	}
	for _, utxo := range utxos {
		if string(utxo.ScriptPubkey) == string(pubkeyHash) {
			balance += utxo.Value
		}
	}

	return balance, nil
}
