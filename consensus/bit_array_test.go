package consensus

import (
	"testing"
)

func TestBitArrayPickRandom(t *testing.T) {
	testCases := []struct {
		len    int
		ones   []int
		repeat int
		expect int
	}{
		{10, []int{}, 2, -1},
		{10, []int{0}, 2, 0},
		{10, []int{1}, 2, 1},
		{10, []int{9}, 2, 9},
		{65, []int{0}, 2, 0},
		{65, []int{62}, 2, 62},
		{65, []int{63}, 2, 63},
		{65, []int{64}, 2, 64},
	}
	for _, c := range testCases {
		ba := newBitArray(c.len)
		for i := 0; i < len(c.ones); i++ {
			ba.Set(c.ones[i])
			if !ba.Get(c.ones[i]) {
				t.Errorf("not set\n")
			}
		}
		for i := 0; i < c.repeat; i++ {
			v := ba.PickRandom()
			if v != c.expect {
				t.Errorf("bad random value ones:%v expected:%v actual:%v\n", c.ones, c.expect, v)
			}
		}
	}
}

func TestBitArrayPickRandom2(t *testing.T) {
	testCases := []struct {
		len    int
		ones   []int
		repeat int
	}{
		{100, []int{1, 2, 3, 4, 5}, 1000},
		{100, []int{1, 2, 3, 4, 5, 63, 64, 77}, 1000},
		{500, []int{64, 77, 399}, 1000},
	}
	for _, c := range testCases {
		ba := newBitArray(c.len)
		for i := 0; i < len(c.ones); i++ {
			ba.Set(c.ones[i])
		}
		ba2 := newBitArray(c.len)
		for i := 0; i < c.repeat; i++ {
			v := ba.PickRandom()
			if !ba.Get(v) {
				t.Errorf("bad random value ones:%v random:%v\n", c.ones, v)
			}
			ba2.Set(v)
		}
		for i := 0; i < len(ba.Words); i++ {
		}
		if !ba.Equal(ba2) {
			t.Errorf("not random enough, expected:%v random:%v\n", ba, ba2)
		}
	}
}
