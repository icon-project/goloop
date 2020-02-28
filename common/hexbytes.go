package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type RawHexBytes []byte

func (rh RawHexBytes) MarshalJSON() ([]byte, error) {
	if rh == nil {
		return []byte("null"), nil
	}
	s := hex.EncodeToString(rh)
	return json.Marshal(s)
}

func (rh *RawHexBytes) UnmarshalJSON(b []byte) error {
	var os *string
	if err := json.Unmarshal(b, &os); err != nil {
		return err
	}
	if os == nil {
		*rh = nil
		return nil
	}
	s := *os
	if bin, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		*rh = bin
		return nil
	}
}

func (rh RawHexBytes) Bytes() []byte {
	if rh == nil {
		return nil
	}
	return rh[:]
}

func (rh RawHexBytes) String() string {
	if rh == nil {
		return "null"
	}
	return hex.EncodeToString(rh)
}

type HexBytes []byte

func (hs HexBytes) MarshalJSON() ([]byte, error) {
	if hs == nil {
		return []byte("null"), nil
	}
	s := "0x" + hex.EncodeToString(hs)
	return json.Marshal(s)
}

func (hs *HexBytes) UnmarshalJSON(b []byte) error {
	var os *string
	if err := json.Unmarshal(b, &os); err != nil {
		return err
	}
	if os == nil {
		*hs = nil
		return nil
	}
	s := *os
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if bin, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		*hs = bin
		return nil
	}
}

func (hs HexBytes) Bytes() []byte {
	if hs == nil {
		return nil
	}
	return hs[:]
}

func (hs HexBytes) String() string {
	if hs == nil {
		return "null"
	}
	return "0x" + hex.EncodeToString(hs)
}

const PrefixLen = 4

// HexPre returns hexadecimal string of prefix of byte slice bs.
func HexPre(bs []byte) string {
	if bs == nil {
		return "<nil>"
	}
	if len(bs) > PrefixLen {
		return fmt.Sprintf("%x..", bs[:PrefixLen])
	}
	return fmt.Sprintf("%x", bs)
}

func SliceOfHexBytes(bss [][]byte) []HexBytes {
	hbs := make([]HexBytes, len(bss))
	for i, bs := range bss {
		hbs[i] = bs
	}
	return hbs
}
