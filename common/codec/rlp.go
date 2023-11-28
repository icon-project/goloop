package codec

import (
	"bytes"
	"io"
	"reflect"
	"sync"

	cerrors "github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
)

var rlpCodecObject rlpCodec
var RLP = bytesWrapperFrom(&rlpCodecObject)

// MaxSizeForBytes is size limit for bytes buffer.
// msgpack decoder already has limit to 1 MB
const MaxSizeForBytes = 1e6

type rlpCodec struct {
}

type rlpReader struct {
	reader io.Reader
	maxSB  int // maxSB is max size of bytes buffer on ReadBytes() or ReadRaw()
}

func sizeToBytes(s int) []byte {
	return intconv.SizeToBytes(uint64(s))
}

func bytesToSize(bs []byte) (int, error) {
	if value, ok := intconv.SafeBytesToSize(bs); !ok {
		return 0, cerrors.Wrapf(ErrInvalidFormat, "InvalidSizeFormat(bs=%#x)", bs)
	} else {
		return value, nil
	}
}

func minSize(sz1, sz2 int) int {
	if sz1 < sz2 {
		return sz1
	} else {
		return sz2
	}
}

type limitReader struct {
	reader io.Reader
	offset int64
	limit  int64
}

func (l *limitReader) Read(p []byte) (n int, err error) {
	avail := l.limit - l.offset
	if avail <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > avail {
		p = p[:avail]
	}
	n, err = l.reader.Read(p)
	if err == io.EOF {
		err = ErrInvalidFormat
	}
	l.offset += int64(n)
	return
}

func LimitReader(r io.Reader, n int64) io.Reader {
	return &limitReader{
		reader: r,
		limit:  n,
	}
}

func (r *rlpReader) skipN(sz int) error {
	if _, err := io.CopyN(io.Discard, r.reader, int64(sz)); err != nil {
		if err == io.EOF {
			return cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(expect=%d)", sz)
		}
		return cerrors.WithStack(err)
	}
	return nil
}

func (r *rlpReader) readSize(buffer []byte) (int, error) {
	if err := r.readAll(buffer); err != nil {
		return 0, err
	}
	return bytesToSize(buffer)
}

func (r *rlpReader) readAll(buffer []byte) error {
	if _, err := io.ReadFull(r.reader, buffer); err != nil {
		if err == io.EOF {
			return cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(sz=%d,err=%s)", len(buffer), err)
		}
		return cerrors.WithStack(err)
	}
	return nil
}

func (r *rlpReader) skipOne() error {
	var header [9]byte
	if _, err := io.ReadFull(r.reader, header[0:1]); err != nil {
		return err
	}
	tag := int(header[0])
	switch {
	case tag < 0x80:
		return nil
	case tag <= 0xB7:
		size := tag - 0x80
		return r.skipN(size)
	case tag < 0xC0:
		sz := tag - 0xB7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return err
		}
		return r.skipN(sz2)
	case tag <= 0xF7:
		sz := tag - 0xC0
		return r.skipN(sz)
	default:
		sz := tag - 0xF7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return err
		}
		return r.skipN(sz2)
	}
}

func (r *rlpReader) Skip(cnt int) error {
	for i := 0; i < cnt; i++ {
		if err := r.skipOne(); err != nil {
			return err
		}
	}
	return nil
}

func (r *rlpReader) readList() (Reader, error) {
	var header [9]byte
	if _, err := io.ReadFull(r.reader, header[0:1]); err != nil {
		return nil, err
	}
	tag := int(header[0])
	switch {
	case tag < 0xC0:
		return nil, cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RLPBytes)")
	case tag <= 0xF7:
		size := tag - 0xC0
		return &rlpReader{
			reader: LimitReader(r.reader, int64(size)),
			maxSB:  minSize(r.maxSB, size),
		}, nil
	default:
		sz := tag - 0xF7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return nil, err
		}
		if sz == 1 && sz2 == 0 {
			return nil, ErrNilValue
		}
		return &rlpReader{
			reader: LimitReader(r.reader, int64(sz2)),
			maxSB:  minSize(r.maxSB, sz2),
		}, nil
	}
}

func (r *rlpReader) ReadList() (Reader, error) {
	return r.readList()
}

func (r *rlpReader) ReadMap() (Reader, error) {
	return r.readList()
}

func (r *rlpReader) readBytes() ([]byte, error) {
	var header [9]byte
	if _, err := io.ReadFull(r.reader, header[0:1]); err != nil {
		return nil, err
	}
	tag := int(header[0])
	switch {
	case tag < 0x80:
		return []byte{header[0]}, nil
	case tag <= 0xB7:
		buffer := make([]byte, int(header[0])-0x80)
		if err := r.readAll(buffer); err != nil {
			return nil, err
		} else {
			return buffer, nil
		}
	case tag < 0xC0:
		sz := int(header[0]) - 0xB7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return nil, err
		}
		if sz2 > r.maxSB {
			return nil, cerrors.Wrapf(ErrInvalidFormat, "InvalidSize(%d>%d)", sz2, r.maxSB)
		}
		buffer := make([]byte, sz2)
		if err := r.readAll(buffer); err != nil {
			return nil, err
		} else {
			return buffer, nil
		}
	case tag == 0xF8:
		if sz2, err := r.readSize(header[1:2]); err != nil {
			return nil, err
		} else {
			if sz2 == 0 {
				return nil, ErrNilValue
			}
		}
		fallthrough
	default:
		return nil, cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RLPList)")
	}
}

func (r *rlpReader) readUintValue(v reflect.Value) error {
	bs, err := r.readBytes()
	if err != nil {
		return err
	}
	value, ok := intconv.SafeBytesToUint64(bs)
	if !ok {
		return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x)", bs)
	}
	switch v.Kind() {
	case reflect.Bool:
		if value == 0 {
			v.SetBool(false)
		} else if value == 1 {
			v.SetBool(true)
		} else {
			return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x,type=bool)", bs)
		}
		return nil
	case reflect.Uint:
		if value != uint64(uint(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x,type=uint)", bs)
		}
	case reflect.Uint8:
		if value != uint64(uint8(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x,type=uint8)", bs)
		}
	case reflect.Uint16:
		if value != uint64(uint16(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x,type=uint16)", bs)
		}
	case reflect.Uint32:
		if value != uint64(uint32(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "UintOverflow(bs=%#x,type=uint32)", bs)
		}
	}
	v.SetUint(value)
	return nil
}

func (r *rlpReader) readIntValue(v reflect.Value) error {
	bs, err := r.readBytes()
	if err != nil {
		return err
	}
	value, ok := intconv.SafeBytesToInt64(bs)
	if !ok {
		return cerrors.Wrapf(ErrInvalidFormat, "Int64Overflow(bs=%#x)", bs)
	}
	switch v.Kind() {
	case reflect.Int:
		if value != int64(int(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "IntOverflow(bs=%#x,type=int)", bs)
		}
	case reflect.Int8:
		if value != int64(int8(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "IntOverflow(bs=%#x,type=int8)", bs)
		}
	case reflect.Int16:
		if value != int64(int16(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "IntOverflow(bs=%#x,type=int16)", bs)
		}
	case reflect.Int32:
		if value != int64(int32(value)) {
			return cerrors.Wrapf(ErrInvalidFormat, "IntOverflow(bs=%#x,type=int32)", bs)
		}
	}
	v.SetInt(value)
	return nil
}

func (r *rlpReader) ReadValue(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.readUintValue(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.readIntValue(v)
	case reflect.String:
		bs, err := r.readBytes()
		if err != nil {
			return err
		}
		v.SetString(string(bs))
		return nil
	}
	return cerrors.Wrapf(ErrIllegalType, "IllegalType(%s)", v.Type())
}

func (r *rlpReader) Close() error {
	_, err := io.Copy(io.Discard, r.reader)
	return err
}

func (r *rlpReader) ReadBytes() ([]byte, error) {
	return r.readBytes()
}

func (r *rlpReader) readMore(org []byte, size int) ([]byte, error) {
	if size+len(org) > r.maxSB {
		return nil, cerrors.Wrapf(ErrInvalidFormat, "IllegalFormat(%d>%d)", size+len(org), r.maxSB)
	}
	buffer := make([]byte, len(org)+size)
	copy(buffer, org)
	if err := r.readAll(buffer[len(org):]); err != nil {
		return nil, err
	}
	return buffer, nil
}

func (r *rlpReader) ReadRaw() ([]byte, error) {
	var header [9]byte
	if _, err := io.ReadFull(r.reader, header[0:1]); err != nil {
		return nil, err
	}
	tag := int(header[0])
	switch {
	case tag < 0x80:
		return header[0:1], nil
	case tag <= 0xB7:
		size := tag - 0x80
		return r.readMore(header[0:1], size)
	case tag < 0xC0:
		sz := tag - 0xB7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return nil, err
		}
		return r.readMore(header[0:1+sz], sz2)
	case tag <= 0xF7:
		sz := tag - 0xC0
		return r.readMore(header[0:1], sz)
	default:
		sz := tag - 0xF7
		sz2, err := r.readSize(header[1 : 1+sz])
		if err != nil {
			return nil, err
		}
		return r.readMore(header[0:1+sz], sz2)
	}
}

func (r *rlpReader) SetMaxBytes(sz int) {
	r.maxSB = sz
}

type rlpParent struct {
	buffer *bytes.Buffer
	writer *rlpWriter
	isMap  bool
	cnt    int
}

var rlpParentPool = sync.Pool {
	New: func() interface{} {
		return &rlpParent {
			buffer: bytes.NewBuffer(nil),
		}
	},
}

func allocRLPParent(writer *rlpWriter, isMap bool) *rlpParent {
	p := rlpParentPool.Get().(*rlpParent)
	p.writer = writer
	p.isMap = isMap
	return p
}

func freeRLPParent(p *rlpParent) {
	p.cnt = 0
	p.writer = nil
	p.buffer.Reset()
	rlpParentPool.Put(p)
}

type rlpWriter struct {
	parent *rlpParent
	writer io.Writer
}

func (w *rlpWriter) countN(cnt int) {
	if w.parent != nil {
		w.parent.cnt += cnt
	}
}

func (w *rlpWriter) WriteList() (Writer, error) {
	w.countN(1)
	p := allocRLPParent(w, false)
	return &rlpWriter{
		parent: p,
		writer: p.buffer,
	}, nil
}

func (w *rlpWriter) WriteMap() (Writer, error) {
	w.countN(1)
	p := allocRLPParent(w, true)
	return &rlpWriter{
		parent: p,
		writer: p.buffer,
	}, nil
}

func (w *rlpWriter) writeAll(b []byte) error {
	for written := 0; written < len(b); {
		n, err := w.writer.Write(b[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}

var nullSequence = []byte{0xf8, 0}

func (w *rlpWriter) writeNull() error {
	return w.writeAll(nullSequence)
}

func (w *rlpWriter) writeBytes(b []byte) error {
	if b == nil {
		return w.writeNull()
	}
	switch l := len(b); {
	case l == 0:
		return w.writeAll([]byte{0x80})
	case l == 1:
		if b[0] < 0x80 {
			return w.writeAll(b)
		}
		fallthrough
	case l <= 55:
		var header [1]byte
		header[0] = byte(0x80 + l)
		if err := w.writeAll(header[:]); err != nil {
			return err
		}
		return w.writeAll(b)
	default:
		sz := sizeToBytes(l)
		var header [1]byte
		header[0] = byte(0x80 + 55 + len(sz))
		if err := w.writeAll(header[:]); err != nil {
			return err
		}
		if err := w.writeAll(sz); err != nil {
			return err
		}
		return w.writeAll(b)
	}
}

func (w *rlpWriter) writeList(b []byte) error {
	switch l := len(b); {
	case l == 0:
		return w.writeAll([]byte{0xc0})
	case l <= 55:
		var header [1]byte
		header[0] = byte(0xC0 + l)
		if err := w.writeAll(header[:]); err != nil {
			return err
		}
		return w.writeAll(b)
	default:
		sz := sizeToBytes(l)
		var header [1]byte
		header[0] = byte(0xC0 + 55 + len(sz))
		if err := w.writeAll(header[:]); err != nil {
			return err
		}
		if err := w.writeAll(sz); err != nil {
			return err
		}
		return w.writeAll(b)
	}
}

func (w *rlpWriter) WriteBytes(b []byte) error {
	w.countN(1)
	return w.writeBytes(b)
}

func (w *rlpWriter) WriteRaw(b []byte) error {
	w.countN(1)
	return w.writeAll(b)
}

func (w *rlpWriter) WriteValue(v reflect.Value) error {
	w.countN(1)
	switch v.Kind() {
	case reflect.Bool:
		var buffer [1]byte
		if v.Bool() {
			buffer[0] = 1
		} else {
			buffer[0] = 0
		}
		return w.writeBytes(buffer[:])

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return w.writeBytes(intconv.Uint64ToBytes(v.Uint()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return w.writeBytes(intconv.Int64ToBytes(v.Int()))

	case reflect.String:
		return w.writeBytes([]byte(v.String()))

	default:
		return cerrors.Wrapf(ErrIllegalType, "IllegalType(%s)", v.Kind())
	}
}

func (w *rlpWriter) WriteNull() error {
	w.countN(1)
	return w.writeNull()
}

func (w *rlpWriter) Close() error {
	if p := w.parent; p != nil {
		if p.isMap {
			if (p.cnt % 2) != 0 {
				return ErrInvalidFormat
			}
		}
		if err := p.writer.writeList(p.buffer.Bytes()); err != nil {
			return err
		}
		w.parent = nil
		freeRLPParent(p)
	}
	return nil
}

func (c *rlpCodec) Name() string {
	return "rlp"
}

func (c *rlpCodec) NewDecoder(r io.Reader) DecodeAndCloser {
	return NewDecoder(&rlpReader{
		reader: r,
		maxSB:  MaxSizeForBytes,
	})
}

func (c *rlpCodec) NewEncoder(w io.Writer) EncodeAndCloser {
	return NewEncoder(&rlpWriter{
		writer: w,
	})
}
