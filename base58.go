package main

import (
	"math/big"
)

const Base58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func Base58encode(fullPayload []byte) []byte {

	var result []byte

	x := big.NewInt(0).SetBytes(fullPayload)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, Base58[mod.Int64()])
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	for b := range fullPayload {
		if b == 0x00 {
			result = append([]byte{Base58[0]}, result...)
		} else {
			break
		}
	}

	return result

}

func Base58decode(address []byte) []byte {
	x := big.NewInt(0)
	base := big.NewInt(58)

	for i := range address {
		for j := range Base58 {
			if Base58[j] == address[i] {
				x.Mul(x, base)
				x.Add(x, big.NewInt(int64(j)))
			}
		}
	}

	res := x.Bytes()

	for i := range address {
		if address[i] == Base58[0] {
			res = append([]byte{0x00}, res...)
		} else {
			break
		}
	}

	return res
}
