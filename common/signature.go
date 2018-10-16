package common

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	"github.com/icon-project/goloop/common/crypto"
)

type Signature []byte

func (sig *Signature) MarshalJSON() ([]byte, error) {
	s := base64.StdEncoding.EncodeToString(*sig)
	return []byte(s), nil
}

func (sig *Signature) UnmarshalJSON(s []byte) error {
	var str string
	err := json.Unmarshal(s, &str)
	if err != nil {
		return err
	}
	if b, err := base64.StdEncoding.DecodeString(str); err == nil {
		*sig = b
		return nil
	} else {
		return err
	}
}

func (sig Signature) String() string {
	return "0x" + hex.EncodeToString(sig)
}

func (sig *Signature) RecoverPublicKeyWithHash(h []byte) ([]byte, error) {
	return crypto.RecoverPublicKey(h, *sig)
}

func (sig *Signature) RecoverAddressWithHash(h []byte) (string, error) {
	p, err := sig.RecoverPublicKeyWithHash(h)
	if err != nil {
		return "", err
	}
	return crypto.PublicKeyToUserAddr(p)
}
