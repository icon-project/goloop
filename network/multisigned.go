package network

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type MultiSigned[T any] struct {
	Message    T
	Signatures [][]byte
}

func (s *MultiSigned[T]) MarshalBinary() (data []byte, err error) {
	return codec.BC.MarshalToBytes(s)
}

func (s *MultiSigned[T]) UnmarshalBinary(data []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(data, s)
	return err
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
		singers []module.PeerID
	)
	for _, signature := range s.Signatures {
		if sig, err = crypto.ParseSignature(signature); err != nil {
			return nil, err
		}
		if pubk, err = sig.RecoverPublicKey(h); err != nil {
			return nil, err
		}
		singers = append(singers, NewPeerIDFromAddress(common.NewAccountAddressFromPublicKey(pubk)))
	}
	return singers, nil
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
	return nil
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
