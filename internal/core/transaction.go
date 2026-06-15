package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"

	"fmt"
	"math/big"
)

type TxInput struct {
	Txid      []byte
	OutIndex  int
	Signature []byte
	Pubkey    []byte
}

type TxOutput struct {
	Value        int
	ScriptPubkey []byte
}

type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

const (
	coinbaseNonceSize = 8
	coinbaseOutIndex  = -1
	coinbaseReward    = 50
	pubKeyCoordLen    = 32
	pubKeyLen         = pubKeyCoordLen * 2
)

func (tx *Transaction) Hash() ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("hash transaction: nil transaction")
	}

	txBytes, err := tx.canonicalBytes()
	if err != nil {
		return nil, fmt.Errorf("hash transaction: encode transaction: %w", err)
	}

	hash := sha256.Sum256(txBytes)
	return hash[:], nil
}

func (tx *Transaction) canonicalBytes() ([]byte, error) {
	var buf bytes.Buffer

	// ID is intentionally excluded: it is the result of hashing this payload.
	if err := writeUint64(&buf, uint64(len(tx.Vin))); err != nil {
		return nil, err
	}
	for _, vin := range tx.Vin {
		if err := writeBytes(&buf, vin.Txid); err != nil {
			return nil, err
		}
		if err := writeInt64(&buf, int64(vin.OutIndex)); err != nil {
			return nil, err
		}
		if err := writeBytes(&buf, vin.Signature); err != nil {
			return nil, err
		}
		if err := writeBytes(&buf, vin.Pubkey); err != nil {
			return nil, err
		}
	}

	if err := writeUint64(&buf, uint64(len(tx.Vout))); err != nil {
		return nil, err
	}
	for _, vout := range tx.Vout {
		if err := writeInt64(&buf, int64(vout.Value)); err != nil {
			return nil, err
		}
		if err := writeBytes(&buf, vout.ScriptPubkey); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func writeBytes(buf *bytes.Buffer, data []byte) error {
	if err := writeUint64(buf, uint64(len(data))); err != nil {
		return err
	}
	_, err := buf.Write(data)
	return err
}

func writeInt64(buf *bytes.Buffer, value int64) error {
	return binary.Write(buf, binary.BigEndian, value)
}

func writeUint64(buf *bytes.Buffer, value uint64) error {
	return binary.Write(buf, binary.BigEndian, value)
}

func (tx *Transaction) SerializeTxOutput(outindex int) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx.Vout[outindex])
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction output: %w", err)
	}

	return buf.Bytes(), nil
}

func DeserializeTxOutput(bytesOutput []byte) (TxOutput, error) {
	var txo TxOutput
	dec := gob.NewDecoder(bytes.NewReader(bytesOutput))
	err := dec.Decode(&txo)
	if err != nil {
		return TxOutput{}, err
	}

	return txo, nil
}

func NewCoinBase(pubkeyHash []byte) (*Transaction, error) {
	nonce := make([]byte, coinbaseNonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("new coinbase tx: generate nonce: %w", err)
	}

	txin := TxInput{
		Txid:      []byte{},
		OutIndex:  coinbaseOutIndex,
		Signature: nonce,
		Pubkey:    nil,
	}

	txout := TxOutput{
		Value:        coinbaseReward,
		ScriptPubkey: pubkeyHash,
	}

	tx := &Transaction{
		Vin:  []TxInput{txin},
		Vout: []TxOutput{txout},
	}

	txID, err := tx.Hash()
	if err != nil {
		return nil, fmt.Errorf("new coinbase tx: get coinbase tx hash  %w", err)
	}
	tx.ID = txID

	return tx, nil
}

func (tx *Transaction) Verify(prevOutputs map[string]TxOutput) error {
	if IsCoinBase(tx) {
		return nil
	}

	for idx, input := range tx.Vin {

		txCopy := tx.TrimTx()

		fmt.Printf("tx copy txid is %x, nums of Vin is %d, nums of Vout is %d\n",
			txCopy.ID,
			len(txCopy.Vin),
			len(txCopy.Vout))

		prevOutput, ok := prevOutputs[OutPoint{TxID: input.Txid, OutIndex: input.OutIndex}.String()]
		fmt.Printf("prevOutput: value:%d, scirptPubkey: %x\n",
			prevOutput.Value,
			prevOutput.ScriptPubkey)
		if !ok {
			return fmt.Errorf("verify transaction: previous output not found")
		}

		fmt.Printf("OUTPOINT: input.Txid: %x, input.OutIndex: %x\n", input.Txid, input.OutIndex)

		txCopy.Vin[idx].Pubkey = prevOutput.ScriptPubkey
		fmt.Printf("Vin %d: outindex: %d, signature: %x, pulickey:%x\n",
			idx,
			txCopy.Vin[idx].OutIndex,
			txCopy.Vin[idx].Signature,
			txCopy.Vin[idx].Pubkey)

		for i, vout := range txCopy.Vout {
			fmt.Printf("Vout %d: value: %d, scriptPubkey: %x\n",
				i,
				vout.Value,
				vout.ScriptPubkey)
		}

		txID, err := txCopy.Hash()
		fmt.Printf("tx hash: %x\n", txID)

		if err != nil {
			return fmt.Errorf("failed to verify transaction: %w", err)
		}

		verifyHash := sha256.Sum256(txID)
		fmt.Printf("verifyHash: %x\n", verifyHash)

		if len(input.Signature) == 0 || len(input.Signature)%2 != 0 {
			return fmt.Errorf("verify transaction: invalid signature length")
		}
		if len(input.Pubkey) != pubKeyLen {
			return fmt.Errorf("verify transaction: invalid public key length")
		}

		r, s := &big.Int{}, &big.Int{}
		siglen := len(input.Signature) / 2
		r.SetBytes(input.Signature[:siglen])
		s.SetBytes(input.Signature[siglen:])

		x := new(big.Int).SetBytes(input.Pubkey[:pubKeyCoordLen])
		y := new(big.Int).SetBytes(input.Pubkey[pubKeyCoordLen:])

		pubKey := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}

		fmt.Printf("pubkey, x: %x, y: %x\n", x, y)

		if !ecdsa.Verify(pubKey, verifyHash[:], r, s) {
			return fmt.Errorf("verify transaction: ecdsa verification failed")
		}

	}

	return nil

}

func (tx *Transaction) TrimTx() (txcopy *Transaction) {
	var copyVin []TxInput
	var copyVout []TxOutput
	for _, txi := range tx.Vin {
		copyVin = append(copyVin, TxInput{txi.Txid, txi.OutIndex, nil, nil})
	}

	for _, txo := range tx.Vout {
		copyVout = append(copyVout, TxOutput{txo.Value, txo.ScriptPubkey})
	}

	return &Transaction{nil, copyVin, copyVout}
}

func IsCoinBase(tx *Transaction) bool {
	return len(tx.Vin[0].Txid) == 0 && tx.Vin[0].OutIndex == coinbaseOutIndex
}
