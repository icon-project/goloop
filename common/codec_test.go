package common

import (
	"log"
	"reflect"
	"testing"
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
		{"Addr", args{
			map[string]interface{}{
				"addr":  NewAddressFromString("hx8888888888888888888888888888888888888888"),
				"bytes": []byte{0x23, 0x45},
				"lst":   []interface{}{"test", "puha"},
				"value": NewHexInt(2),
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalAny(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalAny() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			log.Printf("Bytes:% x\n", got)

			o2, err := UnmarshalAny(got)
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
