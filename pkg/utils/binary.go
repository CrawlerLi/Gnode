package utils

import (
	"encoding/binary"
	"fmt"
	"math"
)

const IntBytesLen = 8

func IntToBytes(value int) ([]byte, error) {
	if value < 0 {
		return nil, fmt.Errorf("negative int value %d", value)
	}

	return Uint64ToBytes(uint64(value)), nil
}

func BytesToInt(data []byte) (int, error) {
	value, err := BytesToUint64(data)
	if err != nil {
		return 0, err
	}
	if value > uint64(math.MaxInt) {
		return 0, fmt.Errorf("uint64 value %d overflows int", value)
	}

	return int(value), nil
}

func Int64ToBytes(value int64) ([]byte, error) {
	if value < 0 {
		return nil, fmt.Errorf("negative int64 value %d", value)
	}

	return Uint64ToBytes(uint64(value)), nil
}

func BytesToInt64(data []byte) (int64, error) {
	value, err := BytesToUint64(data)
	if err != nil {
		return 0, err
	}
	if value > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("uint64 value %d overflows int64", value)
	}

	return int64(value), nil
}

func Uint64ToBytes(value uint64) []byte {
	var buf [IntBytesLen]byte
	binary.BigEndian.PutUint64(buf[:], value)
	return buf[:]
}

func BytesToUint64(data []byte) (uint64, error) {
	if len(data) != IntBytesLen {
		return 0, fmt.Errorf("invalid uint64 byte length %d", len(data))
	}

	return binary.BigEndian.Uint64(data), nil
}
