package crypto

import (
	"crypto/sha256"
	"fmt"

	"github.com/CrawlerLi/Gnode/pkg/utils"
	"golang.org/x/crypto/ripemd160"
)

const (
	addressVersionByte = 0x00
	addressVersionLen  = 1
	checksumLen        = 4
)

func HashPubkey(pubkey []byte) []byte {
	sha256Pubkey := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, _ = ripemd160Hasher.Write(sha256Pubkey[:])

	return ripemd160Hasher.Sum(nil)
}

func PublicKeyToAddress(pubkey []byte) (address []byte, err error) {
	Hashpub := sha256.Sum256(pubkey)
	ripemd160Hasher := ripemd160.New()
	_, err = ripemd160Hasher.Write(Hashpub[:])
	if err != nil {
		return nil, fmt.Errorf("public key to address: write to ripemd160 hasher: %w", err)
	}

	pubkeyHash := ripemd160Hasher.Sum(nil)
	versionPayload := append([]byte{addressVersionByte}, pubkeyHash...)

	FirstSHA := sha256.Sum256(versionPayload)
	SecondSHA := sha256.Sum256(FirstSHA[:])
	checksum := SecondSHA[:checksumLen]

	fullPayload := append(versionPayload, checksum...)

	address = utils.Base58encode(fullPayload)

	return address, nil
}

func AddressToPubkeyHash(address []byte) ([]byte, error) {
	raw := utils.Base58decode(address)

	if len(raw) < addressVersionLen+checksumLen {
		return nil, fmt.Errorf("address to pubkey hash: invalid address length")
	}
	pubkeyHash := raw[addressVersionLen : len(raw)-checksumLen]
	return pubkeyHash, nil
}
