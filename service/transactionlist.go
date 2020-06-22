package service

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	txBucketCount   = 256
	txBloomBits     = 12
	txBloomSize     = 2 << txBloomBits
	txCountForBloom = 256
)

func indexAndBucketKeyFromKey(k string) (int, string) {
	return int(k[0]), k[1:]
}

type transactionList struct {
	size      int
	listFront *txElement
	listBack  *txElement

	idMap        []map[string]*txElement
	srcMapToLast []map[string]*txElement
}

type txElement struct {
	value transaction.Transaction
	ts    int64
	err   error

	list               *transactionList
	listNext, listPrev *txElement
	srcNext, srcPrev   *txElement

	bloom *txBloom
}

func (t *txElement) Next() *txElement {
	return t.listNext
}

func (t *txElement) Prev() *txElement {
	return t.listPrev
}

func (t *txElement) Remove() bool {
	if t.list != nil {
		return t.list.Remove(t)
	}
	return true
}

func (t *txElement) TimeStamp() int64 {
	return t.ts
}

func (t *txElement) Value() transaction.Transaction {
	return t.value
}

func (t *txElement) updateBloom() {
	if t.bloom != nil {
		return
	}
	var bloom *txBloom
	if t.listPrev != nil {
		bloom = t.listPrev.bloom
	} else {
		bloom = new(txBloom)
	}
	t.bloom = bloom.Add(t.value.ID())
}

func (t *txElement) GetBloom() (uint, *big.Int) {
	return t.bloom.GetBloom()
}

func BloomContains(bits uint, value *big.Int, id []byte) bool {
	var filter big.Int
	var result big.Int
	if len(id) < crypto.HashLen {
		id = crypto.SHA3Sum256(id)
	}
	mask := (1 << bits) - 1
	idx1 := int(binary.BigEndian.Uint16(id[:])) & mask
	idx2 := int(binary.BigEndian.Uint16(id[2:])) & mask
	idx3 := int(binary.BigEndian.Uint16(id[4:])) & mask
	filter.SetBit(&filter, idx1, 1)
	filter.SetBit(&filter, idx2, 1)
	filter.SetBit(&filter, idx3, 1)
	result.And(&filter, value)
	return result.Cmp(&filter) == 0
}

func (t *txElement) Contained(bits uint, value *big.Int) bool {
	return BloomContains(bits, value, t.value.ID())
}

type txBloom struct {
	count int
	value big.Int
	next  *txBloom
}

func (b *txBloom) Add(id []byte) *txBloom {
	if b.count >= txCountForBloom {
		ptr := b.next
		for ptr != nil {
			if ptr.count < txCountForBloom {
				return ptr.Add(id)
			}
			ptr = ptr.next
		}
		return new(txBloom).Add(id)
	}
	if len(id) < crypto.HashLen {
		id = crypto.SHA3Sum256(id)
	}
	idx1 := int(binary.BigEndian.Uint16(id[:])) & (txBloomSize - 1)
	idx2 := int(binary.BigEndian.Uint16(id[2:])) & (txBloomSize - 1)
	idx3 := int(binary.BigEndian.Uint16(id[4:])) & (txBloomSize - 1)
	b.value.SetBit(&b.value, idx1, 1)
	b.value.SetBit(&b.value, idx2, 1)
	b.value.SetBit(&b.value, idx3, 1)
	b.count += 1
	return b
}

func (b *txBloom) GetBloom() (uint, *big.Int) {
	value := new(big.Int)
	for ptr := b; ptr != nil; ptr = ptr.next {
		value.Or(value, &ptr.value)
	}
	return txBloomBits, value
}

func (l *transactionList) Add(tx transaction.Transaction, ts bool) error {
	tidBk, tidSlot := indexAndBucketKeyFromKey(string(tx.ID()))
	if _, ok := l.idMap[tidBk][tidSlot]; ok {
		return ErrDuplicateTransaction
	}

	e := &txElement{
		value: tx,
		list:  l,
	}
	if ts {
		e.ts = time.Now().UnixNano()
	}

	l.idMap[tidBk][tidSlot] = e

	uidBk, uidSlot := indexAndBucketKeyFromKey(string(tx.From().ID()))
	t2, ok := l.srcMapToLast[uidBk][uidSlot]

	var insertPos *txElement
	if ok {
		ts := tx.Timestamp()
		if t2.value.Timestamp() > ts {
			insertPos = t2
			for t2 = t2.srcPrev; t2 != nil; t2 = t2.srcPrev {
				if t2.value.Timestamp() > ts {
					insertPos = t2
				} else {
					break
				}
			}
			if insertPos.srcPrev != nil {
				e.srcPrev = insertPos.srcPrev
				e.srcPrev.srcNext = e
			}
			e.srcNext = insertPos
			insertPos.srcPrev = e
		} else {
			l.srcMapToLast[uidBk][uidSlot] = e
		}
	} else {
		l.srcMapToLast[uidBk][uidSlot] = e
	}

	if insertPos != nil {
		if insertPos.listPrev != nil {
			e.listPrev = insertPos.listPrev
			e.listPrev.listNext = e
		} else {
			l.listFront = e
		}
		e.listNext = insertPos
		insertPos.listPrev = e
	} else {
		if l.listBack != nil {
			e.listPrev = l.listBack
			e.listPrev.listNext = e
		} else {
			l.listFront = e
		}
		l.listBack = e
	}
	e.updateBloom()
	l.size += 1
	return nil
}

func (l *transactionList) RemoveTx(tx module.Transaction) (bool, int64) {
	tidBk, tidSlot := indexAndBucketKeyFromKey(string(tx.ID()))
	if e, ok := l.idMap[tidBk][tidSlot]; ok {
		return l.Remove(e), e.ts
	}
	return false, 0
}

func (l *transactionList) Remove(t *txElement) bool {
	if t.list == nil || t.list != l {
		return false
	}

	if l.listFront == t {
		l.listFront = t.listNext
	}
	if l.listBack == t {
		l.listBack = t.listPrev
	}
	if t.listNext != nil {
		t.listNext.listPrev = t.listPrev
	}
	if t.listPrev != nil {
		t.listPrev.listNext = t.listNext
	}
	t.listNext = nil
	t.listPrev = nil

	uidBk, uidSlot := indexAndBucketKeyFromKey(string(t.value.From().ID()))
	t2 := l.srcMapToLast[uidBk][uidSlot]
	if t2 == t {
		if t.srcPrev != nil {
			l.srcMapToLast[uidBk][uidSlot] = t.srcPrev
		} else {
			delete(l.srcMapToLast[uidBk], uidSlot)
		}
	}
	if t.srcPrev != nil {
		t.srcPrev.srcNext = t.srcNext
	}
	if t.srcNext != nil {
		t.srcNext.srcPrev = t.srcPrev
	}
	t.srcNext = nil
	t.srcPrev = nil

	tidBk, tidSlot := indexAndBucketKeyFromKey(string(t.value.ID()))
	delete(l.idMap[tidBk], tidSlot)

	l.size -= 1
	t.list = nil
	return true
}

func (l *transactionList) Front() *txElement {
	return l.listFront
}

func (l *transactionList) Len() int {
	return l.size
}

func (l *transactionList) HasTx(id []byte) bool {
	tidBk, tidSlot := indexAndBucketKeyFromKey(string(id))
	_, ok := l.idMap[tidBk][tidSlot]
	return ok
}

func (l *transactionList) GetBloom() (uint, *big.Int) {
	if l.listFront == nil {
		return 0, new(big.Int)
	}
	return l.listFront.GetBloom()
}

func newTransactionList() *transactionList {
	l := new(transactionList)

	l.idMap = make([]map[string]*txElement, txBucketCount)
	l.srcMapToLast = make([]map[string]*txElement, txBucketCount)
	for i := 0; i < txBucketCount; i++ {
		l.idMap[i] = make(map[string]*txElement)
		l.srcMapToLast[i] = make(map[string]*txElement)
	}
	return l
}
