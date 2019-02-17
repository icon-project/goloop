package transaction

import (
	"reflect"
	"testing"
)

func Test_serialize(t *testing.T) {
	type args struct {
		o map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "basic",
			args: args{
				map[string]interface{}{
					"accounts": map[string]interface{}{
						"balance": "0x1234",
						"name":    "god",
						"address": "cx736846756bcdea54366decfdbdae354789815103",
					},
				},
			},
			want: []byte("accounts.address.cx736846756bcdea54366decfdbdae354789815103.balance.0x1234.name.god"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serialize(tt.args.o); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}
