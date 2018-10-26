package ompt

import (
	"io"
)

func makePrefix(l, prefix int) []byte {
	if l <= 55 {
		return []byte{byte(prefix + l)}
	}

	prefix += 55
	bLen := 0
	tmp := l
	for {
		if tmp == 0 {
			break
		}
		tmp = tmp / 0x100
		bLen++
	}

	r := make([]byte, bLen+1)

	for i := range r {
		if i == 0 {
			r[0] = byte(prefix + bLen)
		} else {
			r[i] = byte(l >> uint(8*bLen) & 0xff)
		}
		bLen--
	}
	return r
}

func encodeByte(d []byte) []byte {
	l := len(d)
	if l == 0 {
		return []byte{0x80}
	}
	if l == 1 && d[0] < 0x80 {
		return d
	}
	return append(makePrefix(l, 0x80), d...)
}

func encodeList(data ...[]byte) []byte {
	r := make([]byte, 0)
	for _, d := range data {
		r = append(r, d...)
	}
	return append(makePrefix(len(r), 0xc0), r...)
}

// TODO: have to modify. ethereum code
func readSize(b []byte, slen byte) (uint64, error) {
	if int(slen) > len(b) {
		return 0, io.ErrUnexpectedEOF
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
		return 0, nil // TODO: define proper error
	}
	return s, nil
}

// TODO: have to modify. ethereum code
func getContentSize(buf []byte) (uint64, uint64, error) {
	if len(buf) == 0 {
		return 0, 0, nil // TODO: define proper error
	}
	b := buf[0]
	var tagsize uint64
	var contentsize uint64
	var err error
	switch {
	case b < 0x80:
		tagsize = 0
		contentsize = 1
	case b < 0xB8:
		tagsize = 1
		contentsize = uint64(b - 0x80)
		// Reject strings that should've been single bytes.
		if contentsize == 1 && len(buf) > 1 && buf[1] < 128 {
			return 0, 0, nil // TODO: define proper error
		}
	case b < 0xC0:
		tagsize = uint64(b-0xB7) + 1
		contentsize, err = readSize(buf[1:], b-0xB7)
	case b < 0xF8:
		tagsize = 1
		contentsize = uint64(b - 0xC0)
	default:
		tagsize = uint64(b-0xF7) + 1
		contentsize, err = readSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return 0, 0, err
	}
	// Reject values larger than the input slice.
	if contentsize > uint64(len(buf))-tagsize {
		return 0, 0, nil
	}
	return tagsize, contentsize, err
}

func countListMember(b []byte) (int, error) {
	i := 0
	listTagsize, _, _ := getContentSize(b)
	list := b[listTagsize:]
	for ; len(list) > 0; i++ {
		tagsize, size, err := getContentSize(list)
		if err != nil {
			return 0, err
		}
		list = list[tagsize+size:]
	}
	return i, nil
}
