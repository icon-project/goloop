package codec

import (
	"io"

	"gopkg.in/vmihailenco/msgpack.v4"
)

var MP = bytesWrapper{&mpCodecObject}

var mpCodecObject mpCodec

type mpCodec struct {
}

func (c *mpCodec) NewEncoder(w io.Writer) Encoder {
	e := msgpack.NewEncoder(w)
	e.UseCompactEncoding(true)
	e.StructAsArray(true)
	e.SortMapKeys(true)
	return e
}

func (c *mpCodec) NewDecoder(r io.Reader) Decoder {
	return msgpack.NewDecoder(r)
}
