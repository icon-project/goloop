package codec

import (
	"bytes"
	"encoding"
	"io"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/pkg/errors"
)

var rlpCodecObject rlpCodec
var RLP = bytesWrapper{&rlpCodecObject}

type rlpCodec struct{}

type RLPEncoder interface {
	io.Writer
	Encode(o interface{}) error
	EncodeContainer() (RLPEncoder, error)
}

type RLPDecoder interface {
	io.Reader
	Decode(o interface{}) error
	DecodeContainer() (RLPDecoder, error)
}

type RLPSelfer interface {
	RLPEncodeSelf(e RLPEncoder) error
	RLPDecodeSelf(d RLPDecoder) error
}

var selferType = reflect.TypeOf((*RLPSelfer)(nil)).Elem()

type rlpEncoder struct {
	io.Writer
	containerBuffer  *bytes.Buffer
	containerEncoder *rlpEncoder
}

func (e *rlpEncoder) flush() error {
	if e.containerEncoder != nil {
		if err := e.containerEncoder.flush(); err != nil {
			return err
		}
		if err := e.writeContainer(e.containerBuffer.Bytes()); err != nil {
			return err
		}
		e.containerEncoder = nil
		e.containerBuffer = nil
	}
	return nil
}

func (e *rlpEncoder) EncodeContainer() (RLPEncoder, error) {
	e, err := e.newContainerEncoder()
	return e, err
}

func (e *rlpEncoder) newContainerEncoder() (*rlpEncoder, error) {
	if err := e.flush(); err != nil {
		return nil, err
	}
	e.containerBuffer = new(bytes.Buffer)
	e.containerEncoder = &rlpEncoder{Writer: e.containerBuffer}
	return e.containerEncoder, nil
}

var binaryMarshaler = reflect.TypeOf((*encoding.BinaryMarshaler)(nil)).Elem()

func (e *rlpEncoder) tryCustom(v reflect.Value) (bool, error) {
	if v.Type().Implements(selferType) {
		if v.IsNil() {
			return true, nil
		}
		if i, ok := v.Interface().(RLPSelfer); ok {
			// log.Printf("Calling RLPSelfer.Encode for %+v", v.Type())
			return true, i.RLPEncodeSelf(e)
		}
	}
	if v.Type().Implements(binaryMarshaler) {
		if v.IsNil() {
			return true, nil
		}
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

func (e *rlpEncoder) encodeContainer(b []byte) error {
	if err := e.flush(); err != nil {
		return err
	}
	return e.writeContainer(b)
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
		return e.encodeBytes(uint64ToBytes(v.Uint()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.encodeBytes(int64ToBytes(v.Int()))

	case reflect.Slice, reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			return e.encodeBytes(v.Bytes())
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

	case reflect.String:
		return e.encodeBytes([]byte(v.String()))

	case reflect.Ptr:
		if v.IsNil() {
			return e.encodeContainer(nil)
		}
		e2, err := e.newContainerEncoder()
		if err != nil {
			return err
		}
		err = e2.encodeValue(v.Elem())
		if err != nil {
			return err
		}
		return e.flush()

	case reflect.Struct:
		log.Printf("Struct %+v", v.Type())
		e2, err := e.newContainerEncoder()
		if err != nil {
			return err
		}
		n := v.NumField()
		for i := 0; i < n; i++ {
			fv := v.Field(i)
			err := e2.encodeValue(fv)
			if err != nil {
				return err
			}
		}
		return e.flush()

	case reflect.Map:
		e2, err := e.newContainerEncoder()
		if err != nil {
			return err
		}
		keys := v.MapKeys()
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

	default:
		return errors.New("IllegalParameter")
	}
}

func (e *rlpEncoder) Encode(o interface{}) error {
	switch o := o.(type) {
	case []byte:
		return e.encodeBytes(o)
	case RLPSelfer:
		return o.RLPEncodeSelf(e)
	case reflect.Value:
		return e.encodeValue(o)
	default:
		return e.encodeValue(reflect.ValueOf(o))
	}
}

func (e *rlpEncoder) writeAll(b []byte) error {
	for written := 0; written < len(b); {
		n, err := e.Write(b[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}

func sizeToBytes(s int) []byte {
	return uint64ToBytes(uint64(s))
}

func bytesTosize(b []byte) int {
	return int(bytesToUint64(b))
}

func uint64ToBytes(v uint64) []byte {
	var buf [8]byte
	var idx int
	for idx = len(buf) - 1; idx >= 0; idx-- {
		buf[idx] = byte(v & 0xff)
		v = v >> 8
		if v == 0 {
			return buf[idx:]
		}
	}
	return buf[:]
}

func int64ToBytes(v int64) []byte {
	if v >= 0 {
		return uint64ToBytes(uint64(v))
	}
	v = -v
	var buf [8]byte
	var idx int
	for idx = len(buf) - 1; idx >= 0; idx-- {
		buf[idx] = byte(v & 0xff)
		if (v & 0x80) != 0 {
			v = v >> 8
		} else {
			v = v >> 8
			if v == 0 {
				buf[idx] |= 0x80
				return buf[idx:]
			}
		}
	}
	buf[0] |= 0x80
	return buf[:]
}

func bytesToInt64(bs []byte) int64 {
	negative := false
	var value int64
	for i := 0; i < len(bs); i++ {
		b := bs[i]
		if i == 0 {
			if (b & 0x80) != 0 {
				b &= 0x7f
				negative = true
			}
		}
		value = (value << 8) + int64(b)
	}
	if negative {
		value = -value
	}
	return value
}

func bytesToUint64(bs []byte) uint64 {
	var value uint64
	for i := 0; i < len(bs); i++ {
		value = (value << 8) + uint64(bs[i])
	}
	return value
}

func (e *rlpEncoder) encodeBytes(b []byte) error {
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

func (e *rlpEncoder) writeContainer(b []byte) error {
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

type rlpDecoder struct {
	io.Reader
	containerReader  io.Reader
	containerDecoder *rlpDecoder
}

func (d *rlpDecoder) decodeBytes() ([]byte, error) {
	var header [9]byte
	if _, err := io.ReadFull(d.Reader, header[0:1]); err != nil {
		return nil, err
	}
	switch {
	case header[0] < 0x80:
		return header[0:1], nil
	case header[0] >= 0xC0:
		return nil, errors.New("IllegalDataDecoding")
	case header[0] < (0x80 + 055):
		buffer := make([]byte, int(header[0])-0x80)
		if _, err := io.ReadFull(d.Reader, buffer); err != nil {
			return nil, err
		} else {
			return buffer, nil
		}
	default:
		sz := int(header[0]) - (0x80 + 55)
		if _, err := io.ReadFull(d.Reader, header[1:1+sz]); err != nil {
			return nil, err
		}

		blen := bytesTosize(header[1 : 1+sz])
		buffer := make([]byte, blen)
		if _, err := io.ReadFull(d.Reader, buffer); err != nil {
			return nil, err
		} else {
			return buffer, nil
		}
	}
}

func (d *rlpDecoder) decodeContainer() (io.Reader, error) {
	reader := d.Reader

	var header [9]byte
	if _, err := io.ReadFull(reader, header[0:1]); err != nil {
		return nil, err
	}
	switch {
	case header[0] < 0xC0:
		return nil, errors.New("IllegalDataDecoding")
	case header[0] < (0xC0 + 055):
		return io.LimitReader(reader, int64(header[0])-0xC0), nil
	default:
		sz := int(header[0]) - (0xC0 + 55)
		if _, err := io.ReadFull(reader, header[1:1+sz]); err != nil {
			return nil, err
		}

		blen := bytesToUint64(header[1 : 1+sz])
		return io.LimitReader(d.Reader, int64(blen)), nil
	}
}

func (d *rlpDecoder) flush() error {
	if d.containerDecoder != nil {
		if err := d.containerDecoder.flush(); err != nil {
			return err
		}
		if _, err := ioutil.ReadAll(d.containerReader); err != nil {
			return err
		}
		d.containerDecoder = nil
		d.containerReader = nil
	}
	return nil
}

func (d *rlpDecoder) DecodeContainer() (RLPDecoder, error) {
	d2, err := d.newContainerDecoder()
	return d2, err
}

func (d *rlpDecoder) newContainerDecoder() (*rlpDecoder, error) {
	if err := d.flush(); err != nil {
		return nil, err
	}
	reader, err := d.decodeContainer()
	if err != nil {
		return nil, err
	}
	d.containerReader = reader
	return &rlpDecoder{Reader: reader}, nil
}

var binaryUnmarshaler = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()

func (d *rlpDecoder) tryCustom(v reflect.Value) (bool, error) {
	if v.Type().Implements(selferType) {
		if self, ok := v.Interface().(RLPSelfer); ok {
			return true, self.RLPDecodeSelf(d)
		}
	}
	if v.Type().Implements(binaryUnmarshaler) {
		if unmarshaler, ok := v.Interface().(encoding.BinaryUnmarshaler); ok {
			b, err := d.decodeBytes()
			if err != nil {
				return true, err
			}
			if err := unmarshaler.UnmarshalBinary(b); err != nil {
				return true, err
			}
		}
	}
	return false, nil
}

func (d *rlpDecoder) decodeValue(v reflect.Value) error {
	if v.Kind() != reflect.Ptr {
		return errors.New("ReadOnlyParameterUsedForDecode")
	}

	if ok, err := d.tryCustom(v); ok {
		return err
	}

	elem := v.Elem()

	switch elem.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bs, err := d.decodeBytes()
		if err != nil {
			return err
		}
		elem.SetUint(bytesToUint64(bs))
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bs, err := d.decodeBytes()
		if err != nil {
			return err
		}
		elem.SetInt(bytesToInt64(bs))
		return nil

	case reflect.Array:
		switch elem.Type().Elem().Kind() {
		case reflect.Uint8:
			bs, err := d.decodeBytes()
			if err != nil {
				return err
			}
			reflect.Copy(elem.Slice(0, elem.Len()), reflect.ValueOf(bs))
			return nil

		default:
			d2, err := d.newContainerDecoder()
			if err != nil {
				return err
			}
			for i := 0; i < elem.Len(); i++ {
				err := d2.decodeValue(elem.Index(i).Addr())
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
			}
			return nil
		}

	case reflect.Slice:
		switch elem.Type().Elem().Kind() {
		case reflect.Uint8:
			bs, err := d.decodeBytes()
			if err != nil {
				return err
			}
			elem.SetBytes(bs)
			return nil
		default:
			d2, err := d.newContainerDecoder()
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
				err := d2.decodeValue(elem.Index(i).Addr())
				if err != nil {
					if err != io.EOF {
						elem.Set(reflect.Zero(elem.Type()))
						return err
					}
					elem.SetLen(i)
					break
				}
			}
			return nil
		}

	case reflect.Ptr:
		d2, err := d.newContainerDecoder()
		if err != nil {
			return err
		}
		v2 := reflect.New(elem.Type().Elem())
		err = d2.decodeValue(v2)
		if err != nil {
			if err == io.EOF {
				elem.Set(reflect.Zero(elem.Type()))
				return d.flush()
			}
			return err
		}
		elem.Set(v2)
		return d.flush()

	case reflect.String:
		bs, err := d.decodeBytes()
		if err != nil {
			return err
		}
		elem.SetString(string(bs))
		return nil

	case reflect.Struct:
		d2, err := d.newContainerDecoder()
		if err != nil {
			return err
		}
		n := elem.NumField()
		for i := 0; i < n; i++ {
			fv := elem.Field(i)
			if !fv.CanSet() {
				continue
			}
			err := d2.decodeValue(fv.Addr())
			if err != nil {
				if err != io.EOF {
					return err
				}
				break
			}
		}
		return d.flush()

	}
	log.Printf("Unknown data object type:%+v", elem.Type())
	return errors.New("UnknownType")
}

func (d *rlpDecoder) Decode(o interface{}) error {
	switch o := o.(type) {
	case RLPSelfer:
		tmp := new(bytes.Buffer)
		r := io.TeeReader(d.Reader, tmp)
		if err := o.RLPDecodeSelf(&rlpDecoder{Reader: r}); err != nil {
			d.Reader = io.MultiReader(tmp, d.Reader)
			return err
		}
		return nil
	case *[]byte:
		if b, err := d.decodeBytes(); err != nil {
			return err
		} else {
			*o = b
		}
		return nil
	case reflect.Value:
		return d.decodeValue(o)
	default:
		return d.decodeValue(reflect.ValueOf(o))
	}
}

func (*rlpCodec) Marshal(w io.Writer, o interface{}) error {
	e := rlpEncoder{Writer: w}
	if err := e.Encode(o); err != nil {
		return err
	}
	return e.flush()
}

func (*rlpCodec) Unmarshal(r io.Reader, o interface{}) error {
	d := rlpDecoder{Reader: r}
	return d.Decode(o)
}
