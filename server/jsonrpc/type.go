package jsonrpc

import (
	"encoding/hex"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
)

type HexBytes string

func (hs HexBytes) Bytes() []byte {
	bs, _ := hex.DecodeString(string(hs[2:]))
	return bs
}

type HexInt string

func (i HexInt) ParseInt(bits int) (int64, error) {
	if len(i) == 0 {
		return 0, nil
	}
	return intconv.ParseInt(string(i), bits)
}

func (i HexInt) Value() int64 {
	v, err := i.Int64()
	if err != nil {
		return 0
	}
	return v
}

func (i HexInt) Int64() (int64, error) {
	return i.ParseInt(64)
}

func (i HexInt) BigInt() (*big.Int, error) {
	bi := new(big.Int)
	if err := intconv.ParseBigInt(bi, string(i)); err != nil {
		return nil, err
	} else {
		return bi, nil
	}
}

type HexBool string

func (b HexBool) Bool() (bool, error) {
	return common.ParseHexBool(string(b))
}

type Address string

func (addr Address) Address() module.Address {
	return common.MustNewAddressFromString(string(addr))
}
