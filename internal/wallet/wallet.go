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

func NewTrasaction(wallet *crypto.Wallet, to string, amount int, bc *core.BlockChain) *core.Transaction {
	var tx *core.Transaction

	pubkeyHash := crypto.HashPubkey(wallet.Publickey)

	payable, acc := core.FindSpendableUTXOS(amount, pubkeyHash, bc)
	if acc < amount {
		fmt.Println("balance of the address is not enough")
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

	prevTxs := make(map[string]core.Transaction)
	for _, in := range Vin {
		prevTx, err := bc.FindTransaction(in.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTxs[string(in.Txid)] = prevTx
	}

	Sign(tx, prevTxs, wallet.PrivateKey)

	return tx
}

func Sign(tx *core.Transaction, prevTXs map[string]core.Transaction, privateKey *ecdsa.PrivateKey) {
	if core.IsCoinBase(tx) {
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

func GetBalance(bc *core.BlockChain, address string) int {
	var balance int
	pubkeyHash := utils.Base58decode([]byte(address))
	pubkeyHash = pubkeyHash[1 : len(pubkeyHash)-4]

	utxos := core.FindAllUTXO(bc)
	for _, utxo := range utxos {
		if string(utxo.ScriptPubkey) == string(pubkeyHash) {
			balance += utxo.Value
		}
	}

	return balance
}
