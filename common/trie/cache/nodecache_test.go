package cache

import "testing"

func Test_indexByNibs(t *testing.T) {
	type args struct {
		nibs []byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"root", args{[]byte{}}, 0},
		{"lv1-first", args{[]byte{0x0}}, 0x1},
		{"lv1-last", args{[]byte{0xf}}, 0x10},
		{"lv2-first", args{[]byte{0x0, 0x0}}, 0x11},
		{"lv2-last", args{[]byte{0xf, 0xf}}, 0x110},
		{"lv3-first", args{[]byte{0x0, 0x0, 0x0}}, 0x111},
		{"lv3-last", args{[]byte{0xf, 0xf, 0xf}}, 0x1110},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexByNibs(tt.args.nibs); got != tt.want {
				t.Errorf("indexByNibs() = %v, want %v", got, tt.want)
			}
		})
	}
}
