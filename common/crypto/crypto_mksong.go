package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"

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

// RecoverPublicKey recover public key from hash of message and signature.
func RecoverPublicKey(hash []byte, signature []byte) ([]byte, error) {
	if len(signature) != 65 {
		return nil, errors.New("signature length is invalid")
	}
	if hash == nil || len(hash) > 32 {
		return nil, errors.New("message hash is illegal")
	}
	publicKey := secp256k1.RecoverPubkey(hash[:], signature)
	return publicKey, nil
}

// SignWithPrivateKey sign hash data with private key.
func SignWithPrivateKey(hash []byte, privateKey []byte) ([]byte, error) {
	if hash == nil || len(hash) != 32 || privateKey == nil ||
		len(privateKey) != 32 {
		return nil, errors.New("Invalid arguments")
	}
	return secp256k1.Sign(hash, privateKey), nil
}

func GenerateKeyPair() ([]byte, []byte) {
	return secp256k1.GenerateKeyPair()
}

func PublicKeyFromPrivateKey(secret []byte) []byte {
	return secp256k1.PubkeyFromSeckey(secret)
}

func VerifySignature(msg []byte, signature []byte, publicKey []byte) bool {
	ret := secp256k1.VerifySignature(msg, signature, publicKey)
	if ret == 0 {
		if (signature[32] >> 7) == 1 {
			log.Println("VALID SIG but fails malleability")
		}
		if signature[64] >= 4 {
			log.Println("RECOVER BYTE INVALID")
		}
	}
	return ret != 0
}

func SHA3Sum256(m []byte) []byte {
	d := sha3.Sum256(m)
	return d[:]
}

func SHASum256(m []byte) []byte {
	d := sha256.Sum256(m)
	return d[:]
}
