package intconv

import (
	"encoding/hex"
	"math/big"
	"regexp"
	"strconv"

	"github.com/icon-project/goloop/common/errors"
)

func FormatBigInt(i *big.Int) string {
	return encodeHexNumber(i.Sign() < 0, i.Bytes())
}

var underDigit = regexp.MustCompile(`_([0-9]+)`)

func ParseBigInt(i *big.Int, s string) error {
	s2 := s
	base := 0
	if len(s2) > 0 && s2[0] == '-' {
		s2 = s2[1:]
	}
	if len(s2) > 1 && s2[0] == '0' {
		switch s2[1] {
		case 'o', 'O', 'X', 'b', 'B':
			return errors.Errorf("InvalidPrefix(str=%q)", s)
		case 'x':
			break
		default:
			base = 10
			s = underDigit.ReplaceAllString(s, "$1")
		}
	}
	if _, ok := i.SetString(s, base); ok {
		return nil
	}
	return errors.Errorf("InvalidNumberFormat(str=%q)", s)
}

func encodeHexNumber(neg bool, b []byte) string {
	s := hex.EncodeToString(b)
	if len(s) == 0 {
		return "0x0"
	}
	if s[0] == '0' {
		s = s[1:]
	}
	if neg {
		return "-0x" + s
	} else {
		return "0x" + s
	}
}

func ParseInt(s string, bits int) (int64, error) {
	if v, err := strconv.ParseInt(s, 0, bits); err != nil {
		return 0, err
	} else {
		return v, nil
	}
}

func ParseUint(s string, bits int) (uint64, error) {
	if v, err := strconv.ParseUint(s, 0, bits); err != nil {
		return 0, err
	} else {
		return v, nil
	}
}

func FormatInt(v int64) string {
	var bs []byte
	if v < 0 {
		bs = SizeToBytes(uint64(-v))
		return encodeHexNumber(true, bs)
	} else {
		bs = SizeToBytes(uint64(v))
		return encodeHexNumber(false, bs)
	}
}

func FormatUint(v uint64) string {
	return encodeHexNumber(false, SizeToBytes(v))
}
