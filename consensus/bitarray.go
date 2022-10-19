package consensus

import (
	"fmt"
	"math/bits"
	"math/rand"
)

type word = uint

const wordBits = 32 << (^uint(0) >> 63) // either 32 or 64

type bitArray struct {
	NumBits int
	Words   []word
}

func (ba *bitArray) Len() int {
	return ba.NumBits
}

func (ba *bitArray) Set(idx int) {
	if idx >= ba.NumBits {
		return
	}
	ba.Words[idx/wordBits] = ba.Words[idx/wordBits] | (1 << uint(idx%wordBits))
}

func (ba *bitArray) Unset(idx int) {
	if idx >= ba.NumBits {
		return
	}
	ba.Words[idx/wordBits] = ba.Words[idx/wordBits] &^ (1 << uint(idx%wordBits))
}

func (ba *bitArray) Put(idx int, v bool) {
	if idx >= ba.NumBits {
		return
	}
	if v {
		ba.Set(idx)
	} else {
		ba.Unset(idx)
	}
}

func (ba *bitArray) Get(idx int) bool {
	if idx >= ba.NumBits {
		return false
	}
	return ba.Words[idx/wordBits]&(1<<uint(idx%wordBits)) != 0
}

func (ba *bitArray) Flip() {
	l := len(ba.Words)
	for i := 0; i < l; i++ {
		ba.Words[i] = ^ba.Words[i]
	}
	if l > 0 {
		ba.Words[l-1] = ba.Words[l-1] & ((1 << uint(ba.NumBits%wordBits)) - 1)
	}
}

func (ba *bitArray) AssignAnd(ba2 *bitArray) {
	lba := len(ba.Words)
	lba2 := len(ba2.Words)
	if ba.NumBits > ba2.NumBits {
		ba.Words = ba.Words[:lba2]
		ba.NumBits = ba2.NumBits
		lba = lba2
	}
	for i := 0; i < lba; i++ {
		ba.Words[i] &= ba2.Words[i]
	}
}

func (ba *bitArray) PickRandom() int {
	var count int
	for i := 0; i < len(ba.Words); i++ {
		count = count + bits.OnesCount(ba.Words[i])
	}
	if count == 0 {
		return -1
	}
	pick := rand.Intn(count)
	for i := 0; i < len(ba.Words); i++ {
		c := bits.OnesCount(ba.Words[i])
		if pick < c {
			for idx := i * wordBits; idx < ba.NumBits; idx++ {
				if ba.Get(idx) {
					if pick == 0 {
						return idx
					}
					pick--
				}
			}
		}
		pick = pick - c
	}
	panic("PickRandom: internal error")
}

func (ba bitArray) String() string {
	// TODO better form?
	return fmt.Sprintf("%x", ba.Words)
}

func (ba *bitArray) Equal(ba2 *bitArray) bool {
	lba := len(ba.Words)
	if ba.NumBits != ba2.NumBits {
		return false
	}
	for i := 0; i < lba; i++ {
		if ba.Words[i] != ba2.Words[i] {
			return false
		}
	}
	return true
}

func (ba *bitArray) Copy() *bitArray {
	ba2 := newBitArray(ba.NumBits)
	copy(ba2.Words, ba.Words)
	return ba2
}

func newBitArray(n int) *bitArray {
	return &bitArray{n, make([]word, (n+wordBits-1)/wordBits)}
}
