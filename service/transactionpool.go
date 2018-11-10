package service

import (
	"bytes"
	"container/list"
	"sort"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/trie"
)

const (
	txPoolSize     = 100
	txLiveDuration = int64(60 * time.Second / time.Millisecond) // 60 seconds in millisecond
)

////////////////////
// Transaction Pool
////////////////////
// TODO garbage를 정리하는 방법 필요. 간단하게는 removeList()에 넣어두면 되긴 한데...
// add()할 때 개수 체크 및 candidate()에서 정리
// TODO GC 방법은 정리 필요
// TODO transaction 시간 순으로 정렬 필요
// KN.KIM - transactionPool내에서 TX관리는 linek list로 관리를 해야 삽입삭제가 용이할 것으로 보임.( 삽입삭제가 빈번할 수 있을 것으로 보임)
type transactionPool struct {
	txList *list.List
	//txList.Len() int
	// transactionPool내에 입력하려 하는 txHash가 존재하는지 확인하기 위한 map.
	// list를 끝까지 순환하면서 확인하는 것 보다 map을 사요하는 것이 더 효율적일 것이라 판단.
	txHashMap map[string]*transaction
	mutex     sync.Mutex
}

func NewtransactionPool() *transactionPool {
	return &transactionPool{txList: list.New(), txHashMap: make(map[string]*transaction)}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// TODO: check thread safe below
func (txPool *transactionPool) runGc(expired int64) error {
	txList := txPool.txList

	for iter := txList.Front(); iter != nil; {
		if iter.Value.(*transaction).Timestamp() >= expired {
			break
		}
		tmp := iter.Next()
		txList.Remove(iter)
		delete(txPool.txHashMap, string(iter.Value.(*transaction).ID()))
		iter = tmp
	}
	return nil
}

// transaction에 넣을 때 간단한 검증이 필요하다면, 검증은 외부에서 해야 함.
func (txPool *transactionPool) add(tx *transaction) error {
	txPool.addList([]*transaction{tx})
	return nil
	// KN.KIM 시간이 만료된 것은 받지 않을 것이냐,...
	//expired := makeTimestamp() - txLiveDuration
	//if tx.TimeStamp() < expired {
	//	return nil
	//}
	//txList := txPool.txList
	//
	//if txList.Len() >= txPoolSize {
	//	return nil
	//}
	//txPool.mutex.Lock()
	//defer txPool.mutex.Unlock()
	//if iter := txList.Front(); iter != nil {
	//	if iter.Value.(*transaction).TimeStamp() < expired {
	//		txPool.runGc(expired)
	//	}
	//}
	//
	//if txPool.txList.Len() == 0 {
	//    txList.PushBack(tx)
	//	fmt.Println("first push ID = ", tx.ID(), ", timestamp = ", tx.TimeStamp())
	//	txPool.txHashMap[string(tx.ID())] = tx
	//    return nil
	//}
	//
	//// check whether this transaction is already in txPool
	//if _, ok := txPool.txHashMap[string(tx.ID())]; ok {
	//	// TODO: 추가적으로 address, nonce까지 검사할 필요가 있을까?
	//	fmt.Println("drop ID = ", tx.ID(), ", timestamp = ", tx.TimeStamp())
	//	return nil
	//}
	//
	//inserted := false
	//// TODO: built-in list를 사용하면 아래처럼 search & insert가 효과적이지 않은 것으로 보인다.
	//// 그리고 명시적인 형변환도 필요하다. 런타임시에 효율적이지 않다. 이후에는 직접 list를 구현하는 것이 좋지 않을까???
	//for backIter := txList.Back(); backIter != nil; backIter = backIter.Prev() {
	//    v := backIter.Value.(*transaction)
	//    if v.TimeStamp() <= tx.TimeStamp() {
	//        txList.InsertAfter(tx, backIter)
	//        txPool.txHashMap[string(tx.ID())] = tx
	//        inserted = true
	//        break
	//    }
	//}
	//if inserted == false {
	//    txList.PushFront(tx)
	//	txPool.txHashMap[string(tx.ID())] = tx
	//}
	//return nil
}

// 없다면, len()이 0인 TransactionList를 리턴한다. (nil 아님)
// It returns all candidates for a negative integer n.
func (txPool *transactionPool) candidate(state trie.Mutable, max int) []*transaction {
	// TODO state를 전달받더라도 실제 account info는 address를 통해서 바로 찾는 것이
	// 유리할텐데... trie를 통해서 Get하면 비효율적임.
	// TODO max가 음수이면 모든 transaction을 리턴한다. patch pool에 대해서 필요할 것
	// 같음.
	// TODO validate 작업도 필요.
	// TODO ServiceManager에 하나의 pool을 관리하고 candidate를 구할 때 transition
	// 기반으로 사용된 적이 있는 것을 제외하는 방식으로 구현하려고 하는데, unfinalized
	// branch가 긴 것을 감안하면 좀 더 효과적인 구현이 있을지 고민 필요

	// KN.KIM 먼저 date순으로 정렬되어 있는 transactionPool의 front에서부터 validate를 한 후 transaction에 넣고 전달한다.
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	if max < 0 {
		txList := txPool.txList
		resultTxs := make([]*transaction, txList.Len())
		i := 0
		for iter := txList.Front(); iter != nil; iter = iter.Next() {
			resultTxs[i] = iter.Value.(*transaction)
			i++
		}
		return resultTxs
	}
	txsLen := max
	if txPool.txList.Len() < txsLen {
		txsLen = txPool.txList.Len()
	}

	if txsLen == 0 {
		return []*transaction{}
	}

	txs := make([]*transaction, txsLen)
	txsIndex := 0
	for iter := txPool.txList.Front(); iter != nil; iter = iter.Next() {
		// 현재 validate에 실패한 tx에 대해서 삭제함.
		if iter.Value.(*transaction).validate(state) != nil {
			tmp := iter.Prev()
			txPool.txList.Remove(iter)
			iter = tmp
			continue
		}
		txs[txsIndex] = iter.Value.(*transaction)
		txsIndex++
		if txsIndex == max {
			break
		}
	}

	return txs[:txsIndex]
}

// 이것을 사용할 경우 없음.
func (txPool *transactionPool) remove(tx *transaction) {
	txPool.removeList([]*transaction{tx})
}

// 사용할 경우 없음. 이것도 간단한 검증은 외부에서 수행
func (txPool *transactionPool) addList(txs []*transaction) {
	if len(txs) == 0 {
		return
	}
	expired := makeTimestamp() - txLiveDuration
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	addTxs := append([]*transaction{}, txs...)
	sort.Slice(addTxs, func(i, j int) bool {
		return addTxs[i].Timestamp() > addTxs[j].Timestamp()
	})

	txList := txPool.txList

	if iter := txList.Front(); iter != nil {
		if iter.Value.(*transaction).Timestamp() < expired {
			txPool.runGc(expired)
		}
	}

	if txList.Len() >= txPoolSize {
		return
	}

	// check whether this transaction is already in txPool
	revIter := txList.Back()
	for _, addTx := range addTxs {
		if _, ok := txPool.txHashMap[string(addTx.ID())]; ok {
			// TODO: 추가적으로 address, nonce까지 검사할 필요가 있을까?
			//fmt.Println("drop ID = ", addTx.ID(), ", timestamp = ", addTx.TimeStamp())
			continue
		}
		if addTx.Timestamp() < expired {
			continue
		}

		inserted := false
		for revIter != nil {
			tx := revIter.Value.(*transaction)
			if tx.Timestamp() <= addTx.Timestamp() {
				revIter = txList.InsertAfter(addTx, revIter)
				txPool.txHashMap[string(addTx.ID())] = addTx
				inserted = true
				break
			}
			revIter = revIter.Prev()
		}

		if inserted == false {
			txList.PushFront(addTx)
			txPool.txHashMap[string(addTx.ID())] = addTx
		}
	}

	return
}

// finalize할 때 호출됨.
func (txPool *transactionPool) removeList(txs []*transaction) {
	// TODO 효과적으로 제거하는 방안 필요
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	i := 0
	rmTxs := append([]*transaction{}, txs...)
	sort.Slice(rmTxs, func(i, j int) bool {
		return rmTxs[i].Timestamp() < rmTxs[j].Timestamp()
	})

	t := txPool.txList.Front()
	rmTxsLen := len(rmTxs)
	for i < rmTxsLen && t != nil {
		v := t.Value.(*transaction)
		if v.Timestamp() == rmTxs[i].Timestamp() {
			if bytes.Compare(v.ID(), rmTxs[i].ID()) == 0 {
				tmp := t.Next()
				txPool.txList.Remove(t)
				i++
				t = tmp
				delete(txPool.txHashMap, string(v.ID()))
				continue
			}
		} else if v.Timestamp() > rmTxs[i].Timestamp() {
			// rmTxs[i].TimeStamp()보다 v가 큰 구간에서 i에 해당하는 tx가 pool에 없다고 판단하고 이전 tx부터 시작한다.
			if t.Prev() != nil {
				t = t.Prev()
			}
			i++
			continue
		}
		t = t.Next()
	}
}
