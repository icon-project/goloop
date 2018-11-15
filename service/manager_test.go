package service_test

import (
	"encoding/json"
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
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
	from      *common.Address
	to        *common.Address
	value     *big.Int
	stepLimit *big.Int
	timestamp int64
	nid       int
	nonce     int64
	signature []byte

	hash  []byte
	bytes []byte
}

type txTestV2 struct {
	from      common.Address  `json:"from"`
	to        common.Address  `json:"to"`
	value     common.HexInt   `json:"value"`
	fee       common.HexInt   `json:"fee"`
	timestamp common.HexInt64 `json:"timestamp"`
	nonce     common.HexInt64 `json:"nonce"`
	signature common.HexBytes `json:"signature"`
}

type txTestV3 struct {
	version   common.HexInt16 `json:"version"`
	from      common.Address  `json:"from"`
	to        common.Address  `json:"to"`
	value     common.HexInt   `json:"value"`
	stepLimit common.HexInt   `json:"stepLimit"`
	timestamp common.HexInt64 `json:"timestamp"`
	nid       common.HexInt16 `json:"nid"`
	nonce     common.HexInt64 `json:"nonce"`
	signature common.HexBytes `json:"signature"`
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
	return tx.from
}

func (tx *txTest) To() module.Address {
	return tx.to
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

var testWallets []testWallet

const (
	TEST_ACCOUNTS             = 17
	TEST_VALID_REQUEST_TX_NUM = 100
)

type testWallet struct {
	pbKey   *crypto.PublicKey
	prKey   *crypto.PrivateKey
	address *common.Address
}

func createWallet(walletNum int) []testWallet {
	ws := make([]testWallet, walletNum)
	for i := 0; i < walletNum; i++ {
		w := testWallet{}
		w.prKey, w.pbKey = crypto.GenerateKeyPair()
		w.address = common.NewAccountAddressFromPublicKey(w.pbKey)
		ws[i] = w
	}
	return ws
}

func createTxInst(wallet *testWallet, to *common.Address, value *big.Int, timestamp int64) module.Transaction {
	r := rand.Int63()
	ver := int((r % 2) + 2)

	tx := txTest{}
	tx.group = module.TransactionGroupNormal
	tx.version = ver
	tx.from = wallet.address
	tx.to = to
	tx.value = value
	tx.stepLimit = new(big.Int).SetInt64(r % 0xffff)
	tx.timestamp = timestamp
	tx.nid = 0
	tx.nonce = r

	// create a signature
	m := make(map[string]interface{})
	switch tx.version {
	case 2:
		m["from"] = *tx.from
		m["to"] = *tx.to
		m["value"] = common.HexInt{*tx.value}
		m["fee"] = common.HexInt{*tx.stepLimit}
		m["nonce"] = common.HexInt64{tx.nonce}
	case 3:
		m["version"] = common.HexInt16{int16(tx.version)}
		m["from"] = *tx.from
		m["to"] = *tx.to
		m["value"] = common.HexInt{*tx.value}
		m["stepLimit"] = common.HexInt{*tx.stepLimit}
		m["nonce"] = common.HexInt64{tx.nonce}
	default:
		log.Fatalln("unknown transaction version:", tx.version)
	}
	bs, err := service.SerializeMap(m, map[string]bool(nil), map[string]bool(nil))
	if err != nil {
		log.Fatalln("fail to create transaction bytes")
	}
	bs = append([]byte("icx_sendTransaction."), bs...)
	h := crypto.SHA3Sum256(bs)
	sig, err := crypto.NewSignature(h, wallet.prKey)
	if err != nil {
		log.Fatalln("fail to create a signature")
	}
	tx.signature, _ = sig.SerializeRSV()

	// create bytes
	tx.bytes = marshalTx(&tx)

	return &tx
}

func marshalTx(tx *txTest) []byte {
	var i interface{}
	switch tx.version {
	case 2:
		ti := &txTestV2{
			from:      *tx.from,
			to:        *tx.to,
			value:     common.HexInt{*tx.value},
			fee:       common.HexInt{*tx.stepLimit},
			timestamp: common.HexInt64{tx.timestamp},
			nonce:     common.HexInt64{tx.nonce},
			signature: tx.signature,
		}
		i = ti
	case 3:
		ti := &txTestV3{
			version:   common.HexInt16{int16(tx.version)},
			from:      *tx.from,
			to:        *tx.to,
			value:     common.HexInt{*tx.value},
			stepLimit: common.HexInt{*tx.stepLimit},
			timestamp: common.HexInt64{tx.timestamp},
			nid:       common.HexInt16{int16(tx.nid)},
			nonce:     common.HexInt64{tx.nonce},
			signature: tx.signature,
		}
		i = ti
	}
	b, _ := json.Marshal(i)
	return b
}

//const txLiveDuration = int64(60 * time.Second / time.Millisecond) // 60 seconds in millisecond
//// true if valid transaction
func createRandTx(valid bool, time int64, validNum int) module.Transaction {
	idFrom := rand.Int() % toNum
	//tx.hash = []byte{id}
	// valid 하도록 만든다. 기존에 없는 ID, time 등을 이용하도록.
	// insert transaction to valid transaction (expected txPool).
	// ID map, time map 사용.
	// 중복될 경우 새로운 ID, time을 생성한다.
	idTo := rand.Int() % toNum
	for idTo == idFrom {
		idTo = rand.Int() % toNum
	}
	//tx.to = addresses[toId]
	fromString := testWallets[idFrom].address.String()
	toString := testWallets[idTo].address.String()
	walletFrom := testWallets[idTo]
	addrTo := testWallets[idTo].address
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
		return createTxInst(&walletFrom, addrTo, value, timestamp)
	}
	//invalid하도록 만든다.
	// ID를 map에서 가져다가 쓰거나 전달받은 시간보다 작은 시간을 설정한다.
	// 처음에 진입하여 ID가 없을 경우 time을 설정한다.
	// sleep을 줄까...
	// TODO: ADD verify
	timestamp := time - service.TestTxLiveDuration() - 1000 - int64(rand.Int()%10)
	return createTxInst(&walletFrom, addrTo, value, timestamp)
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
	testWallets = createWallet(TEST_ACCOUNTS)
	for i := 0; i < TEST_ACCOUNTS; i++ {
		accountState := service.TestNewAccountState(db)
		accountState.SetBalance(big.NewInt(deposit))
		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())
		addr := testWallets[i].address
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
