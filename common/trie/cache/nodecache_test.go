package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
)

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

func bytesToNibs(k []byte) []byte {
	ks := len(k)
	nibs := make([]byte, ks*2)

	for i, v := range k {
		nibs[i*2] = (v >> 4) & 0x0F
		nibs[i*2+1] = v & 0x0F
	}
	return nibs
}

func TestNodeCache_Index(t *testing.T) {
	cache := NewNodeCache(3, 0, "")

	d1 := []byte("data")
	h1 := crypto.SHA3Sum256(d1)
	n1 := bytesToNibs(h1)

	cache.Put(n1[0:2], h1, d1)
	data, ok := cache.Get(n1[0:2], h1)

	assert.True(t, ok)
	assert.Equal(t, data, d1)

	d2 := []byte("hello")
	h2 := crypto.SHA3Sum256(d2)
	n2 := bytesToNibs(h2)

	cache.Put(n2[0:3], h2, d2)
	data, ok = cache.Get(n2[0:3], h2)

	assert.False(t, ok)
	assert.Nil(t, data)
}

func Benchmark_NodeCache(b *testing.B) {
	cache := NewNodeCache(3, 0, "")

	d2 := []byte("hello")
	h2 := crypto.SHA3Sum256(d2)
	n2 := bytesToNibs(h2)

	b.Run("long", func(b *testing.B) {
		for i := 0; i < 1000; i = i + 1 {
			cache.Put(n2[0:3], h2, d2)
			cache.Get(n2[0:3], h2)
		}
	})

	b.Run("long2", func(b *testing.B) {
		for i := 0; i < 1000; i = i + 1 {
			cache.Put(n2[0:8], h2, d2)
			cache.Get(n2[0:8], h2)
		}
	})

	b.Run("short", func(b *testing.B) {
		for i := 0; i < 1000; i = i + 1 {
			cache.Put(n2[0:2], h2, d2)
			cache.Get(n2[0:2], h2)
		}
	})
}
