package state

import (
	"bytes"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"

	"github.com/icon-project/goloop/module"
)

func TestValidatorFromAddress(t *testing.T) {
	type args struct {
		a module.Address
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "EOA1",
			args: args{
				common.MustNewAddressFromString("hx4567db98764567db98764567db98764567db9876"),
			},
			wantErr: false,
		},
		{
			name: "EOA2Error",
			args: args{
				nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatorFromAddress(tt.args.a)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Fail to make address err=")
				}
				return
			}
			if !bytes.Equal(got.Address().Bytes(), tt.args.a.Bytes()) {
				t.Errorf("Invalid Validator.Address exp=%x ret=%x",
					tt.args.a.Bytes(),
					got.Address().Bytes())
			}
			if pk := got.PublicKey(); pk != nil {
				t.Errorf("Invalid Validator.PublicKey exp=nil ret=%v", pk)
			}
		})
	}
}

func TestValidatorSerializeWithPubKey(t *testing.T) {
	_, pk1 := crypto.GenerateKeyPair()
	pk1Bytes := pk1.SerializeUncompressed()
	v1, err := ValidatorFromPublicKey(pk1Bytes)
	if err != nil {
		t.Errorf("Fail to make validator with publickey=[%x]", pk1Bytes)
		return
	}
	t.Logf("Test public key: %x\n", pk1Bytes)

	addr1 := common.NewAccountAddressFromPublicKey(pk1)
	if addr1 == nil {
		t.Errorf("Address from publickey is nil")
		return
	}
	t.Logf("Test address: %s\n", addr1.String())

	addr2 := v1.Address()
	if !addr1.Equal(addr2) {
		t.Errorf("Different address exp=%v ret=%v", addr1, addr2)
		return
	}

	pk2Bytes := v1.PublicKey()
	pk2, err := crypto.ParsePublicKey(pk2Bytes)
	if err != nil {
		t.Errorf("Returned public key not parsible:%x", pk2Bytes)
		return
	}
	if !pk2.Equal(pk1) {
		t.Errorf("Returned public key isn't same")
		return
	}

	v1Bytes := v1.Bytes()

	var v3 *validator
	if _, err := codec.BC.UnmarshalFromBytes(v1Bytes, &v3); err != nil {
		t.Errorf("Fail to unmarshal bytes")
		return
	}

	addr3 := v3.Address()
	if !addr1.Equal(addr3) {
		t.Errorf("Different unmarshalled address exp=%v ret=%v", addr1, addr3)
		return
	}

	pk3Bytes := v3.PublicKey()
	pk3, err := crypto.ParsePublicKey(pk3Bytes)
	if err != nil {
		t.Errorf("Unmarshalled public key not parsible:%x", pk3Bytes)
		return
	}
	if !pk3.Equal(pk1) {
		t.Errorf("Unmarshalled public key isn't same")
		return
	}
}

func TestValidatorSerializeWithAddr(t *testing.T) {
	addr := common.MustNewAddressFromString("hx0000000000000000000000000000000000000000")
	v, err := ValidatorFromAddress(addr)
	if err != nil {
		t.Errorf("Fail to make Validator from addr=%v err=%+v", addr, err)
		return
	}

	b, err := codec.BC.MarshalToBytes(v)
	if err != nil {
		t.Errorf("Fail to marshal Validator from validator=%v err=%+v", v, err)
		return
	}

	t.Logf("Serialized:[%x]", b)

	var v2 *validator
	if _, err := codec.BC.UnmarshalFromBytes(b, &v2); err != nil {
		t.Errorf("Fail to unmarshal Validator from bytes=%x", b)
		return
	}

	if v2 == nil || v2.Address() == nil {
		t.Logf("Fail to unmarshal validator\n")
		return
	}

	if !bytes.Equal(v2.Address().Bytes(), addr.Bytes()) {
		t.Errorf("Unmarshalled address[%x] is different from [%x]", v2.Address().Bytes(), addr.Bytes())
		return
	}
}
