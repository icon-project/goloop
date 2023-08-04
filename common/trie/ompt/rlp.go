package ompt

import (
	"errors"
	"fmt"

	cerrors "github.com/icon-project/goloop/common/errors"
)

var (
	errRLPNotEnoughBytes  = errors.New("RLP:Not enough bytes to decode")
	errRLPInvalidEncoding = errors.New("RLP:Invalid encoding")
)

func rlpReadSize(b []byte, slen byte) (uint64, error) {
	if int(slen) > len(b) {
		return 0, cerrors.WithStack(errRLPNotEnoughBytes)
	}
	var s uint64
	switch slen {
	case 1:
		s = uint64(b[0])
	case 2:
		s = uint64(b[0])<<8 | uint64(b[1])
	case 3:
		s = uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		s = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		s = uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
	case 6:
		s = uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	case 7:
		s = uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 | uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		s = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	if s < 56 || b[0] == 0 {
		return 0, cerrors.WithStack(errRLPInvalidEncoding)
	}
	return s, nil
}

func rlpParseHeader(buf []byte) (bool, uint64, uint64, error) {
	if len(buf) == 0 {
		return false, 0, 0, cerrors.WithStack(errRLPNotEnoughBytes)
	}
	b := buf[0]
	var tagsize uint64
	var contentsize uint64
	var err error
	var islist bool
	switch {
	case b < 0x80:
		tagsize = 0
		contentsize = 1
	case b < 0xB8:
		tagsize = 1
		contentsize = uint64(b - 0x80)
		// Reject strings that should've been single bytes.
		if contentsize == 1 && len(buf) > 1 && buf[1] < 128 {
			return false, 0, 0, cerrors.WithStack(errRLPInvalidEncoding)
		}
	case b < 0xC0:
		tagsize = uint64(b-0xB7) + 1
		contentsize, err = rlpReadSize(buf[1:], b-0xB7)
	case b < 0xF8:
		islist = true
		tagsize = 1
		contentsize = uint64(b - 0xC0)
	default:
		islist = true
		tagsize = uint64(b-0xF7) + 1
		contentsize, err = rlpReadSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return false, 0, 0, err
	}
	// Reject values larger than the input slice.
	if contentsize > uint64(len(buf))-tagsize {
		return false, 0, 0, cerrors.WithStack(errRLPNotEnoughBytes)
	}
	return islist, tagsize, contentsize, err
}

func rlpParseList(b []byte) ([][]byte, error) {
	islist, tsize, csize, err := rlpParseHeader(b)
	if err != nil {
		return nil, err
	}
	if !islist {
		return nil, cerrors.WithStack(cerrors.ErrIllegalArgument)
	}
	if uint64(len(b)) < tsize+csize {
		return nil, cerrors.WithStack(errRLPNotEnoughBytes)
	}
	b = b[tsize : tsize+csize]
	var items = [][]byte{}
	for len(b) > 0 {
		_, tsize, csize, err := rlpParseHeader(b)
		if err != nil {
			return nil, err
		}
		if uint64(len(b)) < tsize+csize {
			return nil, cerrors.WithStack(errRLPNotEnoughBytes)
		}
		items = append(items, b[:tsize+csize])
		b = b[tsize+csize:]
	}
	return items, nil
}

func rlpParseBytes(b []byte) ([]byte, error) {
	islist, tsize, csize, err := rlpParseHeader(b)
	if err != nil {
		return nil, err
	}
	if islist {
		return nil, cerrors.WithStack(cerrors.ErrIllegalArgument)
	}
	if uint64(len(b)) < tsize+csize {
		return nil, cerrors.WithStack(errRLPNotEnoughBytes)
	}
	return b[tsize : tsize+csize], nil
}

func rlpCountBytesForSize(b int) int {
	var cnt = int(1)
	for b >>= 8; b > 0; cnt++ {
		b >>= 8
	}
	return cnt
}

func rlpEncodeBytes(b []byte) []byte {
	var blen = len(b)
	if blen == 1 && b[0] < 0x80 {
		return b
	}
	if blen <= 55 {
		buf := make([]byte, blen+1)
		buf[0] = byte(0x80 + blen)
		copy(buf[1:], b)
		return buf
	}
	tslen := rlpCountBytesForSize(blen)
	buf := make([]byte, 1+tslen+blen)
	buf[0] = byte(0x80 + 55 + tslen)
	for tsidx := tslen; tsidx > 0; tsidx-- {
		buf[tsidx] = byte(blen & 0xff)
		blen >>= 8
	}
	copy(buf[tslen+1:], b)
	return buf
}

func rlpEncodeList(blist [][]byte) []byte {
	blen := 0
	for _, b := range blist {
		blen += len(b)
	}
	if blen <= 55 {
		buf := make([]byte, blen+1)
		buf[0] = byte(0xC0 + blen)
		bidx := buf[1:]
		for _, b := range blist {
			copy(bidx, b)
			bidx = bidx[len(b):]
		}
		return buf
	}

	tslen := rlpCountBytesForSize(blen)
	buf := make([]byte, 1+tslen+blen)
	buf[0] = byte(0xC0 + 55 + tslen)
	for tsidx := tslen; tsidx > 0; tsidx-- {
		buf[tsidx] = byte(blen & 0xff)
		blen >>= 8
	}
	bidx := buf[1+tslen:]
	for _, b := range blist {
		copy(bidx, b)
		bidx = bidx[len(b):]
	}
	return buf
}

type RLPEncoder interface {
	RLPEncode(o interface{}) error
	RLPWrite(b []byte)
}

type rlpListEncoder struct {
	data [][]byte
}

func (e *rlpListEncoder) RLPEncode(o interface{}) error {
	b, err := rlpEncode(o)
	if err != nil {
		return err
	}
	e.data = append(e.data, b)
	return nil
}

func (e *rlpListEncoder) RLPWrite(b []byte) {
	e.data = append(e.data, b)
}

func (e *rlpListEncoder) RLPSerialize() []byte {
	return rlpEncodeList(e.data)
}

type RLPListEncoder interface {
	RLPListSize() int
	RLPListEncode(e RLPEncoder) error
}

type RLPSerializer interface {
	RLPSerialize() []byte
}

func rlpEncode(o interface{}) ([]byte, error) {
	switch o := o.(type) {
	case []byte:
		return rlpEncodeBytes(o), nil
	case [][]byte:
		blist := make([][]byte, len(o))
		for i := range blist {
			blist[i] = rlpEncodeBytes(o[i])
		}
		return rlpEncodeList(blist), nil
	case []interface{}:
		blist := make([][]byte, len(o))
		for i := range blist {
			b, err := rlpEncode(o[i])
			if err != nil {
				return nil, err
			}
			blist[i] = b
		}
		return rlpEncodeList(blist), nil
	case RLPListEncoder:
		sz := o.RLPListSize()
		e := &rlpListEncoder{make([][]byte, 0, sz)}
		if err := o.RLPListEncode(e); err != nil {
			return nil, err
		}
		return e.RLPSerialize(), nil
	case RLPSerializer:
		return o.RLPSerialize(), nil
	case nil:
		return rlpEncodeBytes(nil), nil
	default:
		return nil, fmt.Errorf("Fail to encode object type %T", o)
	}
}
