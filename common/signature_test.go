package common

import (
	"bytes"
	"testing"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
)

func TestSignatureCoding(t *testing.T) {
	obs := []byte("01234567890123456789012345678901234567890123456789012345678901234")
	var sig Signature
	var err error
	sig.Signature, err = crypto.ParseSignature(obs)
	if err != nil {
		t.Fail()
	}
	sigBS, err := codec.MarshalToBytes(&sig)
	if err != nil {
		t.Fail()
	}
	var sig2 Signature
	_, err = codec.UnmarshalFromBytes(sigBS, &sig2)
	if err != nil {
		t.Fail()
	}
	rsv, err := sig2.Signature.SerializeRSV()
	if err != nil {
		t.Fail()
	}
	if !bytes.Equal(obs, rsv) {
		t.Fail()
	}
}
