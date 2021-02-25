package common

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common/codec"
)

func TestMarshalAny(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"map", args{
			map[string]interface{}{
				"addr":  MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
				"bytes": []byte{0x23, 0x45},
				"lst":   []interface{}{"test", "puha"},
				"value": NewHexInt(2),
				"null":  nil,
			},
		}, false},
		{"addr", args{
			MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
		}, false},
		{"slice", args{
			[]interface{}{
				MustNewAddressFromString("hx8888888888888888888888888888888888888888"),
				"test",
				nil,
				[]byte{0x02, 0x03},
			},
		}, false},
		{"null", args{
			nil,
		}, false},
	}
	for _, tt := range tests {
		for _, c := range []struct {
			name  string
			codec codec.Codec
		}{
			{"MPK", codec.MP},
			{"RLP", codec.RLP},
		} {
			t.Run(fmt.Sprint(tt.name, "-", c.name), func(t *testing.T) {
				got, err := MarshalAny(c.codec, tt.args.obj)
				if (err != nil) != tt.wantErr {
					t.Errorf("MarshalAny() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				log.Printf("Bytes(%s):% x\n", c.name, got)

				o2, err := UnmarshalAny(c.codec, got)
				if err != nil {
					if tt.wantErr {
						return
					}
					t.Errorf("UnmarshalAny() error = %+v", err)
					return
				}
				if !reflect.DeepEqual(tt.args.obj, o2) {
					log.Printf("%+v != %+v", tt.args.obj, o2)
					t.Errorf("MarshalAny() -> UnmarshalAny() results are different")
				}
			})
		}
	}
}
