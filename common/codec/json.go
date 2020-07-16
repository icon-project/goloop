package codec

import (
	"encoding/json"
	"io"
)

var jsonCodecObject jsonCodec
var JSON = bytesWrapper{&jsonCodecObject}

type jsonCodec struct {
}

func (c *jsonCodec) Name() string {
	return "json"
}

func (c *jsonCodec) NewEncoder(w io.Writer) SimpleEncoder {
	return json.NewEncoder(w)
}

func (c *jsonCodec) NewDecoder(r io.Reader) SimpleDecoder {
	return json.NewDecoder(r)
}
