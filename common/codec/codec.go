package codec

import (
	"bytes"
	"encoding/json"
	"github.com/ugorji/go/codec"
	"io"
)

type Codec interface {
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
}

var (
	JSON jsonCodec
	MP   mpCodec
)

type jsonCodec struct {
}

func (c *jsonCodec) Marshal(w io.Writer, v interface{}) error {
	e := json.NewEncoder(w)
	return e.Encode(v)
}

func (c *jsonCodec) Unmarshal(r io.Reader, v interface{}) error {
	d := json.NewDecoder(r)
	return d.Decode(v)
}

type normalCodec interface {
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
}

func marshalToBytes(c normalCodec, v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := c.Marshal(buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshalFromBytes(c normalCodec, b []byte, v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(b)
	if err := c.Unmarshal(buf, v); err != nil {
		return b, err
	}
	return buf.Bytes(), nil
}

func (c *jsonCodec) MarshalToBytes(v interface{}) ([]byte, error) {
	return marshalToBytes(c, v)
}

func (c *jsonCodec) UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	return unmarshalFromBytes(c, b, v)
}

type mpCodec struct {
	handle *codec.MsgpackHandle
}

func (c *mpCodec) Marshal(w io.Writer, v interface{}) error {
	e := codec.NewEncoder(w, c.handle)
	return e.Encode(v)
}

func (c *mpCodec) Unmarshal(r io.Reader, v interface{}) error {
	e := codec.NewDecoder(r, c.handle)
	return e.Decode(v)
}

func (c *mpCodec) MarshalToBytes(v interface{}) ([]byte, error) {
	return marshalToBytes(c, v)
}

func (c *mpCodec) UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	return unmarshalFromBytes(c, b, v)
}

func init() {
	mh := new(codec.MsgpackHandle)
	mh.StructToArray = true
	mh.Canonical = true
	MP.handle = mh
}
