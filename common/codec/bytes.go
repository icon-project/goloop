package codec

import (
	"bytes"
	"io"

	"github.com/icon-project/goloop/common/log"
)

type bytesWrapper struct {
	codecImpl
}

func (c bytesWrapper) Marshal(w io.Writer, v interface{}) error {
	return c.NewEncoder(w).Encode(v)
}

func (c bytesWrapper) Unmarshal(r io.Reader, v interface{}) error {
	return c.NewDecoder(r).Decode(v)
}

func (c bytesWrapper) MarshalToBytes(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := c.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c bytesWrapper) UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(b)
	if err := c.NewDecoder(buf).Decode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c bytesWrapper) MustMarshalToBytes(v interface{}) []byte {
	bs, err := c.MarshalToBytes(v)
	if err != nil {
		log.Panicf("MustMarshalToBytes() fails for object=%T err=%+v", v, err)
		return nil
	} else {
		return bs
	}
}

func (c bytesWrapper) MustUnmarshalFromBytes(b []byte, v interface{}) []byte {
	bs, err := c.UnmarshalFromBytes(b, v)
	if err != nil {
		log.Panicf("MustUnmarshalFromBytes() fails for bytes=% x buffer=%T err=%+v", b, v, err)
		return nil
	} else {
		return bs
	}
}

type bytesWriter struct {
	buf *[]byte
}

func (w bytesWriter) Write(bs []byte) (int, error) {
	*w.buf = append(*w.buf, bs...)
	return len(bs), nil
}

func (c bytesWrapper) NewEncoderBytes(b *[]byte) SimpleEncoder {
	if len(*b) > 0 {
		*b = (*b)[:0]
	}
	return c.codecImpl.NewEncoder(&bytesWriter{b})
}
