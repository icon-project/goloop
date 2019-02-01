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

func MustMarshalToBytes(v interface{}) []byte {
	return codec.MustMarshalToBytes(v)
}

func MustUnmarshalFromBytes(b []byte, v interface{}) []byte {
	return codec.MustUnmarshalFromBytes(b, v)
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

func (c *bytesWrapper) MustMarshalToBytes(v interface{}) []byte {
	bs, err := MarshalToBytes(v)
	if err != nil {
		panic(err)
		return nil
	} else {
		return bs
	}
}

func (c *bytesWrapper) MustUnmarshalFromBytes(b []byte, v interface{}) []byte {
	bs, err := UnmarshalFromBytes(b, v)
	if err != nil {
		panic(err)
		return nil
	} else {
		return bs
	}
}
