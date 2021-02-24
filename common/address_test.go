package common

import (
	"bytes"
	"encoding/hex"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

var (
	addr1_str1  = "hx1234567890abcdef1234"
	addr1_str2  = "cx1234567890abcdef1234"
	addr1_str3  = "0x1234567890abcdef1234"
	addr1_str4  = "1234567890abcdef1234"
	addr1_bytes = []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34}

	addr2_str1  = "hx00011234567890abcdef"
	addr2_str2  = "cx00011234567890abcdef"
	addr2_str3  = "0x00011234567890abcdef"
	addr2_str4  = "00011234567890abcdef"
	addr2_str5  = "11234567890abcdef"
	addr2_bytes = []byte{0x00, 0x01, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}
)

func TestNewAddressFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want *Address
	}{
		{
			"Account1-1",
			args{s: addr1_str1},
			NewAccountAddress(addr1_bytes),
		},
		{
			"Contract1-2",
			args{s: addr1_str2},
			NewContractAddress(addr1_bytes),
		},
		{
			"Account1-3",
			args{s: addr1_str3},
			NewAccountAddress(addr1_bytes),
		},
		{
			"Account1-4",
			args{s: addr1_str4},
			NewAccountAddress(addr1_bytes),
		},
		{
			"Account2-1",
			args{s: addr2_str1},
			NewAccountAddress(addr2_bytes),
		},
		{
			"Contract2-2",
			args{s: addr2_str2},
			NewContractAddress(addr2_bytes),
		},
		{
			"Account2-3",
			args{s: addr2_str3},
			NewAccountAddress(addr2_bytes),
		},
		{
			"Account2-4",
			args{s: addr2_str4},
			NewAccountAddress(addr2_bytes),
		},
		{
			"Account2-5",
			args{s: addr2_str5},
			NewAccountAddress(addr2_bytes),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustNewAddressFromString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAddressFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddressEncodingDecoding(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Case1",
			args{"hx0000000000000000000000000000000000000000"},
			"000000000000000000000000000000000000000000",
		},
		{
			"Case2",
			args{"cx1908581ed9f09c45810405897123badefcbfefa0"},
			"011908581ed9f09c45810405897123badefcbfefa0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := hex.DecodeString(tt.want)
			if err != nil {
				log.Printf("Test(%s) want=%s illegal", tt.name, tt.want)
				return
			}
			var b2 []byte
			b2, err = codec.MarshalToBytes(want)
			if err != nil {
				log.Printf("Test(%s) fail to marshal bytes err=%+v",
					tt.name, err)
				return
			}

			a := MustNewAddressFromString(tt.args.s)
			b, err := codec.MarshalToBytes(a)
			if err != nil {
				t.Error(err)
				return
			}
			log.Printf("Encoded:[%x]", b)
			log.Printf("Expect :[%x]", b2)
			if !bytes.Equal(b, b2) {
				t.Errorf("Encoded bytes are different exp=[%x] result=[%x]", b2, b)
			}

			var a2 Address
			_, err = codec.UnmarshalFromBytes(b, &a2)
			if err != nil {
				t.Error(err)
				return
			}

			log.Printf("Recovered:[%v]", &a2)

			if a2.String() != tt.args.s {
				t.Errorf("Fail to recover expected=%s recovered=%s",
					tt.args.s, a2.String())
			}
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	type args struct {
		a2 module.Address
	}
	tests := []struct {
		name string
		a    *Address
		args args
		want bool
	}{
		{
			name: "NilAndNil",
			a:    nil,
			args: args{nil},
			want: true,
		},
		{
			name: "NilAndNilPtr",
			a:    nil,
			args: args{(*Address)(nil)},
			want: true,
		},
		{
			name: "NilvsNonNil",
			a:    nil,
			args: args{MustNewAddressFromString("hx8888888888888888888888888888888888888888")},
			want: false,
		},
		{
			name: "NonNilvsNil",
			a:    MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{nil},
			want: false,
		},
		{
			name: "NonNilvsNilPtr",
			a:    MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{(*Address)(nil)},
			want: false,
		},
		{
			name: "Same1",
			a:    MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{MustNewAddressFromString("hx8888888888888888888888888888888888888888")},
			want: true,
		},
		{
			name: "Same2",
			a:    MustNewAddressFromString("cx8888888888888888888888888888888888888888"),
			args: args{MustNewAddressFromString("cx8888888888888888888888888888888888888888")},
			want: true,
		},
		{
			name: "Diff1",
			a:    MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{MustNewAddressFromString("cx8888888888888888888888888888888888888888")},
			want: false,
		},
		{
			name: "Diff2",
			a:    MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{MustNewAddressFromString("hx9888888888888888888888888888888888888888")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.args.a2); got != tt.want {
				t.Errorf("Address.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_SetBytes(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		a       Address
		args    args
		wantErr bool
	}{
		{
			name:    "Nil",
			a:       Address{},
			args:    args{nil},
			wantErr: true,
		},
		{
			name:    "Empty",
			a:       Address{},
			args:    args{[]byte{}},
			wantErr: true,
		},
		{
			name:    "ContractNoID",
			a:       Address{},
			args:    args{[]byte{1}},
			wantErr: true,
		},
		{
			name:    "EOANoID",
			a:       Address{},
			args:    args{[]byte{0}},
			wantErr: true,
		},
		{
			name:    "EOA21Bytes",
			a:       Address{},
			args:    args{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			wantErr: false,
		},
		{
			name:    "Contract21Bytes",
			a:       Address{},
			args:    args{[]byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			wantErr: false,
		},
		{
			name:    "InvalidType21Bytes",
			a:       Address{},
			args:    args{[]byte{3, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			wantErr: true,
		},
		{
			name:    "EOA22Bytes",
			a:       Address{},
			args:    args{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1}},
			wantErr: true,
		},
		{
			name:    "EOA20Bytes",
			a:       Address{},
			args:    args{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.SetBytes(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("SetBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddress_SetTypeAndID(t *testing.T) {
	t.Run("SetSmallID", func(t *testing.T) {
		addr1 := new(Address)
		addr1.SetTypeAndID(false, []byte{0x12, 0x34, 0x56})
		assert.Equal(t,
			"hx0000000000000000000000000000000000123456",
			addr1.String())
	})

	t.Run("SetAgainWithSmallerID", func(t *testing.T) {
		addr2 := new(Address)
		addr2.SetTypeAndID(false, []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12,
		})

		addr1 := new(Address)
		addr1.SetTypeAndID(false, []byte{0x12, 0x34, 0x56})
		addr1.SetTypeAndID(false, []byte{0x12})

		assert.Equal(t, addr2, addr1)
	})

	t.Run("SetNilID", func(t *testing.T) {
		addr2 := new(Address)
		addr2.SetTypeAndID(false, nil)
	})
}

func TestAddress_Set(t *testing.T) {
	addr0 := MustNewAddressFromString("hxce6e688a539449c3f9f5c5990749c135bf0ee0e3")

	t.Run("SetWithSelf", func(t *testing.T) {
		addr1 := MustNewAddressFromString("hxce6e688a539449c3f9f5c5990749c135bf0ee0e3")
		addr1.Set(addr1)
		assert.Equal(t, addr0, addr1)
	})

	t.Run("SetOtherOnEmpty", func(t *testing.T) {
		addr2 := new(Address)
		addr2.Set(addr0)
		assert.Equal(t, addr0, addr2)
	})

	t.Run("SetNil", func(t *testing.T) {
		addr2 := new(Address)
		addr2.Set(nil)
		assert.Equal(t, new(Address), addr2)
	})

	t.Run("SetOther", func(t *testing.T) {
		addr1 := MustNewAddressFromString("hxfa6341b183b48fd460b9a42884db7987a46ea92f")
		addr1.Set(addr0)
		assert.Equal(t, addr0, addr1)
	})
}

func TestAddress_ToString(t *testing.T) {
	type arg struct {
		contract bool
		id       []byte
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{
			name: "Treasury",
			arg:  arg{false, []byte{0x01}},
			want: "hx0000000000000000000000000000000000000001",
		},
		{
			name: "ChainSCORE1",
			arg:  arg{true, []byte{}},
			want: "cx0000000000000000000000000000000000000000",
		},
		{
			name: "ChainSCORE2",
			arg:  arg{true, []byte{0}},
			want: "cx0000000000000000000000000000000000000000",
		},
		{
			name: "Governance",
			arg:  arg{true, []byte{0x01}},
			want: "cx0000000000000000000000000000000000000001",
		},
		{
			name: "EOA",
			arg: arg{false, []byte{
				0xfa, 0x63, 0x41, 0xb1, 0x83, 0xb4, 0x8f, 0xd4, 0x60, 0xb9,
				0xa4, 0x28, 0x84, 0xdb, 0x79, 0x87, 0xa4, 0x6e, 0xa9, 0x2f,
			}},
			want: "hxfa6341b183b48fd460b9a42884db7987a46ea92f",
		},
		{
			name: "Contract",
			arg: arg{true, []byte{
				0xfa, 0x63, 0x41, 0xb1, 0x83, 0xb4, 0x8f, 0xd4, 0x60, 0xb9,
				0xa4, 0x28, 0x84, 0xdb, 0x79, 0x87, 0xa4, 0x6e, 0xa9, 0x2f,
			}},
			want: "cxfa6341b183b48fd460b9a42884db7987a46ea92f",
		},
		{
			name: "EOAShort",
			arg: arg{false, []byte{
				0xa4, 0x28, 0x84, 0xdb, 0x79, 0x87, 0xa4, 0x6e, 0xa9, 0x2f,
			}},
			want: "hx00000000000000000000a42884db7987a46ea92f",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr := new(Address)
			addr.SetTypeAndID(test.arg.contract, test.arg.id)
			assert.Equal(t, test.want, addr.String())
		})
	}
}
