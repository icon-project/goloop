package network

import (
	"encoding/hex"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type MultiSigned[T any] struct {
	Message    T
	Signatures [][]byte
	signers    []module.PeerID
}

func (s *MultiSigned[T]) MarshalBinary() (data []byte, err error) {
	v := struct {
		Message    T
		Signatures [][]byte
	}{
		Message:    s.Message,
		Signatures: s.Signatures,
	}
	return codec.BC.MarshalToBytes(v)
}

func (s *MultiSigned[T]) UnmarshalBinary(data []byte) error {
	v := struct {
		Message    T
		Signatures [][]byte
	}{}
	if _, err := codec.BC.UnmarshalFromBytes(data, &v); err != nil {
		return err
	}
	s.Message = v.Message
	s.Signatures = v.Signatures
	return nil
}

func (s *MultiSigned[T]) MessageHash() ([]byte, error) {
	b, err := codec.BC.MarshalToBytes(s.Message)
	if err != nil {
		return nil, err
	}
	return crypto.SHA3Sum256(b), nil
}

func (s *MultiSigned[T]) Recover() ([]module.PeerID, error) {
	h, err := s.MessageHash()
	if err != nil {
		return nil, err
	}
	var (
		sig     *crypto.Signature
		pubk    *crypto.PublicKey
		signers []module.PeerID
	)
	for _, signature := range s.Signatures {
		if sig, err = crypto.ParseSignature(signature); err != nil {
			return nil, err
		}
		if pubk, err = sig.RecoverPublicKey(h); err != nil {
			return nil, err
		}
		signers = append(signers, NewPeerIDFromAddress(common.NewAccountAddressFromPublicKey(pubk)))
	}
	s.signers = signers
	return signers, nil
}

func (s *MultiSigned[T]) Sign(w module.Wallet) error {
	h, err := s.MessageHash()
	if err != nil {
		return err
	}
	signature, err := w.Sign(h)
	if err != nil {
		return err
	}
	s.Signatures = append(s.Signatures, signature)
	s.signers = append(s.signers, NewPeerIDFromAddress(w.Address()))
	return nil
}

func (s *MultiSigned[T]) Signers() []module.PeerID {
	return s.signers[:]
}

func (s MultiSigned[T]) Format(f fmt.State, verb rune) {
	var signatures []string
	for _, signature := range s.Signatures {
		signatures = append(signatures, hex.EncodeToString(signature))
	}
	switch verb {
	case 'v':
		fmt.Fprintf(f, "MultiSigned{Message:%v,Signatures:%v,Signers:%v}",
			s.Message, signatures, s.signers)
	case 's':
		fmt.Fprintf(f, "Signed{Message:%v,Signatures:%v,Signers:%v}",
			s.Message, signatures, s.signers)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

func NewMultiSignedFromBytes[T any](b []byte) (*MultiSigned[T], []module.PeerID, error) {
	s := &MultiSigned[T]{}
	if err := s.UnmarshalBinary(b); err != nil {
		return nil, nil, errors.Wrapf(err, "fail to UnmarshalBinary err:%v", err)
	}
	ids, err := s.Recover()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to Recover err:%v", err)
	}
	return s, ids, nil
}
