package common

import (
	"bytes"
	"encoding/hex"
	"log"
	"reflect"
	"testing"

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
			if got := NewAddressFromString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
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

			a := NewAddressFromString(tt.args.s)
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
			args: args{NewAddressFromString("hx8888888888888888888888888888888888888888")},
			want: false,
		},
		{
			name: "NonNilvsNil",
			a:    NewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{nil},
			want: false,
		},
		{
			name: "NonNilvsNilPtr",
			a:    NewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{(*Address)(nil)},
			want: false,
		},
		{
			name: "Same1",
			a:    NewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{NewAddressFromString("hx8888888888888888888888888888888888888888")},
			want: true,
		},
		{
			name: "Same2",
			a:    NewAddressFromString("cx8888888888888888888888888888888888888888"),
			args: args{NewAddressFromString("cx8888888888888888888888888888888888888888")},
			want: true,
		},
		{
			name: "Diff1",
			a:    NewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{NewAddressFromString("cx8888888888888888888888888888888888888888")},
			want: false,
		},
		{
			name: "Diff2",
			a:    NewAddressFromString("hx8888888888888888888888888888888888888888"),
			args: args{NewAddressFromString("hx9888888888888888888888888888888888888888")},
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
			wantErr: false,
		},
		{
			name:    "EOANoID",
			a:       Address{},
			args:    args{[]byte{0}},
			wantErr: false,
		},
		{
			name:    "EOAWithID20",
			a:       Address{},
			args:    args{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			wantErr: false,
		},
		{
			name:    "EOAWithID21",
			a:       Address{},
			args:    args{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1}},
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
