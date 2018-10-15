package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/crypto/btcec"
	"golang.org/x/crypto/sha3"
)

// TODO Review some codes from ethereum for license issue.
var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// GenerateKey generates a public and private key pair.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(btcec.S256(), rand.Reader)
}

// PubKeyToAddrBytes extracts bytes of the public key for the account address
func PubKeyToAddrBytes(pub ecdsa.PublicKey) []byte {
	pubBytes := GetPubKeyBytes(&pub)
	if pubBytes == nil || len(pubBytes) != 65 {
		return nil
	}
	digest := sha3.Sum256(pubBytes[1:])
	return digest[len(digest)-20:]
}

// GetPubKeyBytes returns the byte format of the public key
func GetPubKeyBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(btcec.S256(), pub.X, pub.Y)
}

// Hash256 returns the 256-byte digest of the data.
// TODO Consider how to provide API for SHA256 hashing partially used in current impl.
func Hash256(raw []byte) (hash [32]byte) {
	return sha3.Sum256(raw)
}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
// TODO Consider malleability
func Sign(hash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	if prv.Curve != btcec.S256() {
		return nil, fmt.Errorf("private key curve is not secp256k1")
	}
	sig, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(prv), hash, false)
	if err != nil {
		return nil, err
	}
	// Convert to signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

// VerifySig checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func VerifySig(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}
	sig := &btcec.Signature{R: new(big.Int).SetBytes(signature[:32]), S: new(big.Int).SetBytes(signature[32:])}
	key, err := btcec.ParsePubKey(pubkey, btcec.S256())
	if err != nil {
		return false
	}

	// Reject malleable signatures. libsecp256k1 does this check but btcec doesn't.
	// if sig.S.Cmp(secp256k1halfN) > 0 {
	// return false
	// }
	return sig.Verify(hash, key)
}

// SigToPubKey returns the public key that created the given signature.
func SigToPubKey(hash, sig []byte) (*ecdsa.PublicKey, error) {
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(pub), err
}
