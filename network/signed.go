package network

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type Signed[T any] struct {
	Message   T
	Signature []byte
}

func (s *Signed[T]) MarshalBinary() (data []byte, err error) {
	return codec.BC.MarshalToBytes(s)
}

func (s *Signed[T]) UnmarshalBinary(data []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(data, s)
	return err
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
	return NewPeerIDFromAddress(common.NewAccountAddressFromPublicKey(pubk)), nil
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
	return nil
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
