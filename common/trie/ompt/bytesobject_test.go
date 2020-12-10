package ompt

import (
	"testing"

	"github.com/icon-project/goloop/common/trie"
)

func Test_bytesObject_Equal(t *testing.T) {
	type args struct {
		n trie.Object
	}
	tests := []struct {
		name string
		o    bytesObject
		args args
		want bool
	}{
		{"NilWithNil", nil, args{nil}, true},
		{"NilWithNilPtr", nil, args{bytesObject(nil)}, true},
		{"NilWithEmpty", nil, args{bytesObject([]byte{})}, true},
		{"NilWithNonNil", nil, args{bytesObject([]byte{0x00})}, false},
		{"EmptyWithNil", bytesObject([]byte{}), args{nil}, true},
		{"EmptyWithNilPtr", bytesObject([]byte{}), args{bytesObject(nil)}, true},
		{"NonNilWithNil", bytesObject([]byte{0x00}), args{nil}, false},
		{"NonNilWithNilPtr", bytesObject([]byte{0x00}), args{bytesObject(nil)}, false},
		{"Case1", bytesObject([]byte{0x00}), args{bytesObject([]byte{0x00})}, true},
		{"Case2", bytesObject([]byte{0x00}), args{bytesObject([]byte{0x02})}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.Equal(tt.args.n); got != tt.want {
				t.Errorf("bytesObject.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
