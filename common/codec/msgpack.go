package codec

import (
	"bytes"
	"io"
	"reflect"

	"github.com/vmihailenco/msgpack/v4"
	"github.com/vmihailenco/msgpack/v4/codes"

	"github.com/icon-project/goloop/common/errors"
)

var mpCodecObject mpCodec
var MP = bytesWrapperFrom(&mpCodecObject)

type mpCodec struct {
}

type mpReader struct {
	real *msgpack.Decoder
	size int
	read int
}

func (r *mpReader) Skip(cnt int) error {
	if err := r.consumeN(cnt); err != nil {
		return err
	}
	for i := 0; i < cnt; i++ {
		if err := r.real.Skip(); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (r *mpReader) consumeN(cnt int) error {
	if r.size >= 0 {
		if r.read+cnt > r.size {
			return io.EOF
		}
		r.read += cnt
	}
	return nil
}

func (r *mpReader) ensureNonNil() error {
	if code, err := r.real.PeekCode(); err != nil {
		return errors.WithStack(err)
	} else {
		if code == codes.Nil {
			r.real.DecodeNil()
			return ErrNilValue
		}
	}
	return nil
}

func (r *mpReader) ReadList() (Reader, error) {
	if err := r.consumeN(1); err != nil {
		return nil, err
	}
	cnt, err := r.real.DecodeArrayLen()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if cnt < 0 {
		return nil, ErrNilValue
	}
	return &mpReader{
		real: r.real,
		size: cnt,
	}, nil
}

func (r *mpReader) ReadMap() (Reader, error) {
	if err := r.consumeN(1); err != nil {
		return nil, err
	}
	cnt, err := r.real.DecodeMapLen()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if cnt < 0 {
		return nil, ErrNilValue
	}
	return &mpReader{
		real: r.real,
		size: cnt * 2,
	}, nil
}

func (r *mpReader) ReadValue(v reflect.Value) error {
	if err := r.consumeN(1); err != nil {
		return err
	}
	if err := r.ensureNonNil(); err != nil {
		return err
	}
	if err := r.real.DecodeValue(v); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *mpReader) Close() error {
	if r.read < r.size {
		return r.Skip(r.size - r.read)
	}
	return nil
}

func (r *mpReader) ReadBytes() ([]byte, error) {
	if err := r.consumeN(1); err != nil {
		return nil, err
	}
	bs, err := r.real.DecodeBytes()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if bs == nil {
		return nil, ErrNilValue
	}
	return bs, nil
}

type rawBytes []byte

func (rb *rawBytes) UnmarshalMsgpack(bs []byte) error {
	*rb = bs
	return nil
}

func (r *mpReader) ReadRaw() ([]byte, error) {
	if err := r.consumeN(1); err != nil {
		return nil, err
	}
	var rb rawBytes
	if err := r.real.Decode(&rb); err != nil {
		return nil, errors.WithStack(err)
	}
	return rb, nil
}

type mpParent struct {
	buffer *bytes.Buffer
	writer *mpWriter
	cnt    int
	isMap  bool
}

type mpWriter struct {
	parent *mpParent
	writer io.Writer
	real   *msgpack.Encoder
}

func (w *mpWriter) countN(cnt int) {
	if w.parent != nil {
		w.parent.cnt += cnt
	}
}

func (w *mpWriter) WriteList() (Writer, error) {
	w.countN(1)
	p := &mpParent{
		buffer: bytes.NewBuffer(nil),
		writer: w,
	}
	return &mpWriter{
		parent: p,
		writer: p.buffer,
		real:   msgpackNewEncoder(p.buffer),
	}, nil
}

func (w *mpWriter) WriteMap() (Writer, error) {
	w.countN(1)
	p := &mpParent{
		buffer: bytes.NewBuffer(nil),
		isMap:  true,
		writer: w,
	}
	return &mpWriter{
		parent: p,
		writer: p.buffer,
		real:   msgpackNewEncoder(p.buffer),
	}, nil
}

func (w *mpWriter) WriteBytes(b []byte) error {
	w.countN(1)
	return w.real.EncodeBytes(b)
}

func (w *mpWriter) WriteRaw(b []byte) error {
	w.countN(1)
	for written := 0; written < len(b); {
		n, err := w.writer.Write(b[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}

func (w *mpWriter) WriteValue(v reflect.Value) error {
	w.countN(1)
	return w.real.EncodeValue(v)
}

func (w *mpWriter) WriteNull() error {
	w.countN(1)
	return w.real.EncodeNil()
}

func (w *mpWriter) Close() error {
	if p := w.parent; p != nil {
		if p.isMap {
			if (p.cnt % 2) != 0 {
				return ErrInvalidFormat
			}
			if err := p.writer.real.EncodeMapLen(p.cnt / 2); err != nil {
				return err
			}
		} else {
			if err := p.writer.real.EncodeArrayLen(p.cnt); err != nil {
				return err
			}
		}
		if _, err := p.writer.writer.Write(p.buffer.Bytes()); err != nil {
			return err
		}
		w.parent = nil
	}
	return nil
}

func (c *mpCodec) Name() string {
	return "msgpack"
}

func (c *mpCodec) NewDecoder(r io.Reader) DecodeAndCloser {
	return NewDecoder(&mpReader{
		real: msgpack.NewDecoder(r),
		size: -1,
	})
}

func msgpackNewEncoder(w io.Writer) *msgpack.Encoder {
	e := msgpack.NewEncoder(w)
	e.SortMapKeys(true)
	e.UseCompactEncoding(true)
	e.StructAsArray(true)
	return e
}

func (c *mpCodec) NewEncoder(w io.Writer) EncodeAndCloser {
	return NewEncoder(&mpWriter{
		writer: w,
		real:   msgpackNewEncoder(w),
	})
}
