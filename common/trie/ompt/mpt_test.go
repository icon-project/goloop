package ompt

import (
	"bytes"
	"encoding/hex"
	"log"
	"reflect"
	"testing"

	ge "github.com/go-errors/errors"
	"github.com/icon-project/goloop/common/db"
)

func TestNewMPT(t *testing.T) {
	type entry struct {
		k, v []byte
	}
	type args struct {
		d db.Database
		h []byte
		t reflect.Type
		e []entry
	}
	type result struct {
		e []entry
		h []byte
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "AddRemove1",
			args: args{
				db.NewMapDB(),
				nil,
				reflect.TypeOf(BytesObject(nil)),
				[]entry{
					{[]byte{1, 2, 3}, []byte{1}},
					{[]byte{1, 2, 3}, []byte{2}},
					{[]byte{1, 2, 3}, []byte{0x11, 0x22, 0x33}},
					{[]byte{1, 2, 4}, []byte{0x11, 0x22, 0x44}},
					{[]byte{1, 2, 3, 4}, []byte{0x11, 0x22, 0x33, 0x44}},
					{[]byte{1, 2, 4}, nil},
					{[]byte{1, 2, 3, 4}, nil},
				},
			},
			want: result{
				[]entry{
					{[]byte{1, 2, 3}, []byte{0x11, 0x22, 0x33}},
					{[]byte{1, 2, 3, 4}, nil},
					{[]byte{1, 2, 4}, nil},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove2",
			args: args{
				db.NewMapDB(),
				nil,
				reflect.TypeOf(BytesObject(nil)),
				[]entry{
					{[]byte{1}, []byte{0x11}},
					{[]byte{1, 2}, []byte{0x11, 0x22, 0x44}},
					{[]byte{1, 2, 3}, []byte{0x11, 0x22, 0x33}},
					{[]byte{1, 2, 3, 4}, []byte{0x11, 0x22, 0x33, 0x44}},
					{[]byte{1, 2}, nil},
					{[]byte{1, 2, 3}, nil},
					{[]byte{1}, nil},
				},
			},
			want: result{
				[]entry{
					{[]byte{1}, nil},
					{[]byte{1, 2}, nil},
					{[]byte{1, 2, 3}, nil},
					{[]byte{1, 2, 3, 4}, []byte{0x11, 0x22, 0x33, 0x44}},
				},
				[]byte{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMPT(tt.args.d, tt.args.h, tt.args.t)
			if got == nil {
				t.Errorf("NewMPT() = %v, want non nil", got)
				return
			}
			for _, e := range tt.args.e {
				var err error
				if e.v != nil {
					err = got.Set(e.k, BytesObject(e.v))
				} else {
					err = got.Delete(e.k)
				}
				if err != nil {
					t.Errorf("FAIL to set key to value")
					return
				}
			}
			s := got.GetSnapshot()
			h := s.Hash()
			// if !bytes.Equal(h, tt.want.h) {
			// 	s.(*mpt).Dump()
			// 	t.Errorf("Hash() = %v, want %v", h, tt.want.h)
			// }
			s.Flush()

			s2 := NewMPT(tt.args.d, h, tt.args.t)
			for _, e := range tt.want.e {
				obj, err := s2.Get(e.k)
				if err != nil {
					log.Println(err.(*ge.Error).ErrorStack())
					t.Errorf("Key(%s) return error=%v",
						hex.EncodeToString(e.k), err)
					continue
				}
				if obj == nil {
					if e.v == nil {
						continue
					} else {
						t.Errorf("Key(%s) expected %s result is nil",
							hex.EncodeToString(e.k), hex.EncodeToString(e.v))
						continue
					}
				}
				if !bytes.Equal(obj.Bytes(), e.v) {
					s2.Dump()
					t.Errorf("Key(%s) expected %s result %s",
						hex.EncodeToString(e.k), hex.EncodeToString(e.v),
						hex.EncodeToString(obj.Bytes()))
					break
				}
			}
		})
	}
}
