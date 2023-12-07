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

type Signed[T any] struct {
	Message   T
	Signature []byte
	signer    module.PeerID
}

func (s *Signed[T]) MarshalBinary() (data []byte, err error) {
	v := struct {
		Message   T
		Signature []byte
	}{
		Message:   s.Message,
		Signature: s.Signature,
	}
	return codec.BC.MarshalToBytes(v)
}

func (s *Signed[T]) UnmarshalBinary(data []byte) error {
	v := struct {
		Message   T
		Signature []byte
	}{}
	if _, err := codec.BC.UnmarshalFromBytes(data, &v); err != nil {
		return err
	}
	s.Message = v.Message
	s.Signature = v.Signature
	return nil
}

func (s *Signed[T]) MessageHash() ([]byte, error) {
	b, err := codec.BC.MarshalToBytes(s.Message)
	if err != nil {
		return nil, err
	}
	return crypto.SHA3Sum256(b), nil
}

func (s *Signed[T]) Recover() (module.PeerID, error) {
	h, err := s.MessageHash()
	if err != nil {
		return nil, err
	}
	sig, err := crypto.ParseSignature(s.Signature)
	if err != nil {
		return nil, err
	}
	pubk, err := sig.RecoverPublicKey(h)
	if err != nil {
		return nil, err
	}
	signer := NewPeerIDFromAddress(common.NewAccountAddressFromPublicKey(pubk))
	s.signer = signer
	return signer, nil
}

func (s *Signed[T]) Sign(w module.Wallet) error {
	h, err := s.MessageHash()
	if err != nil {
		return err
	}
	signature, err := w.Sign(h)
	if err != nil {
		return err
	}
	s.Signature = signature
	s.signer = NewPeerIDFromAddress(w.Address())
	return nil
}

func (s *Signed[T]) Signer() module.PeerID {
	return s.signer
}

func (s *Signed[T]) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "Signed{Message:%v,Signature:%s,Signer:%v}",
			s.Message, hex.EncodeToString(s.Signature), s.signer)
	case 's':
		fmt.Fprintf(f, "{Message:%v,Signature:%s,Signer:%v}",
			s.Message, hex.EncodeToString(s.Signature), s.signer)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

func NewSignedFromBytes[T any](b []byte) (*Signed[T], module.PeerID, error) {
	s := &Signed[T]{}
	if err := s.UnmarshalBinary(b); err != nil {
		return nil, nil, errors.Wrapf(err, "fail to UnmarshalBinary err:%v", err)
	}
	id, err := s.Recover()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to Recover err:%v", err)
	}
	return s, id, nil
}
