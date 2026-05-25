package core

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

type UTXOSet struct {
	db database.DB
}

func (u *UTXOSet) UpdateUTXO(b *Block, dbTx database.Tx) error {
	bucket := dbTx.Bucket("UTXOSet")

	spentTxos := make(map[string]bool)

	for _, tx := range b.Transactions {
		txid := tx.ID
		txidHex := fmt.Sprintf("%x", txid)
		for i, _ := range tx.Vout {
			key := fmt.Sprintf("%s:%d", txidHex, i)
			if !spentTxos[key] {
				err := bucket.Put([]byte(key), tx.SerializeTxOutput())
				if err != nil {
					return err
				}
			}
		}

		if !IsCoinBase(tx) {
			for _, txi := range tx.Vin {
				txidHex := fmt.Sprintf("%x", txi.Txid)
				key := fmt.Sprintf("%s:%d", txidHex, txi.OutIndex)
				spentTxos[key] = true
				err := bucket.Delete([]byte(key))
				if err != nil {
					return err
				}
			}

		}

	}

	return nil
}

func (u *UTXOSet) FindSpendableUTXOS(amount int, pubkeyHash []byte) (map[string][]int, int) {

	payable := make(map[string][]int)
	acc := 0

	u.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Bucket("UTXOSet")
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var output TxOutput
			output = *DeserializeTxOutput(v)

			if bytes.Equal(pubkeyHash, output.ScriptPubkey) {

				parts := strings.Split(string(k), ":")

				txid := parts[0]
				outidx := parts[1]

				acc += output.Value
				outidxInt, _ := strconv.Atoi(outidx)
				payable[txid] = append(payable[txid], outidxInt)

				if acc >= amount {
					break
				}
			}
		}

		return nil
	})

	return payable, acc

}


func (u *UTXOSet) FindTransaction(txID []byte, outindex int) (Transaction, error) {
	for _, b := range bc.blocks {
		for _, tx := range b.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return *tx, nil
			}

		}
	}
	return Transaction{}, fmt.Errorf("Transaction does not exist")
}
