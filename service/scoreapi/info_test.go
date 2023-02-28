package scoreapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
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
	{
		Type:    Fallback,
		Name:    "fallback",
		Flags:   FlagExternal | FlagPayable,
		Indexed: 0,
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

func TestInfo_Codec(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var info *Info
		bs, err := codec.BC.MarshalToBytes(info)
		assert.NoError(t, err)
		assert.NotEmpty(t, bs)

		var info1 *Info
		_, err = codec.BC.UnmarshalFromBytes(bs, &info1)
		assert.NoError(t, err)
		assert.Nil(t, info)
		assert.True(t, info.Equal(info1))
	})
	t.Run( "Simple", func(t *testing.T) {
		info := NewInfo(testMethods)
		bs, err := codec.BC.MarshalToBytes(info)
		assert.NoError(t, err)
		assert.NotEmpty(t, bs)

		var info1 *Info
		_, err = codec.BC.UnmarshalFromBytes(bs, &info1)
		assert.NoError(t, err)
		assert.EqualValues(t, info, info1)
		assert.True(t, info.Equal(info1))
	})
	t.Run("Invalid", func(t *testing.T) {
		var value = []byte("test")
		bs, err := codec.BC.MarshalToBytes(value)
		assert.NoError(t, err)
		assert.NotEmpty(t, bs)

		var info1 *Info
		_, err = codec.BC.UnmarshalFromBytes(bs, &info1)
		assert.Error(t, err)
	})
}
