package common

import (
	"encoding/hex"
	"encoding/json"
)

type RawHexBytes []byte

func (rh RawHexBytes) MarshalJSON() ([]byte, error) {
	if rh == nil {
		return []byte("nil"), nil
	}
	s := hex.EncodeToString(rh)
	return json.Marshal(s)
}

func (rh *RawHexBytes) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
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
		return ""
	}
	return hex.EncodeToString(rh)
}

type HexBytes []byte

func (hs HexBytes) MarshalJSON() ([]byte, error) {
	if hs == nil {
		return []byte("nil"), nil
	}
	s := "0x" + hex.EncodeToString(hs)
	return json.Marshal(s)
}

func (hs *HexBytes) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
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
		return ""
	}
	return "0x" + hex.EncodeToString(hs)
}
