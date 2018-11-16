package codec

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
)

func Test_rlpDecoder_decodeContainer(t *testing.T) {
	type fields struct {
		Reader           io.Reader
		containerReader  io.Reader
		containerDecoder *rlpDecoder
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "NormalCase1",
			fields: fields{
				Reader: bytes.NewBuffer([]byte{0xC5, 0x76, 0x54, 0x32, 0x10, 0x90, 0x80}),
			},
			want:    []byte{0x76, 0x54, 0x32, 0x10, 0x90},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &rlpDecoder{
				Reader:           tt.fields.Reader,
				containerReader:  tt.fields.containerReader,
				containerDecoder: tt.fields.containerDecoder,
			}
			got, err := e.decodeContainer()
			if (err != nil) != tt.wantErr {
				t.Errorf("rlpDecoder.decodeContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			bs, _ := ioutil.ReadAll(got)
			if !bytes.Equal(bs, tt.want) {
				t.Errorf("rlpDecoder.decodeContainer() = %x, want %x", bs, tt.want)
				return
			}
		})
	}
}

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

	buf := bytes.NewBuffer(nil)

	v1 := myType{
		IntValue:    3,
		StringValue: "test",
		BytesValue:  []byte{0x11, 0x22},
		IntSlice:    []int{0x22, 0x6fe3},
	}
	if err := RLP.Marshal(buf, v1); err != nil {
		t.Errorf("Fail to marshal custom structure err=%+v", err)
		return
	}

	buf2 := bytes.NewBuffer(buf.Bytes())

	var v2 myType
	if err := RLP.Unmarshal(buf, &v2); err != nil {
		t.Errorf("Fail to unmarshal custom structure err=%+v", err)
		return
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Errorf("Decoded value isn't same exp=%+v ret=%+v", v1, v2)
	}

	var v3 myTypeArray
	if err := RLP.Unmarshal(buf2, &v3); err != nil {
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

func (o *MyType) RLPEncodeSelf(e RLPEncoder) error {
	if err := e.Encode(o.name); err != nil {
		return err
	}
	if err := e.Encode(o.age); err != nil {
		return err
	}
	return nil
}

func (o *MyType) RLPDecodeSelf(d RLPDecoder) error {
	if err := d.Decode(&o.name); err != nil {
		return err
	}
	if err := d.Decode(&o.age); err != nil {
		return err
	}
	return nil
}

func TestCustomObject1(t *testing.T) {
	a := []MyType{
		{"lion", 2},
	}
	var b []MyType

	buf := bytes.NewBuffer(nil)
	if err := RLP.Marshal(buf, a); err != nil {
		t.Errorf("Fail to marshal object on the buffer err=%+v", err)
		return
	}

	log.Printf("Buffer:% X", buf.Bytes())

	if err := RLP.Unmarshal(buf, &b); err != nil {
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

	buf := bytes.NewBuffer(nil)
	if err := RLP.Marshal(buf, a); err != nil {
		t.Errorf("Fail to marshal object on the buffer err=%+v", err)
		return
	}

	log.Printf("Encoded: % X", buf.Bytes())

	if err := RLP.Unmarshal(buf, &b); err != nil {
		t.Errorf("Fail to unmarshal object err=%+v", err)
		return
	}

	if !reflect.DeepEqual(a, b) {
		t.Errorf("Decoded value isnt' same as original")
		return
	}
}
