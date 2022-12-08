package codec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/errors"
)

var codecsForTyped = []Codec{BC, MP}

type testTypeCodecType struct{}

type testValue struct {
	value []byte
}

func (t *testTypeCodecType) Decode(tag uint8, data []byte) (interface{}, error) {
	switch tag {
	case TypeCustom:
		return &testValue{
			value: data,
		}, nil
	default:
		return nil, errors.Errorf("UnknownTypeTag(tag=%d)", tag)
	}
}

func (t *testTypeCodecType) Encode(o interface{}) (uint8, []byte, error) {
	switch obj := o.(type) {
	case *testValue:
		return TypeCustom, obj.value, nil
	}
	return 0, nil, errors.New("UnknownType")
}

var testTypeCodec = &testTypeCodecType{}

func TestEncodeAny(t *testing.T) {
	assert := assert.New(t)
	value1 := map[string]interface{}{
		"nil":    nil,
		"string": "hello",
		"bytes":  []byte("foo"),
		"bool":   true,
		"slice": []interface{}{
			"slice0",
			false,
			newTypedObj(TypeBytes, []byte("slice2")),
		},
		"typedObj":  newTypedObj(TypeString, "TEST"),
		"typedList": []*TypedObj{newTypedObj(TypeBool, TrueBytes)},
		"typedDict": &TypedDict{
			Map: map[string]*TypedObj{
				"v1": newTypedObj(TypeString, "TestValue1"),
				"v2": newTypedObj(TypeString, "TestValue2"),
			},
			Keys: []string{"v2", "v1"},
		},
		"mapTypedObj": map[string]*TypedObj{
			"x1": newTypedObj(TypeString, "x1"),
		},
		"customObj": &testValue{value: []byte("customValue123")},
	}
	expected1 := newTypedObj(TypeDict, &TypedDict{
		Map: map[string]*TypedObj{
			"nil":    newTypedObj(TypeNil, nil),
			"string": newTypedObj(TypeString, "hello"),
			"bytes":  newTypedObj(TypeBytes, []byte("foo")),
			"bool":   newTypedObj(TypeBool, TrueBytes),
			"slice": newTypedObj(TypeList, []*TypedObj{
				newTypedObj(TypeString, "slice0"),
				newTypedObj(TypeBool, FalseBytes),
				newTypedObj(TypeBytes, []byte("slice2")),
			}),
			"typedObj": newTypedObj(TypeString, "TEST"),
			"typedList": newTypedObj(TypeList, []*TypedObj{
				newTypedObj(TypeBool, TrueBytes),
			}),
			"typedDict": newTypedObj(TypeDict, &TypedDict{
				Map: map[string]*TypedObj{
					"v1": newTypedObj(TypeString, "TestValue1"),
					"v2": newTypedObj(TypeString, "TestValue2"),
				},
				Keys: []string{"v2", "v1"},
			}),
			"mapTypedObj": newTypedObj(TypeDict, &TypedDict{
				Map: map[string]*TypedObj{
					"x1": newTypedObj(TypeString, "x1"),
				},
			}),
			"customObj": newTypedObj(TypeCustom, []byte("customValue123")),
		},
	})
	value2, err := EncodeAny(testTypeCodec, value1)
	assert.NoError(err)
	assert.EqualValues(expected1, value2)

	expectedObj := newTypedObj(TypeDict, &TypedDict{
		Map: map[string]*TypedObj{
			"nil":    newTypedObj(TypeNil, nil),
			"string": newTypedObj(TypeString, "hello"),
			"bytes":  newTypedObj(TypeBytes, []byte("foo")),
			"bool":   newTypedObj(TypeBool, TrueBytes),
			"slice": newTypedObj(TypeList, []*TypedObj{
				newTypedObj(TypeString, "slice0"),
				newTypedObj(TypeBool, FalseBytes),
				newTypedObj(TypeBytes, []byte("slice2")),
			}),
			"typedObj": newTypedObj(TypeString, "TEST"),
			"typedList": newTypedObj(TypeList, []*TypedObj{
				newTypedObj(TypeBool, TrueBytes),
			}),
			"typedDict": newTypedObj(TypeDict, &TypedDict{
				Map: map[string]*TypedObj{
					"v1": newTypedObj(TypeString, "TestValue1"),
					"v2": newTypedObj(TypeString, "TestValue2"),
				},
				Keys: []string{"v2", "v1"},
			}),
			"mapTypedObj": newTypedObj(TypeDict, &TypedDict{
				Map: map[string]*TypedObj{
					"x1": newTypedObj(TypeString, "x1"),
				},
				Keys: []string{"x1"},
			}),
			"customObj": newTypedObj(TypeCustom, []byte("customValue123")),
		},
		Keys: []string{"bool", "bytes", "customObj", "mapTypedObj", "nil", "slice", "string", "typedDict", "typedList", "typedObj"},
	})

	expectRaw := map[string]interface{}{
		"nil":    nil,
		"string": "hello",
		"bytes":  []byte("foo"),
		"bool":   true,
		"slice": []interface{}{
			"slice0",
			false,
			[]byte("slice2"),
		},
		"typedObj":  "TEST",
		"typedList": []interface{}{true},
		"typedDict": map[string]interface{}{
			"v1": "TestValue1",
			"v2": "TestValue2",
		},
		"mapTypedObj": map[string]interface{}{
			"x1": "x1",
		},
		"customObj": &testValue{value: []byte("customValue123")},
	}

	for _, c := range codecsForTyped {
		// marshal values to the bytes
		bs, err := MarshalAny(c, testTypeCodec, value2)
		assert.NoError(err)

		// check Unmarshal recovers correct object tree
		// some of dictionary should have order after serialization.
		var obj *TypedObj
		_, err = c.UnmarshalFromBytes(bs, &obj)
		assert.NoError(err)
		assert.EqualValues(expectedObj, obj)

		// check DecodeAny recovers it as raw types
		raw, err := DecodeAny(testTypeCodec, obj)
		assert.NoError(err)
		assert.EqualValues(expectRaw, raw)

		for l := len(bs) - 1; l > 0; l-- {
			var obj2 *TypedObj
			_, err := c.UnmarshalFromBytes(bs[:l], &obj2)
			assert.Error(err, fmt.Sprintf("should fail on size=%d len=%d", l, len(bs)))
		}
	}
}

func TestErrorHandling(t *testing.T) {
	obj := newTypedObj(TypeList, []*TypedObj{
		newTypedObj(TypeDict, &TypedDict{
			Map: map[string]*TypedObj{
				"boolValue": newTypedObj(TypeBool, []byte{2}),
			},
		}),
	})
	for _, c := range codecsForTyped {
		bs, err := c.MarshalToBytes(obj)
		assert.NoError(t, err)
		_, err = UnmarshalAny(c, testTypeCodec, bs)
		assert.Error(t, err)
	}
}
