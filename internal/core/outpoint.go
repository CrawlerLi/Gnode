package core

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// OutPoint uniquely identifies a transaction output.
type OutPoint struct {
	TxID     []byte
	OutIndex int
}

func NewOutPoint(txID []byte, outIndex int) OutPoint {
	id := make([]byte, len(txID))
	copy(id, txID)
	return OutPoint{
		TxID:     id,
		OutIndex: outIndex,
	}
}

func (o OutPoint) String() string {
	return fmt.Sprintf("%x:%d", o.TxID, o.OutIndex)
}

func ParseOutPoint(encoded string) (OutPoint, error) {
	parts := strings.Split(encoded, ":")
	if len(parts) != 2 {
		return OutPoint{}, fmt.Errorf("outpoint should be txid:index")
	}

	txID, err := hex.DecodeString(parts[0])
	if err != nil {
		return OutPoint{}, fmt.Errorf("invalid txid: %w", err)
	}

	outIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return OutPoint{}, fmt.Errorf("invalid out index: %w", err)
	}

	return NewOutPoint(txID, outIndex), nil
}

func EncodeUTXOKey(txID []byte, outIndex int) []byte {
	outPoint := NewOutPoint(txID, outIndex)
	return []byte(outPoint.String())
}

func DecodeUTXOKey(key []byte) (OutPoint, error) {
	return ParseOutPoint(string(key))
}
