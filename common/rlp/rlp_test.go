package rlp

import (
	"encoding/json"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructEncodeDecode(t *testing.T) {
	type myType struct {
		IntValue    int
		StringValue string
		BytesValue  []byte
		IntSlice    []int
	}
	type myTypeArray struct {
		IntValue    int
		StringValue string
		ByteArray   [1]byte
		IntArray    [3]int
	}

	v1 := myType{
		IntValue:    3,
		StringValue: "test",
		BytesValue:  []byte{0x11, 0x22},
		IntSlice:    []int{0x22, 0x6fe3},
	}
	bs, err := Marshal(v1)
	if err != nil {
		t.Errorf("Fail to marshal custom structure err=%+v", err)
		return
	}

	var v2 myType
	if err := Unmarshal(bs, &v2); err != nil {
		t.Errorf("Fail to unmarshal custom structure err=%+v", err)
		return
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Errorf("Decoded value isn't same exp=%+v ret=%+v", v1, v2)
	}

	var v3 myTypeArray
	if err := Unmarshal(bs, &v3); err != nil {
		t.Errorf("Fail to unmarshal custom structure err=%+v", err)
		return
	}
	if v3.ByteArray[0] != v1.BytesValue[0] ||
		v3.IntArray[0] != v1.IntSlice[0] ||
		v3.IntArray[1] != v1.IntSlice[1] ||
		v3.IntArray[2] != 0 {
		t.Error("Decoded values are not same")
		return
	}
}

func Test_uint64ToBytes(t *testing.T) {
	type args struct {
		v uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"1byte", args{0xf0}, []byte{0xf0}},
		{"2byte", args{0x0189}, []byte{0x01, 0x89}},
		{"3byte", args{0xff0189}, []byte{0xff, 0x01, 0x89}},
		{"8byte", args{0xff018945dd4a9c44}, []byte{0xff, 0x01, 0x89, 0x45, 0xdd, 0x4a, 0x9c, 0x44}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uint64ToBytes(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uint64ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_int64ToBytes(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"byte1", args{-1}, []byte{0x81}},
		{"byte2", args{-0x80}, []byte{0x80, 0x80}},
		{"byte8", args{-0x7fffffffffffffff},
			[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int64ToBytes(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("int64ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

type MyType struct {
	name string
	age  int
}

func (o *MyType) RLPEncodeSelf(e Encoder) error {
	return e.EncodeListOf(o.name, o.age)
}

func (o *MyType) RLPDecodeSelf(d Decoder) error {
	return d.DecodeListOf(&o.name, &o.age)
}

func TestCustomObject1(t *testing.T) {
	a := []MyType{
		{"lion", 2},
	}
	var b []MyType

	bs, err := Marshal(a)
	if err != nil {
		t.Errorf("Fail to marshal object on the buffer err=%+v", err)
		return
	}

	log.Printf("Encoded:% X", bs)

	if err := Unmarshal(bs, &b); err != nil {
		t.Errorf("Fail to unmarshal object err=%+v", err)
		return
	}

	if !reflect.DeepEqual(a, b) {
		t.Errorf("Decoded value isnt' same as original")
		return
	}
}

func TestCustomObject2(t *testing.T) {
	a := []*MyType{
		{"lion", 2},
		nil,
	}
	var b []*MyType

	bs, err := Marshal(a)
	if err != nil {
		t.Errorf("Fail to marshal object on the buffer err=%+v", err)
		return
	}

	log.Printf("Encoded: % X", bs)

	if err := Unmarshal(bs, &b); err != nil {
		t.Errorf("Fail to unmarshal object err=%+v", err)
		return
	}

	assert.Equal(t, a, b)
}

func TestMapObject1(t *testing.T) {
	mo := map[string]string{
		"test2": "value2",
		"test1": "value1",
		"test3": "value3",
	}
	bs, err := Marshal(mo)
	assert.NoError(t, err)

	log.Printf("Encoded: % X", bs)
	for k, v := range mo {
		log.Printf("key=% X  value=% X", []byte(k), []byte(v))
	}

	var mo2 map[string]string
	err = Unmarshal(bs, &mo2)
	assert.NoError(t, err)
	assert.Equal(t, mo, mo2)
}

func TestJSON_MapDecoding(t *testing.T) {
	var mo map[string]string
	mo = make(map[string]string)
	if err := json.Unmarshal([]byte("{ \"a\": \"a\" }"), &mo); err != nil {
		t.Errorf("Fail to unmarshal err=%+v", err)
	}
	if v, ok := mo["a"]; ok {
		assert.Equal(t, "a", v)
	} else {
		t.Error("There is no value for \"a\"")
	}
}

func Test_Nil_Test(t *testing.T) {
	var nullBytes []byte = nil

	bs1, err := Marshal(nil)
	assert.NoError(t, err)
	bs2, err := Marshal(nullBytes)
	assert.Equal(t, bs1, bs2)
}

type StructHavingPointer struct {
	A, B *MyType
}

func Test_Nil_Custom(t *testing.T) {
	a := StructHavingPointer{
		A: &MyType{
			name: "Test",
			age:  2,
		},
		B: nil,
	}
	var b StructHavingPointer
	bs, err := Marshal(a)
	assert.NoError(t, err)
	assert.NotNil(t, bs)

	t.Logf("Encoded: % X", bs)

	err = Unmarshal(bs, &b)
	assert.NoError(t, err)
	assert.Equal(t, a, b)
}
