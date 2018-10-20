package crypto

import (
	"encoding/hex"
	"errors"

	"github.com/haltingstate/secp256k1-go"
	"golang.org/x/crypto/sha3"
)

// PublicKeyToUserAddr get user address from public key.
func PublicKeyToUserAddr(publicKey []byte) (string, error) {
	if len(publicKey) == 33 {
		publicKey = secp256k1.UncompressPubkey(publicKey)
	}
	if publicKey == nil || len(publicKey) != 65 {
		return "", errors.New("illegal public key")
	}
	digest := sha3.Sum256(publicKey[1:])
	return "hx" + hex.EncodeToString(digest[len(digest)-20:]), nil
}
func PublicKeyFromPrivateKey(secret []byte) []byte {
	return secp256k1.PubkeyFromSeckey(secret)
}
