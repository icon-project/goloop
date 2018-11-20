package service

import (
	"bytes"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"log"
)

////////////////////
// Transaction Pool
////////////////////
// TODO garbage를 정리하는 방법 필요. 간단하게는 removeList()에 넣어두면 되긴 한데...
// add()할 때 개수 체크 및 candidate()에서 정리.
// TODO tx를 버리는 기준 확인 필요
// TODO tx 시간 순으로 정렬 필요
// TODO 당연하지만, lock 잘 잡고...
//type transactionPool struct {
//}
//
//// transaction에 넣을 때 간단한 검증이 필요하다면, 검증은 외부에서 해야 함.
//func (pool *transactionPool) add(tx transaction) error {
//	return nil
//}
//
//// 없다면, len()이 0인 TransactionList를 리턴한다. (nil 아님)
//// It returns all candidates for a negative integer n.
//func (pool *transactionPool) candidate(state trie.Mutable, max int) []transaction {
//	// TODO state를 전달받더라도 실제 account info는 address를 통해서 바로 찾는 것이
//	// 유리할텐데... trie를 통해서 Get하면 비효율적임.
//	// TODO max가 음수이면 모든 transaction을 리턴한다. patch pool에 대해서 필요할 것
//	// 같음.
//	// TODO validate 작업도 필요.
//	// TODO ServiceManager에 하나의 pool을 관리하고 candidate를 구할 때 transition
//	// 기반으로 사용된 적이 있는 것을 제외하는 방식으로 구현하려고 하는데, unfinalized
//	// branch가 긴 것을 감안하면 좀 더 효과적인 구현이 있을지 고민 필요
//	return nil
//}
//
//// 이것을 사용할 경우 없음.
//func (pool *transactionPool) remove(tx transaction) {
//}
//
//// 사용할 경우 없음. 이것도 간단한 검증은 외부에서 수행
//func (pool *transactionPool) addList(tx []transaction) {
//
//}
//
//// finalize할 때 호출됨.
//func (pool *transactionPool) removeList(tx []transaction) {
//	// TODO 효과적으로 제거하는 방안 필요
//}

////////////////////
// Transaction List
////////////////////
// TODO to avoid name conflict, temporarily take 'list' instead of 'List'
// TODO to avoid name conflict, temporarily take 'interator' instead of 'Iterator'
type transactionlist struct {
	txs      []*transaction
	snapshot trie.Snapshot
}

type transactionlistIterator struct {
	list []*transaction
	iter trie.Iterator
	idx  int
}

func (l *transactionlist) Get(i int) (module.Transaction, error) {
	//	// TODO handle with trie when txs is nil
	if len(l.txs) > 0 {
		if i >= 0 && i < len(l.txs) {
			return l.txs[i], nil
		} else {
			return nil, common.ErrNotFound
		}
	}
	b, err := codec.MP.MarshalToBytes(uint(i))
	if err != nil {
		return nil, err
	}
	txBytes, err := l.snapshot.Get(b)
	if err != nil || txBytes == nil {
		return nil, err
	}
	var tx module.Transaction
	tx, err = newTransaction(txBytes)
	if err != nil {
		log.Panicf("Failed to create transaction from %x\n", txBytes)
		return nil, err
	}
	return tx, nil
}

func (l *transactionlist) Iterator() module.TransactionIterator {
	return &transactionlistIterator{
		list: l.txs,
		idx:  0,
		iter: l.snapshot.Iterator(),
	}
}

func (l *transactionlist) Hash() []byte {
	return l.snapshot.Hash()
}

func (l *transactionlist) Equal(t module.TransactionList) bool {
	return bytes.Equal(l.snapshot.Hash(), t.Hash())
}

// Add Flush interface in transactionList
func (l *transactionlist) Flush() error {
	return l.snapshot.Flush()
}

func (i *transactionlistIterator) Get() (module.Transaction, int, error) {
	if len(i.list) > 0 {
		if i.idx >= len(i.list) {
			return nil, 0, common.ErrInvalidState
		}
		return i.list[i.idx], i.idx, nil
	}

	txBytes, txKey, err := i.iter.Get()
	if txBytes == nil || err != nil {
		log.Printf("Failed to get through iterator. txBytes = %x, err = %s\n", txKey, err)
		return nil, 0, err
	}

	var idx uint
	if _, err := codec.MP.UnmarshalFromBytes(txKey, &idx); err != nil {
		log.Panicf("Failed to unmarshar from bytes. %x\n", txKey)
		return nil, 0, err
	}

	var tx module.Transaction
	tx, err = newTransaction(txBytes)
	if err != nil {
		log.Panicf("Failed to create transaction from %x\n", txBytes)
		return nil, 0, err
	}
	return tx, int(idx), nil
}

func (i *transactionlistIterator) Has() bool {
	if len(i.list) > 0 {
		return i.idx < len(i.list)
	}
	return i.iter.Has()
}

func (i *transactionlistIterator) Next() error {
	if len(i.list) > 0 {
		if i.idx < len(i.list) {
			i.idx++
			return nil
		} else {
			return common.ErrInvalidState
		}
	}

	if i.iter.Next() != nil {
		return common.ErrInvalidState
	}

	return nil
}

func TestFlush(txs module.TransactionList) error {
	txsImpl := txs.(*transactionlist)
	txsImpl.Flush()
	return nil
}

func newTransactionListFromList(db db.Database, list []module.Transaction) module.TransactionList {
	txs := make([]*transaction, len(list))
	ok := false
	for i, tx := range list {
		if txs[i], ok = tx.(*transaction); ok == false {
			log.Fatalf("Failed to assertion.")
			return nil
		}
	}
	tm := trie_manager.New(db)
	mt := tm.NewMutable(nil)
	for idx, tr := range list {
		k, _ := codec.MP.MarshalToBytes(uint(idx))
		v := tr.Bytes()
		err := mt.Set(k, v)
		if err != nil {
			log.Fatalf("NewTransanctionListFromSlice FAILs")
			return nil
		}
	}
	return &transactionlist{txs: txs, snapshot: mt.GetSnapshot()}
}

func newTransactionListFromHash(db db.Database, hash []byte) module.TransactionList {
	//	// TODO Fill txs or not? If it doesn't fill txs, then fix all using txs directly.
	tm := trie_manager.New(db)
	trie := tm.NewMutable(hash)
	return &transactionlist{snapshot: trie.GetSnapshot()}
}

// TODO Is db is good for parameter?
func newTransactionList(db db.Database, txs []*transaction) *transactionlist {
	if txs == nil {
		txs = make([]*transaction, 0)
	}
	trie := trie_manager.NewMutable(db, nil)
	for i, tx := range txs {
		k, _ := codec.MP.MarshalToBytes(uint(i))
		err := trie.Set(k, tx.Bytes())
		if err != nil {
			return nil
		}
	}
	return &transactionlist{txs: txs, snapshot: trie.GetSnapshot()}
}
