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
	JSON = bytesWrapper{&jsonCodecObject}
	MP   = bytesWrapper{&mpCodecObject}
)

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
