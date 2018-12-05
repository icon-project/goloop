package service

import (
	"container/list"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	configOnCheckingTimestamp = false // set true if you want check timestamp in txpool
	txPoolSize                = 5000
	txLiveDuration            = int64(60 * time.Second / time.Microsecond) // 60 seconds in microsecond
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
	txdb   db.Bucket
	txList *list.List
	//txList.Len() int
	// transactionPool내에 입력하려 하는 txHash가 존재하는지 확인하기 위한 map.
	// list를 끝까지 순환하면서 확인하는 것 보다 map을 사요하는 것이 더 효율적일 것이라 판단.
	txHashMap map[string]*list.Element
	mutex     sync.Mutex
}

func NewtransactionPool(txdb db.Bucket) *transactionPool {
	return &transactionPool{txdb: txdb, txList: list.New(), txHashMap: make(map[string]*list.Element)}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// TODO: check thread safe below
func (txPool *transactionPool) runGc(expired int64) error {
	txList := txPool.txList

	for iter := txList.Front(); iter != nil; {
		if iter.Value.(*transaction).Timestamp() >= expired {
			break
		}
		tmp := iter.Next()
		delete(txPool.txHashMap, string(iter.Value.(*transaction).ID()))
		txList.Remove(iter)
		iter = tmp
	}
	return nil
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
	return ErrDuplicateTransaction if tx exists in pool
	return ErrExpiredTransaction if Timestamp of tx is expired
*/
func (txPool *transactionPool) add(tx *transaction) error {
	if tx == nil {
		return nil
	}
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	var err error
	txList := txPool.txList
	if txList.Len() >= txPoolSize {
		return ErrTransactionPoolOverFlow
	}

	// check whether this transaction is already in txPool
	if _, ok := txPool.txHashMap[string(tx.ID())]; ok {
		// TODO: 추가적으로 address, nonce까지 검사할 필요가 있을까?
		return ErrDuplicateTransaction
	}

	element := txList.PushBack(tx)
	txPool.txHashMap[string(tx.ID())] = element

	return err
}

// It returns all candidates for a negative integer n.
func (txPool *transactionPool) candidate(wc WorldContext, max int) []module.Transaction {
	// TODO state를 전달받더라도 실제 account info는 address를 통해서 바로 찾는 것이
	// 유리할텐데... trie를 통해서 Get하면 비효율적임.
	// TODO max가 음수이면 모든 transaction을 리턴한다. patch pool에 대해서 필요할 것
	// 같음.
	// TODO validate 작업도 필요.
	// TODO ServiceManager에 하나의 pool을 관리하고 candidate를 구할 때 transition
	// 기반으로 사용된 적이 있는 것을 제외하는 방식으로 구현하려고 하는데, unfinalized
	// branch가 긴 것을 감안하면 좀 더 효과적인 구현이 있을지 고민 필요
	txPool.mutex.Lock()

	if txPool.txList.Len() == 0 {
		txPool.mutex.Unlock()
		return []module.Transaction{}
	}

	txsLen := 0
	if max < 0 {
		txsLen = txPool.txList.Len()
	} else {
		if txPool.txList.Len() < max {
			txsLen = txPool.txList.Len()
		} else {
			txsLen = max
		}
	}

	txs := make([]Transaction, txsLen)
	txsIndex := 0
	for iter := txPool.txList.Front(); iter != nil; {

		txs[txsIndex] = iter.Value.(Transaction)
		txsIndex++
		if txsIndex == max {
			break
		}
		iter = iter.Next()
	}

	txPool.mutex.Unlock()
	validTxs := make([]module.Transaction, txsIndex)
	index := 0
	for i, tx := range txs[:txsIndex] {
		if err := tx.(Transaction).PreValidate(wc, true); err == nil {
			validTxs[index] = tx
			txs[i] = nil
			index++
		}
	}
	if index != txsIndex {
		go func() {
			txPool.mutex.Lock()
			defer txPool.mutex.Unlock()
			for _, tx := range txs {
				if tx != nil {
					if v, ok := txPool.txHashMap[string(tx.ID())]; ok {
						txPool.txList.Remove(v)
						delete(txPool.txHashMap, string(tx.ID()))
					}
				}
			}
		}()
	}

	return validTxs[:index]
}

// return true if one of txs is added to pool
//func (txPool *transactionPool) addList(txs []*transaction) error {
//	expired := makeTimestamp() - txLiveDuration
//	txPool.mutex.Lock()
//	defer txPool.mutex.Unlock()
//
//	addTxs := append([]*transaction{}, txs...)
//	sort.Slice(addTxs, func(i, j int) bool {
//		return addTxs[i].Timestamp() > addTxs[j].Timestamp()
//	})
//
//	txList := txPool.txList
//
//	if configOnCheckingTimestamp {
//		if iter := txList.Front(); iter != nil {
//			if iter.Value.(*transaction).Timestamp() < expired {
//				txPool.runGc(expired)
//			}
//		}
//	}
//
//	var err error
//	if txList.Len() >= txPoolSize {
//		return ErrTransactionPoolOverFlow
//	}
//
//	// check whether this transaction is already in txPool
//	revIter := txList.Back()
//	for _, addTx := range addTxs {
//		if _, ok := txPool.txHashMap[string(addTx.ID())]; ok {
//			// TODO: 추가적으로 address, nonce까지 검사할 필요가 있을까?
//			//fmt.Println("drop ID = ", addTx.ID(), ", timestamp = ", addTx.TimeStamp())
//			err = ErrDuplicateTransaction
//			continue
//		}
//		if configOnCheckingTimestamp {
//			if addTx.Timestamp() < expired {
//				err = ErrExpiredTransaction
//				continue
//			}
//		}
//
//		inserted := false
//		for revIter != nil {
//			tx := revIter.Value.(*transaction)
//			if tx.Timestamp() <= addTx.Timestamp() {
//				revIter = txList.InsertAfter(addTx, revIter)
//				txPool.txHashMap[string(addTx.ID())] = addTx
//				inserted = true
//				break
//			}
//			revIter = revIter.Prev()
//		}
//
//		if inserted == false {
//			txList.PushFront(addTx)
//			txPool.txHashMap[string(addTx.ID())] = addTx
//		}
//	}
//
//	return err
//}

// finalize할 때 호출됨.
func (txPool *transactionPool) removeList(txs module.TransactionList) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	if txPool.txList.Len() == 0 {
		return
	}

	for i := txs.Iterator(); i.Has(); i.Next() {
		t, _, err := i.Get()
		if err != nil {
			log.Printf("Failed to get transaction from iterator\n")
			continue
		}
		if tx, ok := t.(*transaction); ok {
			if v, ok := txPool.txHashMap[string(tx.ID())]; ok {
				txPool.txList.Remove(v)
				delete(txPool.txHashMap, string(tx.ID()))
			}
		} else {
			log.Printf("Failed type assertion to transaction. t = %v\n", t)
		}
	}
}
