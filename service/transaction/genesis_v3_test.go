package transaction

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

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
					"chain": map[string]interface{}{},
				},
			},
			want: []byte("accounts.[{address.hx736846756bcdea54366decfdbdae354789815103.balance.0x1234.name.god}].chain.{}"),
		},
		{
			name: "icon",
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
			want: []byte("accounts.address.hx736846756bcdea54366decfdbdae354789815103.balance.0x1234.name.god"),
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

func TestGenesisV3_ICONMainNet(t *testing.T) {
	genesis := "{\"accounts\":[{\"name\":\"god\",\"address\":\"hx54f7853dc6481b670caf69c5a27c7c8fe5be8269\",\"balance\":\"0x2961fff8ca4a62327800000\"},{\"name\":\"treasury\",\"address\":\"hx1000000000000000000000000000000000000000\",\"balance\":\"0x0\"}],\"message\":\"A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \\\"to be\\\" but the fabric of the rhizome is the conjunction, \\\"and ... and ...and...\\\"This conjunction carries enough force to shake and uproot the verb \\\"to be.\\\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\\n\\n - Mille Plateaux, Gilles Deleuze & Felix Guattari\\n\\n\\\"Hyperconnect the world\\\"\"}"
	tx, err := parseV3Genesis([]byte(genesis), false)
	assert.NoError(t, err)
	gtx := tx.(*genesisV3)
	assert.Equal(t, ICONMainNetGenesisIDBytes, gtx.ID())
	assert.Equal(t, ICONMainNetCID, gtx.CID())
	assert.Equal(t, ICONMainNetCID, gtx.NID())
}
