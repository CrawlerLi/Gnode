package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
)

func newTestUTXOSet(t *testing.T, createBucket bool) *UTXOSet {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "utxo_test.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("new db failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	})

	if createBucket {
		if err := db.CreateBucket("UTXOSet"); err != nil {
			t.Fatalf("create UTXOSet bucket failed: %v", err)
		}
	}

	return &UTXOSet{db: db}
}

func mustSerializeTxOutput(t *testing.T, out TxOutput) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(out); err != nil {
		t.Fatalf("serialize TxOutput failed: %v", err)
	}
	return buf.Bytes()
}

func putRawUTXO(t *testing.T, u *UTXOSet, key string, value []byte) {
	t.Helper()

	err := u.db.Update(func(tx database.Tx) error {
		b := tx.Bucket("UTXOSet")
		if b == nil {
			return fmt.Errorf("bucket UTXOSet not found")
		}
		return b.Put([]byte(key), value)
	})
	if err != nil {
		t.Fatalf("put raw utxo failed: %v", err)
	}
}

func fetchRawUTXO(t *testing.T, u *UTXOSet, key string) []byte {
	t.Helper()

	var val []byte
	err := u.db.View(func(tx database.Tx) error {
		b := tx.Bucket("UTXOSet")
		if b == nil {
			return fmt.Errorf("bucket UTXOSet not found")
		}
		val = b.Get([]byte(key))
		return nil
	})
	if err != nil {
		t.Fatalf("fetch raw utxo failed: %v", err)
	}
	return val
}

func TestUTXOSet_FindSpendableUTXOS_OK(t *testing.T) {
	u := newTestUTXOSet(t, true)
	target := []byte("alice")
	other := []byte("bob")

	putRawUTXO(t, u, "aa:0", mustSerializeTxOutput(t, TxOutput{Value: 5, ScriptPubkey: target}))
	putRawUTXO(t, u, "bb:1", mustSerializeTxOutput(t, TxOutput{Value: 7, ScriptPubkey: target}))
	putRawUTXO(t, u, "cc:0", mustSerializeTxOutput(t, TxOutput{Value: 11, ScriptPubkey: other}))

	payable, acc, err := u.FindSpendableUTXOS(10, target)
	if err != nil {
		t.Fatalf("find spendable failed: %v", err)
	}
	if acc < 10 {
		t.Fatalf("expected acc >= 10, got %d", acc)
	}

	if _, ok := payable["aa"]; !ok {
		t.Fatalf("expected tx aa exists in payable, got %+v", payable)
	}
	if _, ok := payable["bb"]; !ok {
		t.Fatalf("expected tx bb exists in payable, got %+v", payable)
	}
	if _, ok := payable["cc"]; ok {
		t.Fatalf("unexpected tx cc exists in payable, got %+v", payable)
	}
}

func TestUTXOSet_FindSpendableUTXOS_Insufficient(t *testing.T) {
	u := newTestUTXOSet(t, true)
	target := []byte("alice")

	putRawUTXO(t, u, "aa:0", mustSerializeTxOutput(t, TxOutput{Value: 5, ScriptPubkey: target}))
	putRawUTXO(t, u, "bb:1", mustSerializeTxOutput(t, TxOutput{Value: 7, ScriptPubkey: target}))

	payable, acc, err := u.FindSpendableUTXOS(20, target)
	if err != nil {
		t.Fatalf("find spendable failed: %v", err)
	}
	if acc != 12 {
		t.Fatalf("expected acc = 12, got %d", acc)
	}
	if len(payable) != 2 {
		t.Fatalf("expected 2 payable txs, got %+v", payable)
	}
}

func TestUTXOSet_FindSpendableUTXOS_BucketNotFound(t *testing.T) {
	u := newTestUTXOSet(t, false)

	payable, acc, err := u.FindSpendableUTXOS(1, []byte("alice"))
	if err == nil {
		t.Fatalf("expected error when bucket is missing")
	}
	if payable != nil {
		t.Fatalf("expected payable nil on error, got %+v", payable)
	}
	if acc != 0 {
		t.Fatalf("expected acc 0 on error, got %d", acc)
	}
}

func TestUTXOSet_FindSpendableUTXOS_BadValue(t *testing.T) {
	u := newTestUTXOSet(t, true)

	putRawUTXO(t, u, "aa:0", []byte("not-gob-tx-output"))

	payable, acc, err := u.FindSpendableUTXOS(1, []byte("alice"))
	if err == nil {
		t.Fatalf("expected deserialize error")
	}
	if payable != nil {
		t.Fatalf("expected payable nil on error, got %+v", payable)
	}
	if acc != 0 {
		t.Fatalf("expected acc 0 on error, got %d", acc)
	}
}

func TestUTXOSet_FindTransaction_OK(t *testing.T) {
	u := newTestUTXOSet(t, true)
	txID := []byte{0x01, 0x02}
	out := TxOutput{Value: 9, ScriptPubkey: []byte("alice")}
	key := fmt.Sprintf("%x:%d", txID, 0)

	putRawUTXO(t, u, key, mustSerializeTxOutput(t, out))

	got, err := u.FindTransaction(txID, 0)
	if err != nil {
		t.Fatalf("find transaction failed: %v", err)
	}
	if got.Value != out.Value {
		t.Fatalf("unexpected value: want %d got %d", out.Value, got.Value)
	}
	if !bytes.Equal(got.ScriptPubkey, out.ScriptPubkey) {
		t.Fatalf("unexpected script pubkey: want %x got %x", out.ScriptPubkey, got.ScriptPubkey)
	}
}

func TestUTXOSet_FindTransaction_NotFound(t *testing.T) {
	u := newTestUTXOSet(t, true)

	got, err := u.FindTransaction([]byte{0x09, 0x09}, 0)
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (TxOutput{}) {
		t.Fatalf("expected zero value output, got %+v", got)
	}
}

func TestUTXOSet_FindTransaction_BucketNotFound(t *testing.T) {
	u := newTestUTXOSet(t, false)

	got, err := u.FindTransaction([]byte{0x01}, 0)
	if err == nil {
		t.Fatalf("expected bucket error")
	}
	if !strings.Contains(err.Error(), "UTXOSet bucket") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (TxOutput{}) {
		t.Fatalf("expected zero value output, got %+v", got)
	}
}

func TestUTXOSet_FindTransaction_BadValue(t *testing.T) {
	u := newTestUTXOSet(t, true)
	txID := []byte{0x0a}
	key := fmt.Sprintf("%x:%d", txID, 1)
	putRawUTXO(t, u, key, []byte("bad-output-value"))

	got, err := u.FindTransaction(txID, 1)
	if err == nil {
		t.Fatalf("expected deserialize error")
	}
	if !strings.Contains(err.Error(), "deserialize") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (TxOutput{}) {
		t.Fatalf("expected zero value output, got %+v", got)
	}
}

func TestUTXOSet_UpdateUTXO_DeleteSpent(t *testing.T) {
	u := newTestUTXOSet(t, true)
	prevTxID := []byte{0xaa, 0xbb}
	prevKey := fmt.Sprintf("%x:%d", prevTxID, 0)

	putRawUTXO(t, u, prevKey, mustSerializeTxOutput(t, TxOutput{
		Value:        8,
		ScriptPubkey: []byte("alice"),
	}))

	tx := &Transaction{
		ID: []byte{0xcc},
		Vin: []TxInput{
			{
				Txid:     prevTxID,
				OutIndex: 0,
			},
		},
		Vout: []TxOutput{
			{Value: 6, ScriptPubkey: []byte("bob")},
		},
	}
	block := &Block{Transactions: []*Transaction{tx}}

	err := u.db.Update(func(dbTx database.Tx) error {
		return u.UpdateUTXO(block, dbTx)
	})
	if err != nil {
		t.Fatalf("update utxo failed: %v", err)
	}

	if val := fetchRawUTXO(t, u, prevKey); val != nil {
		t.Fatalf("expected spent utxo deleted, got value %x", val)
	}
}

func TestUTXOSet_UpdateUTXO_BucketNotFound(t *testing.T) {
	u := newTestUTXOSet(t, false)

	tx := &Transaction{
		ID:  []byte{0x01},
		Vin: []TxInput{{Txid: []byte{}, OutIndex: -1}},
		Vout: []TxOutput{
			{Value: 1, ScriptPubkey: []byte("alice")},
		},
	}
	block := &Block{Transactions: []*Transaction{tx}}

	err := u.db.Update(func(dbTx database.Tx) error {
		return u.UpdateUTXO(block, dbTx)
	})
	if err == nil {
		t.Fatalf("expected error when bucket missing")
	}
}

