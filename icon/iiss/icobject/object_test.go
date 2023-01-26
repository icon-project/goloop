package icobject

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
)

func bytesFactory(tag Tag) (Impl, error) {
	switch tag.Type() {
	case TypeBytes:
		return BytesImpl{}, nil
	default:
		return nil, errors.IllegalArgumentError.Errorf("UnknownTypeTag(tag=%#x)", tag)
	}
}

func TestObject_Equal(t *testing.T) {
	o0 := NewBytesObject([]byte{0, 1, 2, 3})
	o1 := NewBytesObject([]byte{0, 1, 2, 3})
	o2 := NewBytesObject([]byte{1, 2, 3, 4})

	assert.True(t, o0.Equal(o0))
	assert.True(t, o0.Equal(o1))
	assert.True(t, o1.Equal(o0))

	assert.False(t, o0.Equal(o2))
	assert.False(t, o2.Equal(o0))
	assert.False(t, o1.Equal(o2))
	assert.False(t, o2.Equal(o1))
}

func TestObject_BytesValue(t *testing.T) {
	bs := []byte{0, 1, 2, 3}
	o := NewBytesObject(bs)
	assert.Equal(t, []byte{0, 1, 2, 3}, o.BytesValue())
}

func TestObject_RLPDecodeSelf(t *testing.T) {
	bs := []byte{0, 1, 2, 3}
	o := NewBytesObject(bs)

	buf := bytes.NewBuffer(nil)
	e := codec.BC.NewEncoder(buf)

	err := o.RLPEncodeSelf(e)
	assert.NoError(t, err)

	err = e.Close()
	assert.NoError(t, err)

	d := codec.BC.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	o2 := &Object{tag: o.Tag()}
	err = o2.RLPDecodeSelf(d, bytesFactory)
	assert.NoError(t, err)

	assert.True(t, o.Equal(o2))
	assert.Equal(t, o.BytesValue(), o2.BytesValue())
	assert.Equal(t, bs, o2.BytesValue())
}
