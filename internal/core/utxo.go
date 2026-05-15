package core

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func FindAllUTXO(bc *BlockChain) map[string]TxOutput {
	utxo := make(map[string]TxOutput)
	spentTxos := make(map[string]bool)

	for _, block := range bc.blocks {
		for _, tx := range block.Transactions {
			txid := tx.ID
			txidHex := fmt.Sprintf("%x", txid)
			for i, txo := range tx.Vout {
				key := fmt.Sprintf("%s:%d", txidHex, i)
				if !spentTxos[key] {
					utxo[key] = txo
				}
			}

			if !IsCoinBase(tx) {
				for _, txi := range tx.Vin {
					txidHex := fmt.Sprintf("%x", txi.Txid)
					key := fmt.Sprintf("%s:%d", txidHex, txi.OutIndex)
					spentTxos[key] = true
					delete(utxo, key)
				}

			}

		}
	}

	return utxo
}

func FindSpendableUTXOS(amount int, pubkeyHash []byte, bc *BlockChain) (map[string][]int, int) {

	payable := make(map[string][]int)
	acc := 0

	utxos := FindAllUTXO(bc)
	for key, output := range utxos {
		if bytes.Equal(pubkeyHash, output.ScriptPubkey) {

			parts := strings.Split(key, ":")

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

	return payable, acc

}
