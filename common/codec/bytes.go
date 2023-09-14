package codec

import (
	"bytes"
	"io"
	"sync"

	"github.com/icon-project/goloop/common/log"
)

type bytesEncoder struct {
	Encoder
	buffer *bytes.Buffer
}

func (e *bytesEncoder) Reset() {
	e.buffer.Reset()
}

func (e *bytesEncoder) Bytes() []byte {
	return e.buffer.Bytes()
}

type bytesDecoder struct {
	Decoder
	buffer *bytes.Buffer
}

func (d *bytesDecoder) Reset(bs []byte) {
	d.buffer.Reset()
	d.buffer.Write(bs)
	d.Decoder.SetMaxBytes(len(bs))
}

func (d *bytesDecoder) Bytes() []byte {
	return d.buffer.Bytes()
}

type bytesWrapper struct {
	codecImpl
	encoders sync.Pool
	decoders sync.Pool
}

func (c *bytesWrapper) Marshal(w io.Writer, v interface{}) error {
	return c.NewEncoder(w).Encode(v)
}

func (c *bytesWrapper) Unmarshal(r io.Reader, v interface{}) error {
	return c.NewDecoder(r).Decode(v)
}

func bytesDup(bs []byte) []byte {
	sz := len(bs)
	if sz != 0 {
		nbs := make([]byte,len(bs))
		copy(nbs, bs)
		return nbs
	} else {
		return []byte{}
	}
}

func (c *bytesWrapper) MarshalToBytes(v interface{}) ([]byte, error) {
	be := c.encoders.Get().(*bytesEncoder)

	be.Reset()
	if err := be.Encode(v) ; err != nil {
		return nil, err
	}
	remainder := bytesDup(be.Bytes())

	c.encoders.Put(be)
	return remainder, nil
}

func (c *bytesWrapper) UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	bd := c.decoders.Get().(*bytesDecoder)

	bd.Reset(b)
	if err := bd.Decode(v); err != nil {
		return nil, err
	}
	bs := bytesDup(bd.Bytes())

	c.decoders.Put(bd)
	return bs, nil
}

func (c *bytesWrapper) MustMarshalToBytes(v interface{}) []byte {
	bs, err := c.MarshalToBytes(v)
	if err != nil {
		log.Panicf("MustMarshalToBytes() fails for object=%T err=%+v", v, err)
		return nil
	} else {
		return bs
	}
}

func (c *bytesWrapper) MustUnmarshalFromBytes(b []byte, v interface{}) []byte {
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

func (c *bytesWrapper) NewEncoderBytes(b *[]byte) EncodeAndCloser {
	if len(*b) > 0 {
		*b = (*b)[:0]
	}
	return c.codecImpl.NewEncoder(&bytesWriter{b})
}

func bytesWrapperFrom(codec codecImpl) *bytesWrapper {
	return &bytesWrapper{
		codecImpl: codec,
		encoders: sync.Pool{
			New: func() interface{} {
				buffer := bytes.NewBuffer(nil)
				return &bytesEncoder{
					Encoder: codec.NewEncoder(buffer),
					buffer:  buffer,
				}
			},
		},
		decoders: sync.Pool{
			New: func() interface{} {
				buffer := bytes.NewBuffer(nil)
				return &bytesDecoder{
					Decoder: codec.NewDecoder(buffer),
					buffer:  buffer,
				}
			},
		},
	}
}