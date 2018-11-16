package codec

import (
	ugorji "github.com/ugorji/go/codec"
	"io"
)

var mpCodecObject mpCodec
var MP = bytesWrapper{&mpCodecObject}

type mpCodec struct {
	handle *ugorji.MsgpackHandle
}

func (c *mpCodec) Marshal(w io.Writer, v interface{}) error {
	e := ugorji.NewEncoder(w, c.handle)
	return e.Encode(v)
}

func (c *mpCodec) Unmarshal(r io.Reader, v interface{}) error {
	e := ugorji.NewDecoder(r, c.handle)
	return e.Decode(v)
}

func init() {
	mh := new(ugorji.MsgpackHandle)
	mh.StructToArray = true
	mh.Canonical = true
	mpCodecObject.handle = mh
}
