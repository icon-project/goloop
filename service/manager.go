package service

import (
	"errors"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/trie/trie_manager"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

const (
	// maximum number of transactions in a block
	// TODO it should be configured or received from block manager
	txMaxNumInBlock = 10
)

////////////////////
// Service Manager
////////////////////

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	patchTxPool  *transactionPool
	normalTxPool *transactionPool

	db db.Database
}

// TODO It should be declared in module package.
func NewManager(db db.Database) module.ServiceManager {
	return &manager{
		patchTxPool:  new(transactionPool),
		normalTxPool: new(transactionPool),
		db:           db,
	}
}

// ProposeTransition proposes a Transition following the parent Transition.
// parent transition should have a valid result.
// Returned Transition always passes validation.
func (m *manager) ProposeTransition(parent module.Transition) (module.Transition, error) {
	// check validity of transition
	pt, state, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	// find validated transactions
	patchTxs := m.patchTxPool.candidate(state, -1) // try to add all patches in the block
	maxTxNum := txMaxNumInBlock - len(patchTxs)
	var normalTxs []*transaction
	if maxTxNum > 0 {
		normalTxs = m.normalTxPool.candidate(state, txMaxNumInBlock-len(patchTxs))
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]*transaction, 0)
	}

	// create transition instance and return it
	return newTransition(pt,
			newTransactionList(m.db, patchTxs),
			newTransactionList(m.db, normalTxs),
			state,
			true),
		nil
}

// ProposeGenesisTransition proposes a Transition for Genesis
// with transactions of Genesis.
func (m *manager) ProposeGenesisTransition(parent module.Transition) (module.Transition, error) {
	if pt, ok := parent.(*transition); ok {
		// create transition instance and return it
		return newTransition(pt,
				newTransactionList(m.db, nil),
				newTransactionList(m.db, nil),
				trie_manager.NewMutable(m.db, nil),
				true),
			nil
	}
	return nil, common.ErrIllegalArgument
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte, valList module.ValidatorList, height int64) (module.Transition, error) {
	if result == nil {
		if height < 0 {
			// nil result is allowed only at height -1 (prior to Genesis)
			return newInitTransition(m.db, nil, valList), nil
		} else {
			return nil, common.ErrIllegalArgument
		}
	}

	resultBytes, err := newResultBytes(result)
	if err != nil {
		return nil, errors.New("Invalid result")
	}
	// TODO check if result isn't valid. Who's responsible?
	return newInitTransition(m.db, resultBytes, valList), nil
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
func (m *manager) CreateTransition(parent module.Transition, txList module.TransactionList) (module.Transition, error) {
	// check validity of transition
	pt, state, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	// check transaction type
	txlist, ok := txList.(*transactionlist)
	if !ok {
		return nil, common.ErrIllegalArgument
	}

	return newTransition(pt,
			newTransactionList(m.db, make([]*transaction, 0)),
			txlist,
			state,
			false),
		nil
}

// GetPatches returns all patch transactions based on the parent transition.
// If it doesn't have any patches, it returns nil.
func (m *manager) GetPatches(parent module.Transition) module.TransactionList {
	// In fact, state is not necessary for patch transaction candidate validation,
	// but add the following same as that of normal transaction.
	pt, ok := parent.(*transition)
	if !ok {
		return nil
	}
	_, state, err := m.checkTransitionResult(pt.parent)
	if err != nil {
		return nil
	}

	return newTransactionList(m.db, m.patchTxPool.candidate(state, -1))
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patchTxList module.TransactionList) module.Transition {
	// type checking
	pt, ok := t.(*transition)
	if !ok {
		return nil
	}
	tst, state, err := m.checkTransitionResult(pt.parent)
	if err != nil {
		return nil
	}

	// prepare patch transaction list
	var txList *transactionlist
	if patchTxList == nil {
		txList = newTransactionList(m.db, make([]*transaction, 0))
	} else {
		txList, ok = patchTxList.(*transactionlist)
		if !ok {
			return nil
		}
	}

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return newTransition(tst.parent, txList, tst.normalTransactions, state, false)
}

// Finalize finalizes data related to the transition. It usually stores
// data to a persistent storage. opt indicates which data are finalized.
// It should be called for every transition.
func (m *manager) Finalize(t module.Transition, opt int) {
	if tst, ok := t.(*transition); ok {
		tst.finalize(opt)
	}
}

// TransactionFromBytes returns a Transaction instance from bytes.
func (m *manager) TransactionFromBytes(b []byte, blockVersion int) module.Transaction {
	tx, _ := newTransaction(b)
	return tx
}

// TransactionListFromHash returns a TransactionList instance from
// the hash of transactions or nil when no transactions exist.
func (m *manager) TransactionListFromHash(hash []byte) module.TransactionList {
	// TODO nil if hash is invalid?
	return newTransactionListFromHash(m.db, hash)
}

// TransactionListFromSlice returns list of transactions.
func (m *manager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	// TODO What if transaction objects are created outside?
	// TODO: db should be passed as parameter for flush()
	//panic("not implemented")
	return newTransactionListFromList(m.db, txs)
}

// ReceiptFromTransactionID returns receipt from legacy receipt bucket.
func (m *manager) ReceiptFromTransactionID(id []byte) module.Receipt {
	return nil
}

// ReceiptListFromResult returns list of receipts from result.
func (m *manager) ReceiptListFromResult(result []byte, g module.TransactionGroup) module.ReceiptList {
	return nil
}

func (m *manager) checkTransitionResult(t module.Transition) (*transition, trie.Mutable, error) {
	// check validity of transition
	tst, ok := t.(*transition)
	if !ok || tst.step != stepComplete {
		return nil, nil, common.ErrIllegalArgument
	}
	state := trie_manager.NewMutable(m.db, tst.result.stateHash())

	return tst, state, nil
}

func (m *manager) SendTransaction(tx module.Transaction) ([]byte, error) {
	newTx, err := newTransactionFromObject(tx)
	if err != nil {
		log.Printf("Failed to create new transaction from object!. tx : %x\n", newTx.Bytes())
		return nil, err
	}
	if err = newTx.Verify(); err != nil {
		log.Printf("Failed to verify transaction. tx : %x\n", newTx.Bytes())
		return nil, err
	}
	hash := newTx.Hash()
	if hash == nil {
		log.Println("Failed to get hash from tx : %x\n", newTx.Bytes())
		return nil, errors.New("Invalid Transaction. Failed to get hash")
	}

	var txPool *transactionPool
	switch newTx.Group() {
	case module.TransactionGroupNormal:
		txPool = m.normalTxPool
	case module.TransactionGroupPatch:
		txPool = m.patchTxPool
	default:
		log.Panicf("Wrong TransactionGroup. %v", newTx.Group())
	}

	go txPool.add(newTx)
	return hash, nil
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	return nil
}

// test case
// TODO: below test case has to be moved to manager_test.go
var resultMap = make(map[string]*big.Int)
var nameNum = 10
var nameList = []string{
	"KANG DONG WON",
	"JANG DONG GUN",
	"LEE HYO RI",
	"KELVIN DURANT",
	"STEPHEN CURRY",
	"LEBRON JAMES",
	"MICHEAL JORDAN",
	"PATRICK EWING",
	"HAKIM OLAJUWON",
	"CHARLES BARKLEY",
}

var toNum = 17
var toList = []string{
	"KANG DONG WON",
	"JANG DONG GUN",
	"LEE HYO RI",
	"KELVIN DURANT",
	"STEPHEN CURRY",
	"LEBRON JAMES",
	"MICHEAL JORDAN",
	"PATRICK EWING",
	"HAKIM OLAJUWON",
	"CHARLES BARKLEY",
	"NO MARRY",
	"NO TOM",
	"NO JERRY",
	"NO COOLER",
	"NO MACHINE",
	"NO ANGEL",
	"NO DEVIL",
}
var addresses [10]common.Address
var deposit = int64(1000000)

func TxTest() {
	// sendTx.
	target := 100

	database := db.NewMapDB()
	trieManager := trie_manager.New(database)
	mutableTrie := trieManager.NewMutable(nil)

	for i, name := range nameList {
		resultMap[name] = big.NewInt(deposit)
		accountState := newAccountState(database, nil)
		accountState.SetBalance(big.NewInt(deposit))
		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())

		var accInfo accountSnapshotImpl
		if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &accInfo); err != nil {
			log.Println("err is nil")
		}

		addresses[i] = *common.NewAccountAddress([]byte(name))
		mutableTrie.Set(addresses[i].Bytes(), serializedAccount)
	}
	txdb, _ := database.GetBucket(db.TransactionLocatorByHash)
	manager := &manager{
		patchTxPool:  NewtransactionPool(txdb),
		normalTxPool: NewtransactionPool(txdb),
		db:           database}
	requestDone := make(chan bool)
	exeDone := make(chan bool)
	go txRequest(target, manager, requestDone)

	go txExecute(manager, target, exeDone, mutableTrie)
	<-requestDone
	<-exeDone
	// execute
	// waiting for end of execute and request
	totalBalance := big.NewInt(int64(nameNum) * deposit)
	calcTotalBal := big.NewInt(0)
	for _, name := range toList {
		serializedAccount, _ := mutableTrie.Get(common.NewAccountAddress([]byte(name)).Bytes())
		var accInfo accountSnapshotImpl
		//var accInfo accountInfo
		codec.MP.UnmarshalFromBytes(serializedAccount, &accInfo)
		log.Println("[", name, "] has ", accInfo.GetBalance())
		calcTotalBal.Add(calcTotalBal, accInfo.GetBalance())
	}
	if totalBalance.Cmp(calcTotalBal) == 0 {
		log.Println("same total balance : ", totalBalance, ", ", calcTotalBal)

	} else {
		panic("different")
	}
}

// true if valid transaction
func makeTransaction(valid bool, time int64, validNum int) *transaction {
	tx := &transaction{
		stepLimit: big.NewInt(10),
	}
	id := rand.Int() % toNum
	//tx.hash = []byte{id}
	// valid 하도록 만든다. 기존에 없는 ID, time 등을 이용하도록.
	// insert transaction to valid transaction (expected txPool).
	// ID map, time map 사용.
	// 중복될 경우 새로운 ID, time을 생성한다.
	toId := rand.Int() % toNum
	for toId == id {
		toId = rand.Int() % toNum
	}
	//tx.to = addresses[toId]
	tx.from = *common.NewAccountAddress([]byte(toList[id]))
	tx.to = *common.NewAccountAddress([]byte(toList[toId]))
	tx.value = big.NewInt(int64(rand.Int() % 300000))
	tx.bytes = tx.to.Bytes()
	tx.hash = tx.value.Bytes()
	tx.group = module.TransactionGroupNormal

	if valid {
		// TODO: 먼저 from에서 이체 가능금액 확인 & 이체
		balance := resultMap[toList[id]]
		if balance != nil && balance.Cmp(tx.value) > 0 {
			resultMap[toList[id]] = balance.Mul(balance, tx.value)
			if _, ok := resultMap[toList[toId]]; ok == false {
				resultMap[toList[toId]] = big.NewInt(0)
			}
			resultMap[toList[toId]].Add(resultMap[toList[toId]], tx.value)
		}

		tx.timestamp = time + 1000 + int64(rand.Int()%100)
		// TODO: ADD verify
		return tx
	}
	// invalid하도록 만든다.
	// ID를 map에서 가져다가 쓰거나 전달받은 시간보다 작은 시간을 설정한다.
	// 처음에 진입하여 ID가 없을 경우 time을 설정한다.
	// sleep을 줄까...
	// TODO: ADD verify
	tx.timestamp = time - txLiveDuration - 1000 - int64(rand.Int()%10)
	return tx
}

func txRequest(validTxNum int, manager module.ServiceManager, done chan bool) {
	txMap := map[bool]int{}
	for validTxNum > 0 {
		curTime := makeTimestamp()
		validTx := rand.Int()%2 == 0
		tx := makeTransaction(validTx, curTime, validTxNum)

		txMap[validTx]++
		if validTx {
			validTxNum--
		}

		manager.SendTransaction(tx)
		time.Sleep(time.Millisecond * 3) // 0.003 seconds
	}
	log.Println("invalid tx Num : ", txMap[false], ", valid tx Num : ", txMap[true])
	done <- true

	// TODO: send signal for end of request
}

func txExecute(manager *manager, txNum int, done chan bool, mutableTrie trie.Mutable) error {
	txPool := manager.normalTxPool
	for txNum > 0 {
		maxNum := 10
		candidateList := txPool.candidate(nil, maxNum)

		if listLen := len(candidateList); listLen > maxNum {
			return errors.New("candidateList is longer than MaxNum:w")
		} else if listLen == 0 {
			// sleep
			time.Sleep(time.Millisecond * 1) // 0.001 seconds
		}
		for _, v := range candidateList {
			state := &transitionState{state: mutableTrie}
			v.execute(state, manager.db)
		}
		txPool.removeList(candidateList)
		txNum -= len(candidateList)
	}
	done <- true
	return nil
}
