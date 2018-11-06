package codec

import (
	"github.com/ugorji/go/codec"
	"io"
)

var mpCodecObject mpCodec

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

func init() {
	mh := new(codec.MsgpackHandle)
	mh.StructToArray = true
	mh.Canonical = true
	mpCodecObject.handle = mh
}
