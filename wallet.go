package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Publickey  []byte
	Address    []byte
}

func NewWallet() *Wallet {

	var wallet *Wallet
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubkey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	address := GetAddress(pubkey)

	wallet = &Wallet{
		PrivateKey: private,
		Publickey:  pubkey,
		Address:    address,
	}

	return wallet
}

func GetAddress(pubkey []byte) (address []byte) {
	Hashpub := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(Hashpub[:])
	if err != nil {
		log.Panic(err)
	}

	pubkeyHash := ripemd160Hasher.Sum(nil)
	versionPayload := append([]byte{0x00}, pubkeyHash...)

	FirstSHA := sha256.Sum256(versionPayload)
	SecondSHA := sha256.Sum256(FirstSHA[:])
	checksum := SecondSHA[:4]

	fullPayload := append(versionPayload, checksum...)

	address = Base58encode(fullPayload)

	return address
}
