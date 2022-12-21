package scoreapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testMethods = []*Method{
	{
		Type:    Event,
		Name:    "Transfer",
		Indexed: 2,
		Inputs: []Parameter{
			{
				Name: "from",
				Type: Address,
			},
			{
				Name: "to",
				Type: Address,
			},
			{
				Name: "amount",
				Type: Integer,
			},
		},
	},
	{
		Type:    Function,
		Name:    "transfer",
		Flags:   FlagExternal,
		Indexed: 0,
		Inputs: []Parameter{
			{
				Name: "from",
				Type: Address,
			},
		},
	},
}

func TestInfo_CheckEventData(t *testing.T) {
	info := NewInfo(testMethods)

	var cases = []struct {
		name    string
		indexed [][]byte
		data    [][]byte
		wantErr bool
	}{
		{
			"Simple",
			[][]byte{
				[]byte("Transfer(Address,Address,int)"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x12"),
			},
			[][]byte{
				[]byte{0x12, 0x34},
			},
			false,
		},
		{
			"TooMany",
			[][]byte{
				[]byte("Transfer(Address,Address,int)"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x12"),
			},
			[][]byte{
				{0x12, 0x34},
				{0x55, 0x66},
			},
			true,
		},
		{
			"SmallIndexed",
			[][]byte{
				[]byte("Transfer(Address,Address,int)"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11"),
			},
			[][]byte{
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x12"),
				{0x12, 0x34},
			},
			true,
		},
		{
			"InvalidData",
			[][]byte{
				[]byte("Transfer(Address,Address,int)"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11"),
				[]byte("\x03\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x12"),
			},
			[][]byte{
				{0x12, 0x34},
			},
			true,
		},
		{
			"NonExisting",
			[][]byte{
				[]byte("TransferEx(Address,Address,int)"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11"),
				[]byte("\x01\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x12"),
			},
			[][]byte{
				[]byte{0x12, 0x34},
			},
			true,
		},
		{
			"NoSignature",
			[][]byte{},
			[][]byte{},
			true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := info.CheckEventData(tt.indexed, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
