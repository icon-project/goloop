package transaction

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common/crypto"
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
					"accounts": []map[string]interface{}{
						{
							"balance": "0x1234",
							"name":    "god",
							"address": "hx736846756bcdea54366decfdbdae354789815103",
						},
					},
				},
			},
			want: []byte("accounts.[{address.hx736846756bcdea54366decfdbdae354789815103.balance.0x1234.name.god}]"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js, err := json.Marshal(tt.args.o)
			if err != nil {
				t.Errorf("unexpected error on marshal err=%+v", err)
				return
			}

			tx, err := parseV3Genesis(js, false)
			if err != nil {
				t.Errorf("Fail to make genesis tx with supplied err=%+v", err)
				return
			}

			got := tx.ID()
			exp := crypto.SHA3Sum256(append([]byte("genesis_tx."), tt.want...))

			if !reflect.DeepEqual(exp, got) {
				t.Errorf("Has different hash value")
			}
		})
	}
}
