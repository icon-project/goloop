package common

import (
	"bytes"
	"encoding/hex"
	"github.com/icon-project/goloop/common/codec"
	"log"
	"reflect"
	"testing"
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
			"b5000000000000000000000000000000000000000000",
		},
		{
			"Case2",
			args{"cx1908581ed9f09c45810405897123badefcbfefa0"},
			"b5011908581ed9f09c45810405897123badefcbfefa0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := hex.DecodeString(tt.want)
			if err != nil {
				log.Printf("Test(%s) want=%s illegal", tt.name, tt.want)
				return
			}

			a := NewAddressFromString(tt.args.s)
			b, err := codec.MP.MarshalToBytes(a)
			if err != nil {
				t.Error(err)
				return
			}
			log.Printf("Encoded:[%x]", b)
			log.Printf("Expect :[%x]", want)
			if !bytes.Equal(b, want) {
				t.Errorf("Encoded bytes are different exp=[%x] result=[%x]", want, b)
			}

			var a2 Address
			_, err = codec.MP.UnmarshalFromBytes(b, &a2)
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
