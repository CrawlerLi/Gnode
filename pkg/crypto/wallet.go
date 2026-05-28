package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/pkg/utils"
	"golang.org/x/crypto/ripemd160"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Publickey  []byte
	Address    []byte
}

const (
	addressVersionByte = 0x00
	checksumLen        = 4
)

func NewWallet() (*Wallet, error) {

	var wallet *Wallet
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("new wallet: generate key pair: %w", err)
	}
	pubkey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	address := GetAddress(pubkey)

	wallet = &Wallet{
		PrivateKey: private,
		Publickey:  pubkey,
		Address:    address,
	}

	return wallet, nil
}

func GetAddress(pubkey []byte) (address []byte) {
	Hashpub := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, _ = ripemd160Hasher.Write(Hashpub[:])

	pubkeyHash := ripemd160Hasher.Sum(nil)
	versionPayload := append([]byte{addressVersionByte}, pubkeyHash...)

	FirstSHA := sha256.Sum256(versionPayload)
	SecondSHA := sha256.Sum256(FirstSHA[:])
	checksum := SecondSHA[:checksumLen]

	fullPayload := append(versionPayload, checksum...)

	address = utils.Base58encode(fullPayload)

	return address
}

func HashPubkey(pubkey []byte) []byte {
	sha256Pubkey := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, _ = ripemd160Hasher.Write(sha256Pubkey[:])

	return ripemd160Hasher.Sum(nil)

}
