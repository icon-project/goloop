package common

import (
	"encoding/json"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
)

type HexBool struct {
	Value bool
}

func (i *HexBool) String() string {
	if i.Value {
		return "0x1"
	} else {
		return "0x0"
	}
}

func (i *HexBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexBool) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if i.Value, err = ParseHexBool(s); err != nil {
		return err
	}
	return nil
}

func (i *HexBool) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(i.Value)
}

func (i *HexBool) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

func ParseHexBool(s string) (bool, error) {
	if s == "0x1" {
		return true, nil
	} else if s == "0x0" {
		return false, nil
	} else {
		return false, errors.New("invalid value")
	}
}
