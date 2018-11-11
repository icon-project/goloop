package service

import (
	"bytes"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
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
type (
	transactionlist struct {
		txs  []*transaction // can be nil at the beginning
		trie trie.Immutable
	}

	transactioniterator struct {
		list []*transaction
		idx  int
	}
)

func newTransactionList(db db.Database, txs []*transaction) *transactionlist {
	trie := trie_manager.NewMutable(db, nil)
	for i, tx := range txs {
		k, _ := codec.MP.MarshalToBytes(uint(i))
		err := trie.Set(k, tx.Bytes())
		if err != nil {
			return nil
		}
	}
	return &transactionlist{txs: txs, trie: trie.GetSnapshot()}
}

func newTransactionListFromHash(db db.Database, hash []byte) *transactionlist {
	trie := trie_manager.NewImmutable(db, hash)
	return &transactionlist{txs: nil, trie: trie}
}

func (l *transactionlist) Get(n int) (module.Transaction, error) {
	if n < 0 {
		return nil, common.ErrIllegalArgument
	}
	// TODO handle with trie when txs is nil
	if n < len(l.txs) {
		return l.txs[n], nil
	}
	return nil, common.ErrNotFound
}

func (l *transactionlist) Iterator() module.TransactionIterator {
	return &transactioniterator{list: l.txs, idx: 0}
}

func (l *transactionlist) Hash() []byte {
	return l.trie.Hash()
}

func (l *transactionlist) Equal(txList module.TransactionList) bool {
	if txList == nil {
		return false
	}

	return bytes.Equal(l.Hash(), txList.Hash())
}

func (i *transactioniterator) Get() (module.Transaction, int, error) {
	if i.idx >= len(i.list) {
		return nil, 0, common.ErrInvalidState
	}
	return i.list[i.idx], i.idx, nil
}

func (i *transactioniterator) Has() bool {
	return i.idx < len(i.list)
}

func (i *transactioniterator) Next() error {
	if i.idx < len(i.list) {
		i.idx++
		return nil
	}
	return common.ErrInvalidState
}

////////////////////
// Receipt / Receipt List
////////////////////
type receipt struct {
	// TODO 정의
}

func (r *receipt) Bytes() ([]byte, error) {
	return nil, nil
}

type receiptList struct {
	receipts []receipt
	hash     []byte

	trie trie.Mutable
}

func (l *receiptList) Get(n int) (module.Receipt, error) {
	if n < 0 || n >= len(l.receipts) {
		return nil, common.ErrIllegalArgument
	}
	return &l.receipts[n], nil
}
func (l *receiptList) Size() int {
	return len(l.receipts)
}

func (l *receiptList) Hash() []byte {
	if l.hash == nil {
		for i, r := range l.receipts {
			// TODO trie 내부에서 key hash를 안 하는지 확인 필요
			bytes, _ := r.Bytes()
			if len(bytes) > 0 {
				// TODO i가 256를 넘을 경우를 감안한 byte encoding 수정
				l.trie.Set([]byte{byte(i)}, bytes)
			}
		}
		l.hash = l.trie.GetSnapshot().Hash()
	}
	return l.hash
}
