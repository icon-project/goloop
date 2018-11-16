package codec

import (
	"encoding/json"
	"io"
)

var jsonCodecObject jsonCodec
var JSON = bytesWrapper{&jsonCodecObject}

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
