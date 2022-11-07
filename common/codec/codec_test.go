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
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	for _, co := range []Codec{MP, RLP} {
		t.Run(co.Name(), func(t *testing.T) {
			bs0, err := co.MarshalToBytes(v0)
			assert.NoError(t, err)

			bs1, err := co.MarshalToBytes(v1)
			assert.NoError(t, err)
			assert.Equal(t, bs0, bs1)

			bs2, err := co.MarshalToBytes(v2)
			assert.NoError(t, err)
			assert.Equal(t, bs0, bs2)

			bs3, err := co.MarshalToBytes(v3)
			assert.NoError(t, err)
			assert.NotEqual(t, bs0, bs3)

			bs4, err := co.MarshalToBytes(v4)
			assert.NoError(t, err)
			assert.Equal(t, bs3, bs4)

			js4, err := json.Marshal(v4)
			assert.NoError(t, err)
			fmt.Println(string(js4))

			bs5, err := co.MarshalToBytes(v5)
			assert.NoError(t, err)
			assert.Equal(t, bs3, bs5)

			js5, err := json.Marshal(v5)
			assert.NoError(t, err)
			fmt.Println(string(js5))

			var v0v Struct0
			br0, err := co.UnmarshalFromBytes(bs0, &v0v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br0)
			assert.Equal(t, v0, v0v)

			var v1v Struct1
			br1, err := co.UnmarshalFromBytes(bs1, &v1v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br1)
			assert.Equal(t, v1, v1v)

			var v2v Struct2
			br2, err := co.UnmarshalFromBytes(bs2, &v2v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br2)
			assert.Equal(t, v0v.StringValue, v2v.structPrivate.StringValue)
			assert.Equal(t, v0v.IntValue, v2v.structPrivate.IntValue)
			assert.Equal(t, v0v.MyValue, v2v.MyValue)

			var v3v Struct3
			br3, err := co.UnmarshalFromBytes(bs3, &v3v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br3)
			assert.Equal(t, v0v.StringValue, v3v.Inner.StringValue)
			assert.Equal(t, v0v.IntValue, v3v.Inner.IntValue)
			assert.Equal(t, v0v.MyValue, v3v.MyValue)

			var v4v Struct4
			br4, err := co.UnmarshalFromBytes(bs4, &v4v)
			assert.NoError(t, err)
			assert.Equal(t, []byte{}, br4)
			assert.Equal(t, v4.StructPublic, v4v.StructPublic)
			assert.Equal(t, v4.MyValue, v4v.MyValue)
		})
	}
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
	for _, co := range []Codec{MP, RLP} {
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
	}
}

func TestEmbeddedInterface(t *testing.T) {
	type I interface {
		M()
	}
	type S struct {
		I
	}
	var s S
	bs := MustMarshalToBytes(&s)
	_, err := UnmarshalFromBytes(bs, &s)
	assert.NoError(t, err)
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
