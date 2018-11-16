package common

import (
	"github.com/icon-project/goloop/common/crypto"
)

type Wallet struct {
	PbKey   *crypto.PublicKey
	PrKey   *crypto.PrivateKey
	Address *Address
}

func CreateWallet(walletNum int) []Wallet {
	ws := make([]Wallet, walletNum)
	for i := 0; i < walletNum; i++ {
		w := Wallet{}
		w.PrKey, w.PbKey = crypto.GenerateKeyPair()
		w.Address = NewAccountAddressFromPublicKey(w.PbKey)
		ws[i] = w
	}
	return ws
}

//
func SignTransaction(tx interface{}) (signature []byte) {
	signature = nil
	return
}

func HashTransaction(tx interface{}) (hash []byte) {
	hash = nil
	return
}
