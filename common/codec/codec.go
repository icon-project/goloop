package codec

import (
	"bytes"
	"io"
)

type Codec interface {
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
}

var (
	codec = MP
)

func Marshal(w io.Writer, v interface{}) error {
	return codec.Marshal(w, v)
}

func Unmarshal(r io.Reader, v interface{}) error {
	return codec.Unmarshal(r, v)
}

func MarshalToBytes(v interface{}) ([]byte, error) {
	return codec.MarshalToBytes(v)
}

func UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	return codec.UnmarshalFromBytes(b, v)
}

type bytesWrapper struct {
	Codec
}

func (c *bytesWrapper) MarshalToBytes(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := c.Marshal(buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *bytesWrapper) UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(b)
	if err := c.Unmarshal(buf, v); err != nil {
		return b, err
	}
	return buf.Bytes(), nil
}
