package crypto

import (
	"crypto/sha256"
	"errors"
	"math/big"
	"reflect"

	"github.com/haltingstate/secp256k1-go"
	"golang.org/x/crypto/sha3"
)

//////////////////////////////////////////////////////////////////////
// privatekey.go
//////////////////////////////////////////////////////////////////////
const (
	PrivateKeyLen = 32
)

// PrivateKey is a type representing a private key.
// TODO private key always includes public key? or create KeyPair struct
// for both private key and public key
type PrivateKey struct {
	bytes []byte // 32-byte
}

// TODO add 'func ToECDSA() ecdsa.PrivateKey' if needed

//////////////////////////////////////////////////////////////////////
// publickey.go
//////////////////////////////////////////////////////////////////////
const (
	PublicKeyLenCompressed   = 33
	PublicKeyLenUncompressed = 65

	publicKeyCompressed   byte = 0x2 // y_bit + x coord
	publicKeyUncompressed byte = 0x4 // x coord + y coord
)

// PublicKey is a type representing a public key, which can be serialized to
// or deserialized from compressed or uncompressed formats.
type PublicKey struct {
	bytes []byte // 33-byte compressed format to use halting state library efficiently
}

// ParsePublicKey parses the public key into a PublicKey instance. It supports
// uncompressed and compressed formats.
// NOTE: For the efficiency, it may use the slice directly. So don't change any
// internal value of the public key
func ParsePublicKey(pubKey []byte) (*PublicKey, error) {
	new(big.Int).Bytes()
	switch len(pubKey) {
	case 0:
		return nil, errors.New("public key bytes are empty")
	case PublicKeyLenCompressed:
		return &PublicKey{pubKey}, nil
	case PublicKeyLenUncompressed:
		return &PublicKey{uncompToCompPublicKey(pubKey)}, nil
	default:
		return nil, errors.New("wrong format")
	}
}

// uncompToCompPublicKey changes the uncompressed formatted public key to
// the compressed formatted. It assumes the uncompressed key is valid.
func uncompToCompPublicKey(uncomp []byte) (comp []byte) {
	comp = make([]byte, PublicKeyLenCompressed)
	// skip to check the validity of uncompressed key
	format := publicKeyCompressed
	if uncomp[64]&0x1 == 0x1 {
		format |= 0x1
	}
	comp[0] = format
	copy(comp[1:], uncomp[1:33])
	return
}

// SerializeCompressed serializes the public key in a 33-byte compressed format.
// For the efficiency, it returns the slice internally used, so don't change
// any internal value in the returned slice.
func (key *PublicKey) SerializeCompressed() []byte {
	return key.bytes
}

// SerializeUncompressed serializes the public key in a 65-byte uncompressed format.
func (key *PublicKey) SerializeUncompressed() []byte {
	return secp256k1.UncompressPubkey(key.bytes)
}

// IsEqual returns true if the given public key is same as this instance
// semantically
func (key *PublicKey) IsEqual(pubKey *PublicKey) bool {
	return reflect.DeepEqual(key.bytes, pubKey.bytes)
}

// TODO add 'func ToECDSA() ecdsa.PublicKey' if needed

//////////////////////////////////////////////////////////////////////
// ecc.go
//////////////////////////////////////////////////////////////////////
const (
	// SignatureLen is the bytes length of signature
	SignatureLenRawWithV = 65
	SignatureLenRaw      = 64
	invalidV             = 0xff
	// HashLen is the bytes length of hash for signature
	HashLen = 32
)

// GenerateKeyPair generates a private and public key pair.
func GenerateKeyPair() (privKey *PrivateKey, pubKey *PublicKey) {
	pub, priv := secp256k1.GenerateKeyPair()
	privKey = &PrivateKey{priv}
	pubKey, _ = ParsePublicKey(pub)
	return
}

// Signature is a type representing an ECDSA signature with or without V.
type Signature struct {
	bytes []byte // 65 bytes of [R|S|V]
	hasV  bool
}

// NewSignature calculates an ECDSA signature including V, which is 0 or 1.
func NewSignature(hash []byte, privKey *PrivateKey) (*Signature, error) {
	if len(hash) == 0 || len(hash) > HashLen || privKey == nil {
		return nil, errors.New("Invalid arguments")
	}
	return ParseSignature(secp256k1.Sign(hash, privKey.bytes))
}

// ParseSignature parses a signature from the raw byte array of 64([R|S]) or
// 65([R|S|V]) bytes long. If a source signature is formatted as [V|R|S],
// call ParseSignatureVRS instead.
// NOTE: For the efficiency, it may use the slice directly. So don't change any
// internal value of the signature.
func ParseSignature(sig []byte) (*Signature, error) {
	var s Signature
	switch len(sig) {
	case 0:
		return nil, errors.New("sigature bytes are empty")
	case SignatureLenRawWithV:
		s.bytes = sig
		s.hasV = true
	case SignatureLenRaw:
		s.bytes = append(s.bytes, sig...)
		s.bytes[64] = 0x00 // no meaning
		s.hasV = false
	default:
		return nil, errors.New("wrong raw signature format")
	}
	return &s, nil
}

// ParseSignatureVRS parses a signature from the [V|R|S] formatted signature.
// If the format of a source signature is different,
// call ParseSignature instead.
func ParseSignatureVRS(sig []byte) (*Signature, error) {
	if len(sig) != 65 {
		return nil, errors.New("wrong raw signature format")
	}

	var s Signature
	s.bytes = append(s.bytes, sig[1:33]...)
	s.bytes = append(s.bytes, sig[33:]...)
	s.bytes[64] = sig[0]
	return &s, nil
}

// HasV returns whether the signature has V value.
func (sig *Signature) HasV() bool {
	return sig.hasV
}

// SerializeRS returns the 64-byte data formatted as [R|S] from the signature.
// For the efficiency, it returns the slice internally used, so don't change
// any internal value in the returned slice.
func (sig *Signature) SerializeRS() ([]byte, error) {
	if len(sig.bytes) < 64 {
		return nil, errors.New("not a valid signature")
	}
	return sig.bytes[:64], nil
}

// SerializeVRS returns the 65-byte data formatted as [V|R|S] from the signature.
// Make sure that it has a valid V value. If it doesn't have V value, then it
// will throw error.
// For the efficiency, it returns the slice internally used, so don't change
// any internal value in the returned slice.
func (sig *Signature) SerializeVRS() ([]byte, error) {
	if !sig.HasV() {
		return nil, errors.New("no V value")
	}

	s := make([]byte, SignatureLenRawWithV)
	s[0] = sig.bytes[64]
	copy(s[1:33], sig.bytes[:32])
	copy(s[33:], sig.bytes[32:64])
	return s, nil
}

// SerializeRSV returns the 65-byte data formatted as [R|S|V] from the signature.
// Make sure that it has a valid V value. If it doesn't have V value, then it
// will throw error.
// For the efficiency, it returns the slice internally used, so don't change
// any internal value in the returned slice.
func (sig *Signature) SerializeRSV() ([]byte, error) {
	if !sig.HasV() {
		return nil, errors.New("no V value")
	}

	return sig.bytes, nil
}

// RecoverPublicKey recovers a public key from the hash of message and its signature.
func (sig *Signature) RecoverPublicKey(hash []byte) (*PublicKey, error) {
	if !sig.HasV() {
		return nil, errors.New("signature has no V value")
	}
	if len(hash) == 0 || len(hash) > HashLen {
		return nil, errors.New("message hash is illegal")
	}
	s, err := sig.SerializeRSV()
	if err != nil {
		return nil, err
	}
	return ParsePublicKey(secp256k1.RecoverPubkey(hash[:], s))
}

// Verify verifies the signature of hash using the public key.
func (sig *Signature) Verify(msg []byte, pubKey *PublicKey) bool {
	if len(msg) == 0 || len(msg) > HashLen || pubKey == nil {
		return false
	}
	s, err := sig.SerializeRSV()
	if err != nil {
		return false
	}
	ret := secp256k1.VerifySignature(msg, s, pubKey.bytes)
	// TODO disable malleability check?
	// if ret == 0 {
	// 	// TODO why?
	// 	if (s[32] >> 7) == 1 {
	// 		log.Println("VALID SIG but fails malleability")
	// 	}
	// 	// TODO why?
	// 	if s[64] >= 4 {
	// 		log.Println("RECOVER BYTE INVALID")
	// 	}
	// }
	return ret != 0
}

//////////////////////////////////////////////////////////////////////
// hash.go
//////////////////////////////////////////////////////////////////////

// SHA3Sum256 returns the SHA3-256 digest of the data
func SHA3Sum256(m []byte) []byte {
	d := sha3.Sum256(m)
	return d[:]
}

// SHASum256 returns the SHA256 digest of the data
func SHASum256(m []byte) []byte {
	d := sha256.Sum256(m)
	return d[:]
}
