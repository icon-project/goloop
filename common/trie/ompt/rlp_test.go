package ompt

import (
	"reflect"
	"testing"
)

func Test_rlpEncode(t *testing.T) {
	type args struct {
		o interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "Basic",
			args:    args{nil},
			want:    []byte{0x80},
			wantErr: false,
		},
		{
			name:    "SmallNumber",
			args:    args{[]byte{0x7f}},
			want:    []byte{0x7f},
			wantErr: false,
		},
		{
			name:    "ShortList",
			args:    args{[][]byte{[]byte{0x01}}},
			want:    []byte{0xC1, 0x01},
			wantErr: false,
		},
		{
			name: "mpt.leaf1",
			args: args{
				&leaf{keys: []byte{9, 2}, value: bytesObject([]byte{2, 3, 0xFF})},
			},
			want:    []byte{0xC7, 0x82, 0x20, 0x92, 0x83, 0x02, 0x03, 0xff},
			wantErr: false,
		},
		{
			name: "mpt.extension1",
			args: args{
				&extension{
					keys: []byte{2, 3},
					next: &leaf{keys: []byte{9, 2}, value: bytesObject([]byte{2, 3, 0xFF})},
				},
			},
			want:    []byte{0xCB, 0x82, 0x00, 0x23, 0xC7, 0x82, 0x20, 0x92, 0x83, 0x02, 0x03, 0xFF},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rlpEncode(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("rlpEncode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rlpEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzRLPParseBytes(f *testing.F) {
	f.Add([]byte("\x80"))
	f.Add([]byte("\x20"))
	f.Add([]byte("\x82\xab\x78"))
	f.Fuzz(func(t *testing.T, data []byte) {
		rlpParseBytes(data)
	})
}

func FuzzRLPParseList(f *testing.F) {
	f.Add([]byte("\xc0"))
	f.Fuzz(func(t *testing.T, data []byte) {
		rlpParseList(data)
	})
}
