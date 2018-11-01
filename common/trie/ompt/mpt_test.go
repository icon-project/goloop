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
				reflect.TypeOf(bytesObject(nil)),
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
				reflect.TypeOf(bytesObject(nil)),
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
				},
			},
			want: result{
				[]entry{
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01, 0x23, 0x46}, nil},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove3",
			args: args{
				db.NewMapDB(),
				nil,
				reflect.TypeOf(bytesObject(nil)),
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
			},
			want: result{
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
				[]byte{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Println("Makes new MPT")
			got := NewMPT(tt.args.d, tt.args.h, tt.args.t)
			if got == nil {
				t.Errorf("NewMPT() = %v, want non nil", got)
				return
			}
			for _, e := range tt.args.e {
				var err error
				if e.v != nil {
					log.Printf("Set(%x,%x)", e.k, e.v)
					err = got.Set(e.k, bytesObject(e.v))
				} else {
					log.Printf("Delete(%x)", e.k)
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
			// 	t.Errorf("Hash() = %#x, want %#x", h, tt.want.h)
			// }
			log.Println("Flush")
			s.Flush()

			s2 := NewMPT(tt.args.d, h, tt.args.t)
			log.Printf("Dump current snapshot from hash")
			s2.Dump()
			log.Printf("Verify results")
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
						t.Errorf("Key(%x) expected %x result is nil", e.k, e.v)
						continue
					}
				}
				if !bytes.Equal(obj.Bytes(), e.v) {
					t.Errorf("Key(%x) expected %x result %x", e.k, e.v, obj.Bytes())
					s2.Dump()
					break
				}
			}
		})
	}
}
