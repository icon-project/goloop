package transaction

import (
	"bytes"
	"encoding/hex"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type TransactionListV1 struct {
	list []module.Transaction
	hash []byte
}

func (l *TransactionListV1) Get(i int) (module.Transaction, error) {
	if i >= 0 && i < len(l.list) {
		return l.list[i], nil
	}
	return nil, errors.ErrNotFound
}

type transactionListV1Iterator struct {
	list []module.Transaction
	idx  int
}

func (i *transactionListV1Iterator) Get() (module.Transaction, int, error) {
	if i.idx >= len(i.list) {
		return nil, 0, errors.ErrInvalidState
	}
	return i.list[i.idx], i.idx, nil
}

func (i *transactionListV1Iterator) Has() bool {
	return i.idx < len(i.list)
}

func (i *transactionListV1Iterator) Next() error {
	if i.idx < len(i.list) {
		i.idx++
		return nil
	} else {
		return errors.ErrInvalidState
	}
}

func (l *TransactionListV1) Iterator() module.TransactionIterator {
	return &transactionListV1Iterator{
		list: l.list,
		idx:  0,
	}
}

func calcMergedHash(h1, h2 []byte) []byte {
	var b [128]byte
	copy(b[0:], []byte(hex.EncodeToString(h1)))
	copy(b[64:], []byte(hex.EncodeToString(h2)))
	ts := crypto.SHASum256(b[:])
	return ts[:]
}

func calcMerkleTreeRoot(m [][]byte) []byte {
	if len(m) == 0 {
		var empty [32]byte
		return empty[:]
	}
	ml := make([][]byte, len(m))
	copy(ml, m)
	for mlen := len(ml); mlen > 1; mlen = (mlen + 1) / 2 {
		for i := 0; i < mlen; i += 2 {
			if i+1 < mlen {
				ml[i/2] = calcMergedHash(ml[i], ml[i+1])
			} else {
				ml[i/2] = calcMergedHash(ml[i], ml[i])
			}
		}
	}
	return ml[0]
}

func (l *TransactionListV1) Hash() []byte {
	return l.hash
}

func (l *TransactionListV1) Equal(t module.TransactionList) bool {
	return bytes.Equal(l.Hash(), t.Hash())
}

func (l *TransactionListV1) Flush() error {
	return nil
}

func NewTransactionListV1FromSlice(list []module.Transaction) module.TransactionList {
	ml := make([][]byte, len(list))
	for i, t := range list {
		ml[i] = t.ID()
	}
	hash := calcMerkleTreeRoot(ml)
	return &TransactionListV1{list: list, hash: hash}
}
