package intconv

import (
	"math"
	"math/big"
)

var BigIntOne = big.NewInt(1)

func BytesForZero() []byte {
	return []byte{0}
}

func BigIntToBytes(i *big.Int) []byte {
	if i == nil || i.Sign() == 0 {
		return BytesForZero()
	} else if i.Sign() > 0 {
		bl := i.BitLen()
		if (bl % 8) == 0 {
			bs := make([]byte, bl/8+1)
			copy(bs[1:], i.Bytes())
			return bs
		}
		return i.Bytes()
	} else {
		var ti, nb big.Int
		ti.Add(i, BigIntOne)
		bl := ti.BitLen()
		nb.SetBit(&nb, (bl+8)/8*8, 1)
		nb.Add(&nb, i)
		return nb.Bytes()
	}
}

func BigIntSetBytes(i *big.Int, bs []byte) *big.Int {
	i.SetBytes(bs)
	if len(bs) > 0 && (bs[0]&0x80) != 0 {
		var base big.Int
		base.SetBit(&base, i.BitLen(), 1)
		i.Sub(i, &base)
	}
	return i
}

func Uint64ToBytes(v uint64) []byte {
	if v == 0 {
		return BytesForZero()
	}
	bs := make([]byte, 9)
	for idx := 8; idx >= 0; idx-- {
		tv := byte(v & 0xff)
		bs[idx] = tv
		v >>= 8
		if v == 0 && (tv&0x80) == 0 {
			return bs[idx:]
		}
	}
	return bs
}

func SizeToBytes(v uint64) []byte {
	if v == 0 {
		return BytesForZero()
	}
	bs := make([]byte, 8)
	for idx := 7; idx >= 0; idx-- {
		bs[idx] = byte(v & 0xff)
		v >>= 8
		if v == 0 {
			return bs[idx:]
		}
	}
	return bs
}

func SafeBytesToUint64(bs []byte) (uint64, bool) {
	if len(bs) == 0 {
		return 0, true
	}
	if b := bs[0]; b == 0 {
		bs = bs[1:]
	} else if (b & 0x80) != 0 {
		return 0, false
	}
	if len(bs) > 8 {
		return 0, false
	}
	var v uint64
	for _, b := range bs {
		v = (v << 8) | uint64(b)
	}
	return v, true
}

func BytesToUint64(bs []byte) uint64 {
	if value, ok := SafeBytesToUint64(bs); ok {
		return value
	} else {
		panic("BytesToUint64 overflow")
	}
}

func SafeBytesToSize(bs []byte) (int, bool) {
	if s64, ok := SafeBytesToSize64(bs); ok && s64 <= math.MaxInt {
		return int(s64), true
	}
	return 0, false
}

func SafeBytesToSize64(bs []byte) (uint64, bool) {
	if len(bs) == 0 {
		return 0, true
	}
	if len(bs) > 8 {
		return 0, false
	}
	var v uint64
	for _, b := range bs {
		v = (v << 8) | uint64(b)
	}
	return v, true
}

func SafeBytesToInt64(bs []byte) (int64, bool) {
	if len(bs) == 0 {
		return 0, true
	}
	if len(bs) > 8 {
		return 0, false
	}
	var v int64
	if (bs[0] & 0x80) != 0 {
		for _, b := range bs {
			v = (v << 8) | int64(b^0xff)
		}
		return -v - 1, true
	} else {
		for _, b := range bs {
			v = (v << 8) | int64(b)
		}
		return v, true
	}
}

func BytesToInt64(bs []byte) int64 {
	if value, ok := SafeBytesToInt64(bs); ok {
		return value
	} else {
		panic("Int64Overflow")
	}
}

func Int64ToBytes(v int64) []byte {
	if v == 0 {
		return BytesForZero()
	}
	bs := make([]byte, 8)

	const mask int64 = -0x80
	var target int64 = 0
	if v < 0 {
		target = mask
	}
	for idx := 7; idx >= 0; idx-- {
		bs[idx] = byte(v & 0xff)
		if (v & mask) == target {
			return bs[idx:]
		}
		v >>= 8
	}
	return bs
}
