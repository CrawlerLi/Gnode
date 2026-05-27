package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/utils"
)

func NewTrasaction(wallet *crypto.Wallet, to string, amount int, bc *core.BlockChain) (*core.Transaction, error) {
	var tx *core.Transaction

	pubkeyHash := crypto.HashPubkey(wallet.Publickey)

	payable, acc, err := bc.UTXO.FindSpendableUTXOS(amount, pubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("Failed to find payable UTXO: %s", err)
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

	TopubkeyHash := utils.Base58decode([]byte(to))
	TopubkeyHash = TopubkeyHash[1 : len(TopubkeyHash)-4]

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

	tx.ID = tx.Hash()

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
		txCopy.Vin[index].Pubkey = prevOutputs[core.OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()].ScriptPubkey
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

func GetBalance(bc *core.BlockChain, address string) (int, error) {
	var balance int
	pubkeyHash := utils.Base58decode([]byte(address))
	pubkeyHash = pubkeyHash[1 : len(pubkeyHash)-4]

	utxos, err := bc.UTXO.Snapshot()
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
