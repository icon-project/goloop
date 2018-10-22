package codec

import "io"

type Codec interface {
    Marshal(w io.Writer, v interface{}) error
    Unmarshal(r io.Reader, v interface{}) error
}

var (
    JSON jsonCodec
    MP mpCodec
)

type jsonCodec struct {
}

func (c *jsonCodec) Marshal(w io.Writer, v interface{}) error {
    return nil
}

func (c *jsonCodec) Unmarshal(r io.Reader, v interface{}) error {
    return nil
}

type mpCodec struct {
}

func (c *mpCodec) Marshal(w io.Writer, v interface{}) error {
    return nil
}

func (c *mpCodec) Unmarshal(r io.Reader, v interface{}) error {
    return nil
}
