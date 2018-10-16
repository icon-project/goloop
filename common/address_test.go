package common

import (
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

func TestAddress_String(t *testing.T) {
	type fields struct {
		isContract bool
		bytes      []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
		{
			"AddressString1-1",
			fields{false, addr1_bytes},
			addr1_str1,
		},
		{
			"AddressString1-2",
			fields{true, addr1_bytes},
			addr1_str2,
		},
		{
			"AddressString2-1",
			fields{false, addr2_bytes},
			addr2_str1,
		},
		{
			"AddressString2-2",
			fields{true, addr2_bytes},
			addr2_str2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Address{
				isContract: tt.fields.isContract,
				bytes:      tt.fields.bytes,
			}
			if got := a.String(); got != tt.want {
				t.Errorf("Address.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
