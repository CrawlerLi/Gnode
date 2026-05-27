package core

import (
	"bytes"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

type UTXOSet struct {
	db database.DB
}

type SpendabeleUTXO struct {
	OutPoint OutPoint
	Output   TxOutput
}

func (u *UTXOSet) UpdateUTXO(b *Block, dbTx database.Tx) error {
	bucket := dbTx.Bucket("UTXOSet")

	spentTxos := make(map[string]bool)

	for _, tx := range b.Transactions {
		for i, _ := range tx.Vout {
			key := string(EncodeUTXOKey(tx.ID, i))
			if !spentTxos[key] {
				err := bucket.Put([]byte(key), tx.SerializeTxOutput(i))
				if err != nil {
					return err
				}
			}
		}

		if !IsCoinBase(tx) {
			for _, txi := range tx.Vin {
				key := string(EncodeUTXOKey(txi.Txid, txi.OutIndex))
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

func (u *UTXOSet) Snapshot() (utxoSnapshot map[*OutPoint]TxOutput, e error) {
	utxoSnapshot = make(map[*OutPoint]TxOutput)

	err := u.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Bucket("UTXOSet")
		if bucket == nil {
			return fmt.Errorf("failed to find UTXOSet bucket")
		}
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			output, err := DeserializeTxOutput(v)
			if err != nil {
				return err
			}

			op, err := DecodeUTXOKey(k)
			if err != nil {
				return err
			}

			utxoSnapshot[&op] = output

		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return utxoSnapshot, nil
}

func (u *UTXOSet) FindSpendableUTXOS(amount int, pubkeyHash []byte) ([]SpendabeleUTXO, int, error) {

	payable := []SpendabeleUTXO{}
	acc := 0

	err := u.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Bucket("UTXOSet")
		if bucket == nil {
			return fmt.Errorf("failed to find UTXOSet bucket")
		}
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var output TxOutput
			output, err := DeserializeTxOutput(v)
			if err != nil {
				return err
			}

			if bytes.Equal(pubkeyHash, output.ScriptPubkey) {

				outPoint, err := DecodeUTXOKey(k)
				if err != nil {
					return err
				}

				acc += output.Value
				payable = append(payable, SpendabeleUTXO{OutPoint: outPoint, Output: output})

				if acc >= amount {
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return payable, acc, nil

}

func (u *UTXOSet) FindTxOutput(txID []byte, outindex int) (TxOutput, error) {
	key := EncodeUTXOKey(txID, outindex)

	var value []byte
	err := u.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Bucket("UTXOSet")
		if bucket == nil {
			return fmt.Errorf("failed to find UTXOSet bucket")
		}
		value = bucket.Get(key)
		return nil
	})
	if err != nil {
		return TxOutput{}, err
	}

	if value == nil {
		return TxOutput{}, fmt.Errorf("tx output does not exist")
	}

	txo, err := DeserializeTxOutput(value)
	if err != nil {
		return TxOutput{}, fmt.Errorf("failed to deserialize tx output")
	}
	return txo, nil
}
