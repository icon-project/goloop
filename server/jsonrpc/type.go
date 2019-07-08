package jsonrpc

import (
	"encoding/hex"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type HexBytes string

func (hs HexBytes) Bytes() []byte {
	bs, _ := hex.DecodeString(string(hs[2:]))
	return bs
}

type HexInt string

func (i HexInt) ParseInt(bits int) (int64, error) {
	return common.ParseInt(string(i), bits)
}

func (i HexInt) Value() int64 {
	v, err := common.ParseInt(string(i), 64)
	if err != nil {
		return 0
	}
	return v
}

type Address string

func (addr Address) Address() module.Address {
	return common.NewAddressFromString(string(addr))
}
