package codec

import (
	"encoding"
	"errors"
	"io"
	"math/big"
	"reflect"
	"sort"

	cerrors "github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
)

type codecImpl interface {
	Name() string
	NewDecoder(r io.Reader) DecodeAndCloser
	NewEncoder(w io.Writer) EncodeAndCloser
}

type Codec interface {
	codecImpl
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
	MarshalToBytes(v interface{}) ([]byte, error)
	UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error)
	MustMarshalToBytes(v interface{}) []byte
	MustUnmarshalFromBytes(b []byte, v interface{}) []byte
	NewEncoderBytes(b *[]byte) EncodeAndCloser
}

func UnmarshalFromBytes(b []byte, v interface{}) ([]byte, error) {
	return BC.UnmarshalFromBytes(b, v)
}

type Encoder interface {
	Encode(o interface{}) error
	EncodeMulti(objs ...interface{}) error
	EncodeList() (Encoder, error)
	EncodeListOf(objs ...interface{}) error
}

type EncodeAndCloser interface {
	Encoder
	Close() error
}

type Decoder interface {
	Skip(cnt int) error
	Decode(o interface{}) error
	DecodeMulti(objs ...interface{}) (int, error)
	DecodeAll(objs ...interface{}) error
	DecodeBytes() ([]byte, error)
	DecodeList() (Decoder, error)
	DecodeListOf(objs ...interface{}) error
	SetMaxBytes(sz int) bool
}

type DecodeAndCloser interface {
	Decoder
	Close() error
}

type EncodeSelfer interface {
	RLPEncodeSelf(e Encoder) error
}

type DecodeSelfer interface {
	RLPDecodeSelf(d Decoder) error
}

type WriteSelfer interface {
	RLPWriteSelf(w Writer) error
}

type ReadSelfer interface {
	RLPReadSelf(w Reader) error
}

type Unmarshaler interface {
	UnmarshalRLP([]byte) error
}

type Marshaler interface {
	MarshalRLP() ([]byte, error)
}

type Selfer interface {
	EncodeSelfer
	DecodeSelfer
}

type Writer interface {
	WriteList() (Writer, error)
	WriteMap() (Writer, error)
	WriteBytes(b []byte) error
	WriteRaw(b []byte) error
	WriteValue(v reflect.Value) error
	WriteNull() error
	Close() error
}

type Reader interface {
	Skip(cnt int) error
	ReadList() (Reader, error)
	ReadMap() (Reader, error)
	ReadBytes() ([]byte, error)
	ReadRaw() ([]byte, error)
	ReadValue(v reflect.Value) error
	Close() error
}

var (
	ErrNilValue      = errors.New("NilValueError")
	ErrInvalidFormat = errors.New("InvalidFormatError")
	ErrIllegalType   = errors.New("IllegalTypeError")
	ErrPanicInCustom = errors.New("PanicInCustomError")
)

type encoderImpl struct {
	real  Writer
	child *encoderImpl
}

func (e *encoderImpl) flush() error {
	if e.child != nil {
		if err := e.child.flushAndClose(); err != nil {
			return err
		}
		e.child = nil
	}
	return nil
}

func (e *encoderImpl) flushAndClose() error {
	if err := e.flush(); err != nil {
		return err
	}
	return e.real.Close()
}

func (e *encoderImpl) Close() error {
	return e.flushAndClose()
}

func (e *encoderImpl) EncodeList() (Encoder, error) {
	if err := e.flush(); err != nil {
		return nil, err
	}
	e2, err := e.encodeList()
	return e2, err
}

func (e *encoderImpl) encodeList() (*encoderImpl, error) {
	writer, err := e.real.WriteList()
	if err != nil {
		return nil, err
	}
	e.child = &encoderImpl{real: writer}
	return e.child, nil
}

func (e *encoderImpl) encodeMap() (*encoderImpl, error) {
	writer, err := e.real.WriteMap()
	if err != nil {
		return nil, err
	}
	e.child = &encoderImpl{real: writer}
	return e.child, nil
}

func (e *encoderImpl) tryCustom(v reflect.Value) (bool, error) {
	if v.CanInterface() {
		switch value := v.Interface().(type) {
		case EncodeSelfer:
			if err := value.RLPEncodeSelf(e); err == nil {
				return true, e.flush()
			} else {
				return true, err
			}
		case WriteSelfer:
			return true, value.RLPWriteSelf(e.real)
		case encoding.BinaryMarshaler:
			b, err := value.MarshalBinary()
			if err != nil {
				return true, err
			}
			return true, e.real.WriteBytes(b)
		case Marshaler:
			b, err := value.MarshalRLP()
			if err != nil {
				return true, err
			}
			return true, e.real.WriteRaw(b)
		case *big.Int:
			if value != nil {
				return true, e.Encode(intconv.BigIntToBytes(value))
			} else {
				return true, e.Encode(nil)
			}
		}
	}
	return false, nil
}

func (e *encoderImpl) encodeArrayValue(v reflect.Value) error {
	switch v.Type().Elem().Kind() {
	case reflect.Uint8:
		if v.Kind() == reflect.Slice || v.CanAddr() {
			return e.real.WriteBytes(v.Bytes())
		} else {
			bs := make([]byte, v.Len())
			reflect.Copy(reflect.ValueOf(bs), v)
			return e.real.WriteBytes(bs)
		}
	default:
		e2, err := e.encodeList()
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

type encodeFunc func(e *encoderImpl, v reflect.Value) error

func (e *encoderImpl) encodeNullable(v reflect.Value, encode encodeFunc) error {
	if v.IsNil() {
		return e.real.WriteNull()
	}
	return encode(e, v)
}

func encodeRecursiveFields(e *encoderImpl, v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		return nil
	}
	vt := v.Type()
	n := v.NumField()
	for i := 0; i < n; i++ {
		fv := v.Field(i)
		ft := vt.Field(i)
		if ft.Anonymous && ft.Type.Kind() == reflect.Interface {
			continue
		}
		if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
			if err := encodeRecursiveFields(e, fv); err != nil {
				return err
			}
			continue
		}
		if !fv.CanInterface() {
			continue
		}
		if err := e.encodeValue(fv); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoderImpl) encodeValue(v reflect.Value) error {
	if v.CanAddr() {
		if ok, err := e.tryCustom(v.Addr()); ok {
			return err
		}
	}
	switch v.Kind() {
	case reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.String:
		return e.real.WriteValue(v)
	case reflect.Slice:
		return e.encodeNullable(v, func(e *encoderImpl, v reflect.Value) error {
			return e.encodeArrayValue(v)
		})
	case reflect.Array:
		return e.encodeArrayValue(v)
	case reflect.Ptr:
		return e.encodeNullable(v, func(e *encoderImpl, v reflect.Value) error {
			return e.encodeValue(v.Elem())
		})
	case reflect.Struct:
		e2, err := e.encodeList()
		if err != nil {
			return err
		}
		if err := encodeRecursiveFields(e2, v); err != nil {
			return err
		}
		return e.flush()

	case reflect.Map:
		return e.encodeNullable(v, func(e *encoderImpl, v reflect.Value) error {
			e2, err := e.encodeMap()
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
		return e.encodeNullable(v, func(e *encoderImpl, v reflect.Value) error {
			return e.encodeValue(v.Elem())
		})

	default:
		if !v.IsValid() || v.IsNil() {
			return e.real.WriteNull()
		}
		return cerrors.Wrapf(ErrIllegalType, "IllegalType(%s)", v.Kind())
	}
}

func (e *encoderImpl) Encode(o interface{}) error {
	if err := e.flush(); err != nil {
		return err
	}
	switch o := o.(type) {
	case []byte:
		if o == nil {
			return e.real.WriteNull()
		}
		return e.real.WriteBytes(o)
	case reflect.Value:
		return e.encodeValue(o)
	default:
		return e.encodeValue(reflect.ValueOf(o))
	}
}

func (e *encoderImpl) EncodeNullable(o interface{}) error {
	return e.encodeValue(reflect.ValueOf(o))
}

func (e *encoderImpl) EncodeMulti(objs ...interface{}) error {
	for _, obj := range objs {
		if err := e.Encode(obj); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoderImpl) EncodeListOf(objs ...interface{}) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(objs...); err != nil {
		return err
	}
	return e.flush()
}

type decoderImpl struct {
	real  Reader
	child Reader
}

func (d *decoderImpl) decodeList() (*decoderImpl, error) {
	reader, err := d.real.ReadList()
	if err != nil {
		return nil, err
	}
	d.child = reader
	return &decoderImpl{real: reader}, nil
}

func (d *decoderImpl) decodeMap() (*decoderImpl, error) {
	reader, err := d.real.ReadMap()
	if err != nil {
		return nil, err
	}
	d.child = reader
	return &decoderImpl{real: reader}, nil
}

func (d *decoderImpl) flush() error {
	if child := d.child; child != nil {
		d.child = nil
		return child.Close()
	}
	return nil
}

func (d *decoderImpl) Close() error {
	if err := d.flush(); err != nil {
		return err
	}
	return d.real.Close()
}

func (d *decoderImpl) Skip(n int) error {
	if err := d.flush(); err != nil {
		return err
	}
	return d.real.Skip(n)
}

func (d *decoderImpl) Decode(o interface{}) error {
	if err := d.flush(); err != nil {
		return err
	}
	return d.decode(o)
}

func (d *decoderImpl) DecodeBytes() ([]byte, error) {
	if err := d.flush(); err != nil {
		return nil, err
	}
	return d.real.ReadBytes()
}

func (d *decoderImpl) DecodeList() (Decoder, error) {
	if err := d.flush(); err != nil {
		return nil, err
	}
	d2, err := d.decodeList()
	return d2, err
}

func (d *decoderImpl) DecodeMulti(objs ...interface{}) (int, error) {
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

func (d *decoderImpl) DecodeAll(objs ...interface{}) error {
	if _, err := d.DecodeMulti(objs...); err != nil {
		return ErrInvalidFormat
	} else {
		return nil
	}
}

func (d *decoderImpl) DecodeListOf(objs ...interface{}) error {
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

func (d *decoderImpl) SetMaxBytes(sz int) bool {
	type setMaxByteser interface {
		SetMaxBytes(sz int)
	}
	if ri, ok := d.real.(setMaxByteser); ok {
		ri.SetMaxBytes(sz)
		return true
	}
	return false
}

func (d *decoderImpl) tryCustom(v reflect.Value) (consume bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			consume = true
			err = cerrors.Wrapf(ErrPanicInCustom, "panic in custom decoder: %v", r)
		}
	}()
	if v.CanInterface() {
		switch value := v.Interface().(type) {
		case DecodeSelfer:
			if err := value.RLPDecodeSelf(d); err == nil {
				return true, d.flush()
			} else {
				return true, err
			}
		case ReadSelfer:
			return true, value.RLPReadSelf(d.real)
		case encoding.BinaryUnmarshaler:
			b, err := d.real.ReadBytes()
			if err != nil {
				return true, err
			}
			return true, value.UnmarshalBinary(b)
		case Unmarshaler:
			b, err := d.real.ReadRaw()
			if err != nil {
				return true, err
			}
			return true, value.UnmarshalRLP(b)
		case *big.Int:
			b, err := d.real.ReadBytes()
			if err != nil {
				return true, err
			}
			if v := intconv.BigIntSetBytes(value, b); v != nil {
				return true, nil
			} else {
				return true, ErrInvalidFormat
			}
		}
	}
	return false, nil
}

func (d *decoderImpl) decodeNullableValue(v reflect.Value) error {
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

func decodeRecursiveFields(d *decoderImpl, elem reflect.Value) error {
	if elem.Kind() != reflect.Struct {
		return nil
	}
	et := elem.Type()
	n := elem.NumField()
	for i := 0; i < n; i++ {
		fv := elem.Field(i)
		ft := et.Field(i)
		if ft.Anonymous && ft.Type.Kind() == reflect.Interface {
			continue
		}
		if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
			if err := decodeRecursiveFields(d, fv); err != nil {
				return err
			}
			continue
		}
		if !fv.CanSet() {
			continue
		}
		if err := d.decodeValue(fv.Addr()); err != nil {
			if err == io.EOF {
				fv.Set(reflect.Zero(fv.Type()))
				continue
			}
			return err
		}
	}
	return nil
}

func (d *decoderImpl) decodeValue(v reflect.Value) error {
	if v.Kind() != reflect.Ptr {
		return cerrors.Wrap(ErrIllegalType, "IllegalType(NotPointer)")
	}

	if ok, err := d.tryCustom(v); ok {
		return err
	}

	elem := v.Elem()

	switch elem.Kind() {
	case reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.String:
		return d.real.ReadValue(elem)

	case reflect.Array:
		switch elem.Type().Elem().Kind() {
		case reflect.Uint8:
			bs, err := d.real.ReadBytes()
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
			bs, err := d.real.ReadBytes()
			if err != nil && err != ErrNilValue {
				return err
			}
			elem.SetBytes(bs)
			return nil
		default:
			d2, err := d.decodeList()
			if err != nil {
				if err == ErrNilValue {
					elem.Set(reflect.Zero(elem.Type()))
					return nil
				}
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
			if err == ErrNilValue {
				v2 = reflect.Zero(elem.Type())
			} else {
				return err
			}
		}
		elem.Set(v2)
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
		d2, err := d.decodeMap()
		if err != nil {
			if err == ErrNilValue {
				elem.Set(reflect.Zero(elem.Type()))
				return nil
			}
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

func (d *decoderImpl) decodeNullable(o interface{}) error {
	switch o := o.(type) {
	case *[]byte:
		if b, err := d.real.ReadBytes(); err == ErrNilValue {
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

func (d *decoderImpl) decode(o interface{}) error {
	switch o := o.(type) {
	case *[]byte:
		if b, err := d.real.ReadBytes(); err == ErrNilValue {
			*o = nil
			return nil
		} else if err == nil {
			*o = b
			return nil
		} else {
			return err
		}
	case reflect.Value:
		return d.decodeValue(o)
	default:
		return d.decodeValue(reflect.ValueOf(o))
	}
}

func NewEncoder(w Writer) EncodeAndCloser {
	return &encoderImpl{real: w}
}

func NewDecoder(r Reader) DecodeAndCloser {
	return &decoderImpl{real: r}
}
