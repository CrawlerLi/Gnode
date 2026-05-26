package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/utils"
)

func NewTrasaction(wallet *crypto.Wallet, to string, amount int, u *core.UTXOSet) (*core.Transaction, error) {
	var tx *core.Transaction

	pubkeyHash := crypto.HashPubkey(wallet.Publickey)

	payable, acc, err := u.FindSpendableUTXOS(amount, pubkeyHash)
	if err != nil {
		return nil, err
	}

	if acc < amount {
		return nil, fmt.Errorf("banlance do not enough")
	}

	var Vin []core.TxInput

	for txIDHex, idxs := range payable {
		for _, idx := range idxs {
			txID, _ := hex.DecodeString(txIDHex)
			txin := core.TxInput{
				Txid:     txID,
				OutIndex: idx,
				Pubkey:   wallet.Publickey,
			}
			Vin = append(Vin, txin)
		}
	}

	TopubkeyHash := utils.Base58decode([]byte(to))
	TopubkeyHash = TopubkeyHash[1 : len(TopubkeyHash)-4]

	txout := core.TxOutput{
		Value:        amount,
		ScriptPubkey: TopubkeyHash,
	}

	Vout := []core.TxOutput{txout}

	if acc > amount {
		Vout = append(Vout, core.TxOutput{acc - amount, pubkeyHash})
	}

	tx = &core.Transaction{
		Vin:  Vin,
		Vout: Vout,
	}

	tx.ID = tx.Hash()

	prevOutputs := make(map[string]core.TxOutput)
	for _, in := range Vin {
		prevOutput, err := u.FindTransaction(in.Txid, in.OutIndex)
		if err != nil {
			log.Panic(err)
		}
		prevOutputs[core.OutPoint{in.Txid, in.OutIndex}.String()] = prevOutput
	}

	err = Sign(tx, prevOutputs, wallet.PrivateKey)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func Sign(tx *core.Transaction, prevOutputs map[string]core.TxOutput, privateKey *ecdsa.PrivateKey) error {
	if core.IsCoinBase(tx) {
		return nil
	}

	txCopy := tx.TrimTx()

	for index, input := range txCopy.Vin {
		txCopy.Vin[index].Pubkey = prevOutputs[core.OutPoint{input.Txid, input.OutIndex}.String()].ScriptPubkey
		txCopy.ID = txCopy.Hash()
		sighHash := sha256.Sum256(txCopy.ID)

		r, s, err := ecdsa.Sign(rand.Reader, privateKey, sighHash[:])
		if err != nil {
			return fmt.Errorf("failed to sign tx")
		}

		tx.Vin[index].Signature = append(r.Bytes(), s.Bytes()...)
	}

	return nil

}

func GetBalance(u *core.UTXOSet, address string) (int, error) {
	var balance int
	pubkeyHash := utils.Base58decode([]byte(address))
	pubkeyHash = pubkeyHash[1 : len(pubkeyHash)-4]

	utxos, err := u.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("fail to snapshot UTXO: %s", err)
	}
	for _, utxo := range utxos {
		if string(utxo.ScriptPubkey) == string(pubkeyHash) {
			balance += utxo.Value
		}
	}

	return balance, nil
}
