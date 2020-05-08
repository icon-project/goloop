package rlp

import (
	"bytes"
	"encoding"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"sort"

	cerrors "github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
)

type Encoder interface {
	Encode(o interface{}) error
	EncodeMulti(objs ...interface{}) error
	EncodeList() (Encoder, error)
	EncodeListOf(objs ...interface{}) error
}

type Decoder interface {
	Decode(o interface{}) error
	DecodeMulti(objs ...interface{}) (int, error)
	DecodeBytes() ([]byte, error)
	DecodeList() (Decoder, error)
	DecodeListOf(objs ...interface{}) error
}

type EncodeSelfer interface {
	RLPEncodeSelf(e Encoder) error
}

type DecodeSelfer interface {
	RLPDecodeSelf(d Decoder) error
}

type Selfer interface {
	EncodeSelfer
	DecodeSelfer
}

var (
	ErrNilValue      = errors.New("NilValueError")
	ErrInvalidFormat = errors.New("InvalidFormatError")
	ErrIllegalType   = errors.New("IllegalTypeError")
)

type rlpEncoder struct {
	writer           io.Writer
	containerBuffer  *bytes.Buffer
	containerEncoder *rlpEncoder
}

func (e *rlpEncoder) flush() error {
	if e.containerEncoder != nil {
		if err := e.containerEncoder.flush(); err != nil {
			return err
		}
		if err := e.writeList(e.containerBuffer.Bytes()); err != nil {
			return err
		}
		e.containerEncoder = nil
		e.containerBuffer = nil
	}
	return nil
}

func (e *rlpEncoder) EncodeList() (Encoder, error) {
	e2, err := e.newContainerEncoder()
	return e2, err
}

func (e *rlpEncoder) newContainerEncoder() (*rlpEncoder, error) {
	if err := e.flush(); err != nil {
		return nil, err
	}
	e.containerBuffer = new(bytes.Buffer)
	e.containerEncoder = &rlpEncoder{writer: e.containerBuffer}
	return e.containerEncoder, nil
}

var rlpEncodeSelferType = reflect.TypeOf((*EncodeSelfer)(nil)).Elem()
var binaryMarshaler = reflect.TypeOf((*encoding.BinaryMarshaler)(nil)).Elem()

func (e *rlpEncoder) tryCustom(v reflect.Value) (bool, error) {
	if v.Type().Implements(rlpEncodeSelferType) {
		if i, ok := v.Interface().(EncodeSelfer); ok {
			if err := i.RLPEncodeSelf(e); err == nil {
				return true, e.flush()
			} else {
				return true, err
			}
		}
	}
	if v.Type().Implements(binaryMarshaler) {
		if i, ok := v.Interface().(encoding.BinaryMarshaler); ok {
			b, err := i.MarshalBinary()
			if err != nil {
				return true, err
			}
			return true, e.encodeBytes(b)
		}
	}
	return false, nil
}

func (e *rlpEncoder) encodeList(b []byte) error {
	if err := e.flush(); err != nil {
		return err
	}
	return e.writeList(b)
}

func (e *rlpEncoder) encodeArrayValue(v reflect.Value) error {
	switch v.Type().Elem().Kind() {
	case reflect.Uint8:
		if v.Kind() == reflect.Slice || v.CanAddr() {
			return e.encodeBytes(v.Bytes())
		} else {
			bs := make([]byte, v.Len())
			reflect.Copy(reflect.ValueOf(bs), v)
			return e.encodeBytes(bs)
		}
	default:
		e2, err := e.newContainerEncoder()
		if err != nil {
			return err
		}
		n := v.Len()
		for i := 0; i < n; i++ {
			if err := e2.encodeValue(v.Index(i)); err != nil {
				return err
			}
		}
		return e.flush()
	}
}

type encodeFunc func(e *rlpEncoder, v reflect.Value) error

func (e *rlpEncoder) encodeNullable(v reflect.Value, encode encodeFunc) error {
	if v.IsNil() {
		return e.writeNull()
	}
	return encode(e, v)
}

func encodeRecursiveFields(e *rlpEncoder, v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		return nil
	}
	n := v.NumField()
	for i := 0; i < n; i++ {
		fv := v.Field(i)
		if !fv.CanInterface() {
			if err := encodeRecursiveFields(e, fv); err != nil {
				return err
			}
			continue
		}
		if err := e.encodeValue(fv); err != nil {
			return err
		}
	}
	return nil
}

func (e *rlpEncoder) encodeValue(v reflect.Value) error {
	if v.CanAddr() {
		if ok, err := e.tryCustom(v.Addr()); ok {
			return err
		}
	}
	switch v.Kind() {
	case reflect.Bool:
		var buffer [1]byte
		if v.Bool() {
			buffer[0] = 1
		} else {
			buffer[0] = 0
		}
		return e.encodeBytes(buffer[:])

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.encodeBytes(intconv.Uint64ToBytes(v.Uint()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.encodeBytes(intconv.Int64ToBytes(v.Int()))

	case reflect.Slice:
		return e.encodeNullable(v, func(e *rlpEncoder, v reflect.Value) error {
			return e.encodeArrayValue(v)
		})
	case reflect.Array:
		return e.encodeArrayValue(v)

	case reflect.String:
		return e.encodeBytes([]byte(v.String()))

	case reflect.Ptr:
		return e.encodeNullable(v, func(e *rlpEncoder, v reflect.Value) error {
			return e.encodeValue(v.Elem())
		})
	case reflect.Struct:
		e2, err := e.newContainerEncoder()
		if err != nil {
			return err
		}
		if err := encodeRecursiveFields(e2, v); err != nil {
			return err
		}
		return e.flush()

	case reflect.Map:
		return e.encodeNullable(v, func(e *rlpEncoder, v reflect.Value) error {
			e2, err := e.newContainerEncoder()
			if err != nil {
				return err
			}
			keys := v.MapKeys()
			switch v.Type().Key().Kind() {
			case reflect.String:
				sort.Slice(keys, func(i, j int) bool {
					return keys[i].String() < keys[j].String()
				})
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sort.Slice(keys, func(i, j int) bool {
					return keys[i].Int() < keys[j].Int()
				})
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				sort.Slice(keys, func(i, j int) bool {
					return keys[i].Uint() < keys[j].Uint()
				})
			default:
				return cerrors.Wrapf(ErrIllegalType, "IllegalType(key=%s)", v.Type().Key())
			}
			n := len(keys)
			for i := 0; i < n; i++ {
				if err := e2.encodeValue(keys[i]); err != nil {
					return err
				}
				v2 := v.MapIndex(keys[i])
				if err := e2.encodeValue(v2); err != nil {
					return err
				}
			}
			return e.flush()
		})

	case reflect.Interface:
		return e.encodeNullable(v, func(e *rlpEncoder, v reflect.Value) error {
			return e.encodeValue(v.Elem())
		})

	default:
		if !v.IsValid() || v.IsNil() {
			return e.writeNull()
		}
		return cerrors.Wrapf(ErrIllegalType, "IllegalType(%s)", v.Kind())
	}
}

func (e *rlpEncoder) Encode(o interface{}) error {
	switch o := o.(type) {
	case []byte:
		if o == nil {
			return e.writeNull()
		}
		return e.encodeBytes(o)
	case reflect.Value:
		return e.encodeValue(o)
	default:
		return e.encodeValue(reflect.ValueOf(o))
	}
}

func (e *rlpEncoder) EncodeNullable(o interface{}) error {
	return e.encodeValue(reflect.ValueOf(o))
}

func (e *rlpEncoder) EncodeMulti(objs ...interface{}) error {
	for _, obj := range objs {
		if err := e.Encode(obj); err != nil {
			return err
		}
	}
	return nil
}

func (e *rlpEncoder) EncodeListOf(objs ...interface{}) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(objs...); err != nil {
		return err
	}
	return e.flush()
}

func (e *rlpEncoder) writeAll(b []byte) error {
	for written := 0; written < len(b); {
		n, err := e.writer.Write(b[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}

func sizeToBytes(s int) []byte {
	return intconv.SizeToBytes(uint64(s))
}

func bytesToSize(bs []byte) int {
	return int(intconv.BytesToSize(bs))
}

func (e *rlpEncoder) encodeBytes(b []byte) error {
	if b == nil {
		return e.writeNull()
	}
	if err := e.flush(); err != nil {
		return err
	}
	switch l := len(b); {
	case l == 0:
		return e.writeAll([]byte{0x80})
	case l == 1:
		if b[0] < 0x80 {
			return e.writeAll(b)
		}
		fallthrough
	case l <= 55:
		var header [1]byte
		header[0] = byte(0x80 + l)
		if err := e.writeAll(header[:]); err != nil {
			return err
		}
		return e.writeAll(b)
	default:
		sz := sizeToBytes(l)
		var header [1]byte
		header[0] = byte(0x80 + 55 + len(sz))
		if err := e.writeAll(header[:]); err != nil {
			return err
		}
		if err := e.writeAll(sz); err != nil {
			return err
		}
		return e.writeAll(b)
	}
}

func (e *rlpEncoder) writeList(b []byte) error {
	switch l := len(b); {
	case l == 0:
		return e.writeAll([]byte{0xc0})
	case l <= 55:
		var header [1]byte
		header[0] = byte(0xC0 + l)
		if err := e.writeAll(header[:]); err != nil {
			return err
		}
		return e.writeAll(b)
	default:
		sz := sizeToBytes(l)
		var header [1]byte
		header[0] = byte(0xC0 + 55 + len(sz))
		if err := e.writeAll(header[:]); err != nil {
			return err
		}
		if err := e.writeAll(sz); err != nil {
			return err
		}
		return e.writeAll(b)
	}
}

var nullSequence = []byte{0xf8, 0}

func (e *rlpEncoder) writeNull() error {
	return e.writeAll(nullSequence)
}

type rlpDecoder struct {
	reader          io.Reader
	containerReader io.Reader
}

func (d *rlpDecoder) readBytes() ([]byte, error) {
	var header [9]byte
	if _, err := io.ReadFull(d.reader, header[0:1]); err != nil {
		return nil, err
	}
	tag := int(header[0])
	switch {
	case tag < 0x80:
		return []byte{header[0]}, nil
	case tag <= 0xB7:
		buffer := make([]byte, int(header[0])-0x80)
		if _, err := io.ReadFull(d.reader, buffer); err != nil {
			if err == io.EOF {
				return nil, cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(err=%s)", err)
			}
			return nil, cerrors.WithStack(err)
		} else {
			return buffer, nil
		}
	case tag < 0xC0:
		sz := int(header[0]) - 0xB7
		if _, err := io.ReadFull(d.reader, header[1:1+sz]); err != nil {
			if err == io.EOF {
				return nil, cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(sz=%d,err=%s)", sz, err)
			}
			return nil, cerrors.WithStack(err)
		}

		blen := bytesToSize(header[1 : 1+sz])
		buffer := make([]byte, blen)
		if _, err := io.ReadFull(d.reader, buffer); err != nil {
			if err == io.EOF {
				return nil, cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(blen=%d)", blen)
			}
			return nil, cerrors.WithStack(err)
		} else {
			return buffer, nil
		}
	case tag == 0xF8:
		if _, err := io.ReadFull(d.reader, header[1:2]); err != nil {
			if err == io.EOF {
				return nil, cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RLPList)")
			}
			return nil, cerrors.WithStack(err)
		}
		if header[1] == 0 {
			return nil, ErrNilValue
		}
		fallthrough
	default:
		return nil, cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RLPList)")
	}
}

func (d *rlpDecoder) readList() (io.Reader, error) {
	reader := d.reader

	var header [9]byte
	if _, err := io.ReadFull(reader, header[0:1]); err != nil {
		return nil, err
	}
	tag := int(header[0])
	switch {
	case tag < 0xC0:
		return nil, cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RLPBytes)")
	case tag <= 0xF7:
		size := int64(tag - 0xC0)
		return io.LimitReader(reader, size), nil
	default:
		sz := tag - 0xF7
		if _, err := io.ReadFull(reader, header[1:1+sz]); err != nil {
			if err == io.EOF {
				return nil, cerrors.Wrapf(ErrInvalidFormat, "InvalidFormat(sz=%d)", sz)
			}
			return nil, cerrors.WithStack(err)
		}
		if sz == 1 && header[1] == 0 {
			return nil, ErrNilValue
		}
		blen := bytesToSize(header[1 : 1+sz])
		return io.LimitReader(d.reader, int64(blen)), nil
	}
}

func (d *rlpDecoder) decodeList() (*rlpDecoder, error) {
	reader, err := d.readList()
	if err != nil {
		return nil, err
	}
	d.containerReader = reader
	return &rlpDecoder{reader: reader}, nil
}

func (d *rlpDecoder) flush() error {
	if d.containerReader != nil {
		if _, err := ioutil.ReadAll(d.containerReader); err != nil {
			return err
		}
		d.containerReader = nil
	}
	return nil
}

func (d *rlpDecoder) Decode(o interface{}) error {
	if err := d.flush(); err != nil {
		return err
	}
	return d.decode(o)
}

func (d *rlpDecoder) DecodeBytes() ([]byte, error) {
	if err := d.flush(); err != nil {
		return nil, err
	}
	return d.readBytes()
}

func (d *rlpDecoder) DecodeList() (Decoder, error) {
	if err := d.flush(); err != nil {
		return nil, err
	}
	d2, err := d.decodeList()
	return d2, err
}

func (d *rlpDecoder) DecodeMulti(objs ...interface{}) (int, error) {
	if err := d.flush(); err != nil {
		return 0, err
	}
	for idx, obj := range objs {
		if err := d.decodeNullable(obj); err != nil {
			return idx, err
		}
	}
	return len(objs), nil
}

func (d *rlpDecoder) DecodeListOf(objs ...interface{}) error {
	if err := d.flush(); err != nil {
		return err
	}
	d2, err := d.decodeList()
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err := d2.decodeNullable(obj); err != nil {
			return ErrInvalidFormat
		}
	}
	return d.flush()
}

var rlpDecodeSelferType = reflect.TypeOf((*DecodeSelfer)(nil)).Elem()

var binaryUnmarshaler = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()

func (d *rlpDecoder) tryCustom(v reflect.Value) (bool, error) {
	if v.Type().Implements(rlpDecodeSelferType) {
		if err := v.Interface().(DecodeSelfer).RLPDecodeSelf(d); err == nil {
			return true, d.flush()
		} else {
			return true, err
		}
	}
	if v.Type().Implements(binaryUnmarshaler) {
		u := v.Interface().(encoding.BinaryUnmarshaler)
		b, err := d.readBytes()
		if err != nil {
			return true, err
		}
		return true, u.UnmarshalBinary(b)
	}
	return false, nil
}

func (d *rlpDecoder) decodeNullableValue(v reflect.Value) error {
	if v.Kind() != reflect.Ptr {
		return cerrors.Wrap(ErrIllegalType, "IllegalType(NotPointer)")
	}
	if err := d.decodeValue(v); err == ErrNilValue {
		elem := v.Elem()
		elem.Set(reflect.Zero(elem.Type()))
		return nil
	} else {
		return err
	}
}

func decodeRecursiveFields(d *rlpDecoder, elem reflect.Value) error {
	if elem.Kind() != reflect.Struct {
		return nil
	}
	n := elem.NumField()
	for i := 0; i < n; i++ {
		fv := elem.Field(i)
		if !fv.CanSet() {
			if err := decodeRecursiveFields(d, fv); err != nil {
				return err
			}
			continue
		}
		if err := d.decodeNullableValue(fv.Addr()); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (d *rlpDecoder) decodeValue(v reflect.Value) error {
	if v.Kind() != reflect.Ptr {
		return cerrors.Wrap(ErrIllegalType, "IllegalType(NotPointer)")
	}

	if ok, err := d.tryCustom(v); ok {
		return err
	}

	elem := v.Elem()

	switch elem.Kind() {
	case reflect.Bool:
		bs, err := d.readBytes()
		if err != nil {
			return err
		}
		elem.SetBool(intconv.BytesToUint64(bs) != 0)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bs, err := d.readBytes()
		if err != nil {
			return err
		}
		elem.SetUint(intconv.BytesToUint64(bs))
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bs, err := d.readBytes()
		if err != nil {
			return err
		}
		elem.SetInt(intconv.BytesToInt64(bs))
		return nil

	case reflect.Array:
		switch elem.Type().Elem().Kind() {
		case reflect.Uint8:
			bs, err := d.readBytes()
			if err != nil {
				return err
			}
			reflect.Copy(elem.Slice(0, elem.Len()), reflect.ValueOf(bs))
			return nil

		default:
			d2, err := d.decodeList()
			if err != nil {
				return err
			}
			for i := 0; i < elem.Len(); i++ {
				err := d2.decodeNullableValue(elem.Index(i).Addr())
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
			}
			return d.flush()
		}

	case reflect.Slice:
		switch elem.Type().Elem().Kind() {
		case reflect.Uint8:
			bs, err := d.readBytes()
			if err != nil {
				return err
			}
			elem.SetBytes(bs)
			return nil
		default:
			d2, err := d.decodeList()
			if err != nil {
				return err
			}
			elem.Set(reflect.MakeSlice(elem.Type(), 0, 16))
			for i := 0; true; i++ {
				if elem.Cap() < i+1 {
					ns := reflect.MakeSlice(elem.Type(), elem.Len(), elem.Cap()+16)
					reflect.Copy(ns, elem)
					elem.Set(ns)
				}
				elem.SetLen(i + 1)
				err := d2.decodeNullableValue(elem.Index(i).Addr())
				if err != nil {
					if err == io.EOF {
						elem.SetLen(i)
						break
					}
					return err
				}
			}
			return d.flush()
		}

	case reflect.Ptr:
		v2 := reflect.New(elem.Type().Elem())
		if err := d.decodeValue(v2); err != nil {
			return err
		}
		elem.Set(v2)
		return nil

	case reflect.String:
		bs, err := d.readBytes()
		if err != nil {
			return err
		}
		elem.SetString(string(bs))
		return nil

	case reflect.Struct:
		d2, err := d.decodeList()
		if err != nil {
			return err
		}
		if err := decodeRecursiveFields(d2, elem); err != nil {
			return err
		}
		return d.flush()

	case reflect.Map:
		d2, err := d.decodeList()
		if err != nil {
			return err
		}
		m := reflect.MakeMap(elem.Type())
		for {
			key := reflect.New(elem.Type().Key())
			if err := d2.decodeValue(key); err != nil {
				if err == io.EOF {
					break
				}
				if err == ErrNilValue {
					return cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(NilKey)")
				}
				return err
			}
			value := reflect.New(elem.Type().Elem())
			if err := d2.decodeNullableValue(value); err != nil {
				if err == io.EOF {
					return cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(NoValue)")
				}
				return err
			}
			m.SetMapIndex(key.Elem(), value.Elem())
		}
		elem.Set(m)
		return d.flush()

	default:
		return cerrors.Wrapf(ErrIllegalType, "IllegalType(%s)", elem.Type())
	}
}

func (d *rlpDecoder) decodeNullable(o interface{}) error {
	switch o := o.(type) {
	case *[]byte:
		if b, err := d.readBytes(); err == ErrNilValue {
			*o = nil
			return nil
		} else if err == nil {
			*o = b
			return nil
		} else {
			return err
		}
	case reflect.Value:
		return d.decodeNullableValue(o)
	default:
		return d.decodeNullableValue(reflect.ValueOf(o))
	}
}

func (d *rlpDecoder) decode(o interface{}) error {
	switch o := o.(type) {
	case *[]byte:
		if b, err := d.readBytes(); err != nil {
			return err
		} else {
			*o = b
			return nil
		}
	case reflect.Value:
		return d.decodeValue(o)
	default:
		return d.decodeValue(reflect.ValueOf(o))
	}
}

func NewEncoder(w io.Writer) Encoder {
	return &rlpEncoder{writer: w}
}

func NewDecoder(r io.Reader) Decoder {
	return &rlpDecoder{reader: r}
}

func Marshal(obj interface{}) ([]byte, error) {
	bs := bytes.NewBuffer(nil)
	enc := NewEncoder(bs)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return bs.Bytes(), nil
}

func Unmarshal(bs []byte, obj interface{}) error {
	buf := bytes.NewBuffer(bs)
	dec := NewDecoder(buf)
	if err := dec.Decode(obj); err != nil {
		return err
	}
	if buf.Len() > 0 {
		return cerrors.Wrap(ErrInvalidFormat, "InvalidFormat(RemainingBytes)")
	}
	return nil
}
