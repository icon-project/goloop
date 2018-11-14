package service_test

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"log"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type txTest struct {
	group module.TransactionGroup

	version   int
	from      common.Address
	to        common.Address
	value     *big.Int
	stepLimit *big.Int
	timestamp int64
	nid       int
	nonce     int64
	signature []byte

	hash  []byte
	bytes []byte
}

func (tx *txTest) Group() module.TransactionGroup {
	return tx.group
}

func (tx *txTest) ID() []byte {
	return tx.hash
}

func (tx *txTest) Version() int {
	return tx.version
}

func (tx *txTest) Bytes() []byte {
	return tx.bytes
}

func (tx *txTest) Verify() error {
	return nil
}

func (tx *txTest) From() module.Address {
	return &tx.from
}

func (tx *txTest) To() module.Address {
	return &tx.to
}

func (tx *txTest) Value() *big.Int {
	return tx.value
}

func (tx *txTest) StepLimit() *big.Int {
	return tx.stepLimit
}

func (tx *txTest) Timestamp() int64 {
	return tx.timestamp
}

func (tx *txTest) NID() int {
	return tx.nid
}

func (tx *txTest) Nonce() int64 {
	return tx.nonce
}

func (tx *txTest) Hash() []byte {
	return tx.hash
}

func (tx *txTest) Signature() []byte {
	return tx.signature
}

type transitionCb struct {
	exeDone chan bool
}

func (ts *transitionCb) OnValidate(module.Transition, error) {
	log.Printf("OnValidate")
}

func (ts *transitionCb) OnExecute(module.Transition, error) {
	log.Printf("OnExecute")
	ts.exeDone <- true
}

// test case
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

//var addresses [10]common.Address
var deposit = int64(1000000)

var testAddresses [][]byte

const (
	TEST_ACCOUNTS             = 10
	TEST_VALID_REQUEST_TX_NUM = 100
)

type testWallet struct {
	pbKey   []byte
	prKey   []byte
	address []byte
}

// will be implemented by cw.Kwak
func createWallet(walletNum int) []testWallet {
	return []testWallet{}
}

// will be implemented by cw.Kwak
func createTxInst(from, to common.Address, value *big.Int, timestamp int64) module.Transaction {
	return nil
}

//const txLiveDuration = int64(60 * time.Second / time.Millisecond) // 60 seconds in millisecond
//// true if valid transaction
func createRandTx(valid bool, time int64, validNum int) module.Transaction {
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
	fromString := string(testAddresses[id])
	toString := string(testAddresses[toId])
	from := *common.NewAccountAddress([]byte(testAddresses[id]))
	to := *common.NewAccountAddress([]byte(toList[toId]))
	value := big.NewInt(int64(rand.Int() % 300000))

	if valid {
		// TODO: 먼저 from에서 이체 가능금액 확인 & 이체
		balance := resultMap[fromString]
		if balance != nil && balance.Cmp(value) > 0 {
			resultMap[fromString] = balance.Mul(balance, value)
			if _, ok := resultMap[toString]; ok == false {
				resultMap[toString] = big.NewInt(0)
			}
			resultMap[toString].Add(resultMap[toString], value)
		}

		timestamp := time + 1000 + int64(rand.Int()%100)
		// TODO: check value type. no
		return createTxInst(from, to, value, timestamp)
	}
	//invalid하도록 만든다.
	// ID를 map에서 가져다가 쓰거나 전달받은 시간보다 작은 시간을 설정한다.
	// 처음에 진입하여 ID가 없을 경우 time을 설정한다.
	// sleep을 줄까...
	// TODO: ADD verify
	timestamp := time - service.TestTxLiveDuration() - 1000 - int64(rand.Int()%10)
	return createTxInst(from, to, value, timestamp)
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func requestTx(validTxNum int, manager module.ServiceManager, done chan bool) {
	txMap := map[bool]int{}
	for validTxNum > 0 {
		curTime := makeTimestamp()
		validTx := rand.Int()%2 == 0
		tx := createRandTx(validTx, curTime, validTxNum)

		txMap[validTx]++
		if validTx {
			validTxNum--
		}

		manager.SendTransaction(tx)
		//time.Sleep(time.Millisecond * 3) // 0.003 seconds
	}
	fmt.Println("invalid tx Num : ", txMap[false], ", valid tx Num : ", txMap[true])
	done <- true

	// TODO: send signal for end of request
}

// create wallet(private/public keys & address) and set ballance
// then set addresses and accounts to trie
func initTestWallet(db db.Database, mpt trie.Mutable) {
	wallet := createWallet(TEST_ACCOUNTS)
	for i := 0; i < TEST_ACCOUNTS; i++ {
		testAddresses[i] = wallet[i].address
		accountState := service.TestNewAccountState(db)
		accountState.SetBalance(big.NewInt(deposit))
		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())
		addr := *common.NewAccountAddress(testAddresses[i])
		mpt.Set(addr.Bytes(), serializedAccount)
	}
}

func TestServiceManager(t *testing.T) {
	pDb := db.NewMapDB()
	pSm := service.NewManager(pDb)
	result := make([]byte, 64)
	mgr := trie_manager.New(pDb)
	mpt := mgr.NewMutable(nil)
	initTestWallet(pDb, mpt)

	// request transactions
	requestCh := make(chan bool)
	go requestTx(TEST_VALID_REQUEST_TX_NUM, pSm, requestCh)

	//Run service manager for propose
	snapshot := mpt.GetSnapshot()
	snapshot.Flush()
	copy(result, snapshot.Hash())
	initTrs, err := pSm.CreateInitialTransition(result, nil, 0)
	if err != nil {
		log.Panicf("Faile to create initial transition. result = %x, err = %s\n", result, err)
	}
	parentTrs := initTrs
	// propose transition
	for {
		trs, err := pSm.ProposeTransition(parentTrs)
		if err != nil {
			log.Panicf("Failed to propose transition!, err = %s\n", err)
		}
		cb := &transitionCb{exeDone: make(chan bool)}
		trs.Execute(cb)
		// get result then run below
		<-cb.exeDone
		trs = parentTrs
	}
	<-requestCh
	//
	// verify
	//vDb := db.NewMapDB()
	//vSm := service.NewManager(db)
	//vSm.CreateTransition(parent, txs)
	//vSm.Finalize()
	//vSm.Finalize()

	// 결과 확인
}

func TestTransaction(t *testing.T) {
	db := db.NewMapDB()
	sm := service.NewManager(db)

	var transition module.Transition
	var err error
	transition, err = sm.ProposeTransition(transition)
	if err != nil {
		panic("Failed propose transition")
		return
	}
	//cb := &transitionCb{}
	//transition.Execute(cb)
	sm.Finalize(transition, module.FinalizeNormalTransaction|module.FinalizeResult)
	//service.TestExecute()
}

func TestSendTx(t *testing.T) {
	service.TxTest()
	//service.SendTx.../
	// candidate.k
}
