package crypto

import (
	"encoding/hex"
	"errors"

	"github.com/icon-project/goloop/common/codec"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const (
	// SignatureLenRawWithV is the bytes length of signature including V value
	SignatureLenRawWithV = 65
	// SignatureLenRaw is the bytes length of signature not including V value
	SignatureLenRaw = 64
	// HashLen is the bytes length of hash for signature
	HashLen = 32
)

// Signature is a type representing an ECDSA signature with or without V.
type Signature struct {
	bytes []byte // 65 bytes of [V|R|S] if it has V otherwise its [R|S]
}

const compactSigMagicOffset = 27

func recoverFlagToECDSA(flag byte) byte {
	return flag + compactSigMagicOffset
}

func recoverFlagToCompatible(flag byte) byte {
	return flag - compactSigMagicOffset
}

// NewSignature calculates an ECDSA signature including V, which is 0 or 1.
func NewSignature(hash []byte, privKey *PrivateKey) (*Signature, error) {
	if len(hash) == 0 || len(hash) > HashLen || privKey == nil {
		return nil, errors.New("Invalid arguments")
	}
	return &Signature{
		bytes: ecdsa.SignCompact(privKey.real, hash, false),
	}, nil
}

// ParseSignature parses a signature from the raw byte array of 64([R|S]) or
// 65([R|S|V]) bytes long. If a source signature is formatted as [V|R|S],
// call ParseSignatureVRS instead.
// NOTE: For the efficiency, it may use the slice directly. So don't change any
// internal value of the signature.
func ParseSignature(sig []byte) (*Signature, error) {
	if data, err := parseSignature(sig); err != nil {
		return nil, err
	} else {
		return &Signature{
			bytes: data,
		}, nil
	}
}

func parseSignature(sig []byte) ([]byte, error) {
	switch len(sig) {
	case 0:
		return nil, errors.New("signature bytes are empty")
	case SignatureLenRawWithV:
		vrs := make([]byte, SignatureLenRawWithV)
		copy(vrs[1:], sig)
		vrs[0] = recoverFlagToECDSA(sig[SignatureLenRaw])
		return vrs, nil
	case SignatureLenRaw:
		rs := make([]byte, SignatureLenRaw)
		copy(rs, sig)
		return rs, nil
	default:
		return nil, errors.New("wrong raw signature format")
	}
}

// ParseSignatureVRS parses a signature from the [V|R|S] formatted signature.
// If the format of a source signature is different,
// call ParseSignature instead.
func ParseSignatureVRS(sig []byte) (*Signature, error) {
	if len(sig) != SignatureLenRawWithV {
		return nil, errors.New("wrong raw signature format")
	}
	var s Signature
	s.bytes = append(s.bytes, sig...)
	s.bytes[0] = recoverFlagToECDSA(s.bytes[0])
	return &s, nil
}

// HasV returns whether the signature has V value.
func (sig *Signature) HasV() bool {
	return len(sig.bytes) == SignatureLenRawWithV
}

// SerializeRS returns the 64-byte data formatted as [R|S] from the signature.
// For the efficiency, it returns the slice internally used, so don't change
// any internal value in the returned slice.
func (sig *Signature) SerializeRS() ([]byte, error) {
	if sz := len(sig.bytes); sz == SignatureLenRaw {
		return sig.bytes, nil
	} else if sz == SignatureLenRawWithV {
		return sig.bytes[1:], nil
	} else {
		return nil, errors.New("not a valid signature")
	}
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
	copy(s, sig.bytes)
	s[0] = recoverFlagToCompatible(s[0])
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

	s := make([]byte, SignatureLenRawWithV)
	copy(s[:SignatureLenRaw], sig.bytes[1:])
	s[SignatureLenRaw] = recoverFlagToCompatible(sig.bytes[0])
	return s, nil
}

// RecoverPublicKey recovers a public key from the hash of message and its signature.
func (sig *Signature) RecoverPublicKey(hash []byte) (*PublicKey, error) {
	if !sig.HasV() {
		return nil, errors.New("signature has no V value")
	}
	if len(hash) == 0 || len(hash) > HashLen {
		return nil, errors.New("message hash is illegal")
	}
	pk, _, err := ecdsa.RecoverCompact(sig.bytes, hash)
	if err != nil {
		return nil, err
	}
	return &PublicKey{real: pk}, err
}

// Verify verifies the signature of hash using the public key.
func (sig *Signature) Verify(msg []byte, pubKey *PublicKey) bool {
	if len(msg) == 0 || len(msg) > HashLen || pubKey == nil {
		return false
	}
	r := new(secp256k1.ModNScalar)
	s := new(secp256k1.ModNScalar)
	r.SetByteSlice(sig.bytes[1:33])
	s.SetByteSlice(sig.bytes[33:])
	sigReal := ecdsa.NewSignature(r, s)
	return sigReal.Verify(msg, pubKey.real)
}

// String returns the string representation.
func (sig *Signature) String() string {
	if sig == nil || len(sig.bytes) == 0 {
		return "[empty]"
	}
	if len(sig.bytes) == SignatureLenRawWithV {
		return "0x" + hex.EncodeToString(sig.bytes[1:]) +
			hex.EncodeToString([]byte{recoverFlagToCompatible(sig.bytes[0])})
	}
	return "0x" + hex.EncodeToString(sig.bytes) + "[no V]"
}

func (sig *Signature) RLPEncodeSelf(e codec.Encoder) error {
	bs, err := sig.SerializeRSV()
	if err != nil {
		return err
	}
	return e.Encode(bs)
}

func (sig *Signature) RLPDecodeSelf(d codec.Decoder) error {
	var bs []byte
	err := d.Decode(&bs)
	if err != nil {
		return err
	}
	if bs == nil {
		return codec.ErrNilValue
	}
	if data, err := parseSignature(bs); err != nil {
		return err
	} else {
		sig.bytes = data
		return nil
	}
}
