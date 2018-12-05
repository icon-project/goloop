package common

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type softwareWallet struct {
	skey *crypto.PrivateKey
	pkey *crypto.PublicKey
}

func (w *softwareWallet) Address() module.Address {
	return NewAccountAddressFromPublicKey(w.pkey)
}

func (w *softwareWallet) Sign(data []byte) ([]byte, error) {
	sig, err := crypto.NewSignature(data, w.skey)
	if err != nil {
		return nil, err
	}
	return sig.SerializeRSV()
}

func (w *softwareWallet) PublicKey() []byte {
	return w.pkey.SerializeCompressed()
}

func NewWallet() module.Wallet {
	sk, pk := crypto.GenerateKeyPair()
	return &softwareWallet{
		skey: sk,
		pkey: pk,
	}
}

func NewWalletFromPrivateKey(sk *crypto.PrivateKey) (module.Wallet, error) {
	pk := sk.PublicKey()
	return &softwareWallet{
		skey: sk,
		pkey: pk,
	}, nil
}
