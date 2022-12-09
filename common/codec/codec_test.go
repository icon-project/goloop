/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package codec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

var codecsToTest = []Codec{RLP, MP}

type StructPublic struct {
	IntValue    int64  `json:"int"`
	StringValue string `json:"str"`
}

type structPrivate struct {
	IntValue    int64  `json:"int"`
	StringValue string `json:"str"`
}

type Struct0 struct {
	IntValue    int64  `json:"int"`
	StringValue string `json:"str"`
	MyValue     int64  `json:"my"`
}

type Struct1 struct {
	StructPublic
	MyValue int64 `json:"my"`
}

type Struct2 struct {
	structPrivate
	hidden  *StructPublic
	hidden2 structPrivate
	MyValue int64
}

type Struct3 struct {
	Inner   *StructPublic `json:"inner"`
	MyValue int64         `json:"my"`
}

type Struct4 struct {
	*StructPublic
	hidden  *StructPublic `json:"hidden"`
	hidden2 structPrivate `json:"hidden2"`
	MyValue int64         `json:"my"`
}

type Struct5 struct {
	Visible *structPrivate `json:"inner"`
	MyValue int64          `json:"my"`
}

type limitWriter struct {
	writer io.Writer
	offset int
	limit  int
}

func (l *limitWriter) Write(p []byte) (n int, err error) {
	avail := l.limit - l.offset
	var d []byte
	var pErr error
	if len(p) <= avail {
		d = p
	} else {
		d = p[:avail]
		pErr = errors.New("fail to write")
	}
	n, err = l.writer.Write(d)
	l.offset += n
	if err == nil {
		err = pErr
	}
	return
}

func LimitWriter(w io.Writer, limit int) io.Writer {
	return &limitWriter{
		writer: w,
		limit:  limit,
	}
}

func TestHandleWriterError(t *testing.T) {
	RunWithCodecs(t, func(t *testing.T, c Codec) {
		type Inner struct {
			SliceInt64 []int64
			DictString map[string]string
			Bytes      []byte
		}
		type Unsigned struct {
			Uint8  uint8
			Uint16 uint16
			Uint32 uint32
			Uint   uint
			Uint64 uint64
		}
		type Struct1 struct {
			Bool       bool
			Int8       int8
			Int16      int16
			Int32      int32
			Int        int
			Inner      Inner
			ArrayUint8 [4]uint8
			Unsigned   Unsigned
		}
		obj := Struct1{
			Bool:  true,
			Int8:  0x33,
			Int16: -0x2233,
			Int32: 0x7b8300cd,
			Int:   0x88392934,
			Inner: Inner{
				SliceInt64: []int64{0x12345678abcdef90, 0x1234567890abcdef},
				DictString: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Bytes: []byte("this is test for long data.this is test for long data.this is test for long data.this is test for long data.this is test for long data.this is test for long data."),
			},
			ArrayUint8: [4]uint8{0, 4, 2, 1},
			Unsigned: Unsigned{
				Uint8:  0x33,
				Uint16: 0x2233,
				Uint32: 0x7b8300cd,
				Uint:   0x88392934,
				Uint64: 0x012345678abcdef9,
			},
		}
		bs, err := c.MarshalToBytes(obj)
		assert.NoError(t, err)

		var rObj Struct1
		br, err := c.UnmarshalFromBytes(bs, &rObj)
		assert.NoError(t, err)
		assert.Empty(t, br)
		assert.Equal(t, obj, rObj)

		for i := len(bs) - 1; i > 0; i-- {
			w := LimitWriter(io.Discard, i)
			err := c.Marshal(w, obj)
			assert.Error(t, err, fmt.Sprintf("unexpected no error size=%d len=%d", i, len(bs)))
		}
	})
}

func RunWithCodecs(t *testing.T, f func(t *testing.T, c Codec)) {
	for _, c := range codecsToTest {
		t.Run(c.Name(), func(t *testing.T) {
			f(t, c)
		})
	}
}

func TestDecoder_Struct1(t *testing.T) {
	v0 := Struct0{
		1, "Value1", 3,
	}

	v1 := Struct1{
		StructPublic{1, "Value1"},
		3,
	}

	v2 := Struct2{
		structPrivate{1, "Value1"},
		&StructPublic{2, "Value2"},
		structPrivate{4, "Value4"},
		3,
	}

	v3 := Struct3{
		&StructPublic{1, "Value1"},
		3,
	}

	v4 := Struct4{
		&StructPublic{1, "Value1"},
		&StructPublic{2, "Value2"},
		structPrivate{4, "Value4"},
		3,
	}

	v5 := Struct5{
		&structPrivate{1, "Value1"},
		3,
	}

	type Struct6 struct {
		Inner struct {
			IntValue    int64
			StringValue string
			ExtraField  [3]int
		}
		MyValue    int64
		ExtraField int
	}

	t.Run("HandleEmbedded", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs0, err := co.MarshalToBytes(v0)
			assert.NoError(t, err)
			bs1, err := co.MarshalToBytes(v1)
			assert.NoError(t, err)

			assert.Equal(t, bs0, bs1)
		})
	})
	t.Run("IgnoreHiddenFields", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs1, err := co.MarshalToBytes(v1)
			assert.NoError(t, err)
			bs2, err := co.MarshalToBytes(v2)
			assert.NoError(t, err)

			assert.Equal(t, bs1, bs2)
		})
	})
	t.Run("DifferentEmbeddedAndField", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs1, err := co.MarshalToBytes(v1)
			assert.NoError(t, err)
			bs3, err := co.MarshalToBytes(v3)
			assert.NoError(t, err)

			assert.NotEqual(t, bs1, bs3)
		})

	})
	t.Run("IgnoreTypeIsPrivate", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs3, err := co.MarshalToBytes(v3)
			assert.NoError(t, err)
			bs5, err := co.MarshalToBytes(v5)
			assert.NoError(t, err)

			assert.Equal(t, bs3, bs5)
		})
	})
	t.Run("StructureRecover w/o Private", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs0, err := co.MarshalToBytes(v0)
			assert.NoError(t, err)
			var v0v Struct0
			br0, err := co.UnmarshalFromBytes(bs0, &v0v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br0)
			assert.Equal(t, v0, v0v)

			bs1, err := co.MarshalToBytes(v1)
			assert.NoError(t, err)
			var v1v Struct1
			br1, err := co.UnmarshalFromBytes(bs1, &v1v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br1)
			assert.Equal(t, v1, v1v)

			bs3, err := co.MarshalToBytes(v3)
			assert.NoError(t, err)
			var v3v Struct3
			br3, err := co.UnmarshalFromBytes(bs3, &v3v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br3)
			assert.Equal(t, v3, v3v)
		})
	})
	t.Run("StructureRecover w/ Privates", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs2, err := co.MarshalToBytes(v2)
			assert.NoError(t, err)
			var v2v Struct2
			br2, err := co.UnmarshalFromBytes(bs2, &v2v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br2)
			assert.NotEqual(t, v2, v2v)
			assert.Equal(t, v2.structPrivate, v2v.structPrivate)
			assert.Equal(t, v2.MyValue, v2v.MyValue)

			bs4, err := co.MarshalToBytes(v4)
			assert.NoError(t, err)
			var v4v Struct4
			br4, err := co.UnmarshalFromBytes(bs4, &v4v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br4)
			assert.NotEqual(t, v4, v4v)
			assert.Equal(t, v4.StructPublic, v4v.StructPublic)
			assert.Equal(t, v4.MyValue, v4v.MyValue)
		})
	})
	t.Run("StructureRecover Partial", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, co Codec) {
			bs3, err := co.MarshalToBytes(v3)
			assert.NoError(t, err)
			var v6 Struct6
			br6, err := co.UnmarshalFromBytes(bs3, &v6)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br6)
			assert.Equal(t, v3.Inner.IntValue, v6.Inner.IntValue)
			assert.Equal(t, v3.Inner.StringValue, v6.Inner.StringValue)
			assert.Equal(t, v3.MyValue, v6.MyValue)
			assert.Zero(t, v6.Inner.ExtraField)
			assert.Zero(t, v6.ExtraField)
		})
	})
}

type nullableStruct1 struct {
	V1 *string
	V2 *big.Int
}

type nullableStruct2 struct {
	V1 *string
	V2 *big.Int
	V3 *string
	V4 *big.Int
	V5 int
	V6 string
}

func TestNilValueEncoding(t *testing.T) {
	testString := "TEST"
	RunWithCodecs(t, func(t *testing.T, co Codec) {
		s1 := &nullableStruct1{
			V1: nil,
			V2: nil,
		}
		s2 := &nullableStruct1{
			V1: &testString,
			V2: big.NewInt(2),
		}
		s3 := &nullableStruct2{
			V1: &testString,
			V2: big.NewInt(2),
			V3: &testString,
			V4: big.NewInt(3),
			V5: 4,
			V6: "TEST",
		}
		bs, err := co.MarshalToBytes(s1)
		assert.NoError(t, err)

		t.Logf("Encoded:%#x", bs)

		exb, err := co.UnmarshalFromBytes(bs, s2)
		assert.NoError(t, err)
		assert.Len(t, exb, 0, "Should be empty")
		assert.Equal(t, s1, s2)

		exb, err = co.UnmarshalFromBytes(bs, s3)
		assert.NoError(t, err)
		assert.Len(t, exb, 0, "Should be empty")
		assert.Equal(t, s1.V1, s3.V1)
		assert.Equal(t, s1.V2, s3.V2)
		assert.Nil(t, s3.V3)
		assert.Nil(t, s3.V4)
		assert.Equal(t, 0, s3.V5)
		assert.Equal(t, "", s3.V6)
	})
}

func TestEmbeddedInterface(t *testing.T) {
	type I interface {
		M()
	}
	type S struct {
		I
	}
	RunWithCodecs(t, func(t *testing.T, c Codec) {
		var s S
		bs := c.MustMarshalToBytes(&s)
		_, err := c.UnmarshalFromBytes(bs, &s)
		assert.NoError(t, err)
	})
}

func FuzzUnmarshalFromBytes(f *testing.F) {
	type Test struct {
		FInt       int
		FBytes     []byte
		FPtrString *string
		FList      []string
		FString    string
	}
	f.Add([]byte{0x02})
	f.Add([]byte("Hellow this\xc0"))
	f.Fuzz(func(t *testing.T, bs []byte) {
		var buf Test
		_, _ = UnmarshalFromBytes(bs, &buf)
		var v4v Struct4
		_, _ = UnmarshalFromBytes(bs, &v4v)
	})
}

func TestReader_Skip(t *testing.T) {
	RunWithCodecs(t, func(t *testing.T, c Codec) {
		assert := assert.New(t)

		var bs []byte
		ec := c.NewEncoderBytes(&bs)
		err := ec.Encode([]interface{}{
			int64(3),
			int64(2088),
			"test data for larger data over 56 test data for larger data over 56",
			[]interface{}{
				128, 2098, 8899,
			},
			int64(999),
		})
		assert.NoError(err)
		err = ec.Close()
		assert.NoError(err)

		dc := c.NewDecoder(bytes.NewBuffer(bs))
		items, err := dc.DecodeList()
		assert.NoError(err)

		var v1 int64
		err = items.Skip(4)
		assert.NoError(err)
		err = items.Decode(&v1)
		assert.NoError(err)
		assert.Equal(int64(999), v1)

		// ensure skip failure
		err = items.Skip(1)
		assert.Error(err)

		err = dc.Close()
		assert.NoError(err)

		dc = c.NewDecoder(bytes.NewBuffer(bs))
		items, err = dc.DecodeList()
		assert.NoError(err)

		err = items.Skip(3)
		assert.NoError(err)
		subList, err := items.DecodeList()
		assert.NoError(err)
		err = subList.Skip(1)
		assert.NoError(err)

		var v2 int64
		err = subList.Decode(&v2)
		assert.NoError(err)
		assert.EqualValues(int64(2098), v2)

		// read value after list
		err = items.Decode(&v1)
		assert.NoError(err)
		assert.Equal(int64(999), v1)
	})
}

func TestIntValues(t *testing.T) {
	type intValues struct {
		Bool  bool
		Int8  int8
		Int16 int16
		Int32 int32
		Int64 int64
		Int   int
	}
	type uintValues struct {
		Bool   bool
		Uint8  uint8
		Uint16 uint16
		Uint32 uint32
		Uint64 uint64
		Uint   uint
	}
	positiveValues := intValues{
		true, 83, 0x102, 0x304890, 0x4bad39408d, 0x3b8d98,
	}
	negativeValues := intValues{
		false, -83, -0x102, -0x304890, -0x4bad39408d, -0x3b8d98,
	}
	unsignedValues := uintValues{
		true, 83, 0x102, 0x304890, 0x4bad39408d, 0x3b8d98,
	}
	RunWithCodecs(t, func(t *testing.T, c Codec) {
		assert := assert.New(t)

		pvBytes := c.MustMarshalToBytes(positiveValues)
		nvBytes := c.MustMarshalToBytes(negativeValues)
		uvBytes := c.MustMarshalToBytes(unsignedValues)

		var iv1 intValues
		_, err := c.UnmarshalFromBytes(pvBytes, &iv1)
		assert.NoError(err)
		assert.Equal(positiveValues, iv1)

		var iv2 intValues
		_, err = c.UnmarshalFromBytes(nvBytes, &iv2)
		assert.NoError(err)
		assert.Equal(negativeValues, iv2)

		var uv1 uintValues
		_, err = c.UnmarshalFromBytes(uvBytes, &uv1)
		assert.NoError(err)
		assert.Equal(unsignedValues, uv1)

		var uv2 uintValues
		_, err = c.UnmarshalFromBytes(nvBytes, &uv2)
		if err == nil {
			t.Logf("Read Negative to Unsigned: SUCCESS\nUnsigned:%+v\nOriginal:%+v", uv2, negativeValues)
		} else {
			t.Logf("Read Negative to Unsigned: FAILs with error:%v", err)
		}
	})
}

func TestMap(t *testing.T) {
	t.Run("signed", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, c Codec) {
			m1 := map[int]string{
				1:  "value 1",
				2:  "value 2",
				-2: "value -2",
			}
			bs, err := c.MarshalToBytes(m1)
			assert.NoError(t, err)
			var m1v map[int]string
			br, err := c.UnmarshalFromBytes(bs, &m1v)
			assert.NoError(t, err)
			assert.Empty(t, br)
			assert.EqualValues(t, m1, m1v)
		})
	})
	t.Run("unsigned", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, c Codec) {
			m2 := map[uint]string{
				1: "value 1",
				2: "value 2",
			}
			bs, err := c.MarshalToBytes(m2)
			assert.NoError(t, err)
			var m2v map[uint]string
			br, err := c.UnmarshalFromBytes(bs, &m2v)
			assert.NoError(t, err)
			assert.Empty(t, br)
			assert.EqualValues(t, m2, m2v)
		})
	})
	t.Run("invalid", func(t *testing.T) {
		RunWithCodecs(t, func(t *testing.T, c Codec) {
			k1 := "key1"
			k2 := "key2"
			m3 := map[*string]string{
				&k1: "value1",
				&k2: "value2",
			}
			bs, err := c.MarshalToBytes(m3)
			assert.Error(t, err)
			assert.Nil(t, bs)
		})
	})
}
