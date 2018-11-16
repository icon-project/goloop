package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
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
}

func (ts *transitionCb) OnExecute(module.Transition, error) {
	ts.exeDone <- true
}

// test case
const (
	TEST_ACCOUNTS_NUM         = 10
	TEST_VALID_REQUEST_TX_NUM = 100
)

var resultMap = make(map[string]*big.Int)
var deposit = int64(1000000)
var testAddresses [TEST_ACCOUNTS_NUM]*common.Address
var testWallets []testWallet

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

	// TODO find a better way to JSON serialization
	// create a signature
	m := make(map[string]interface{})
	switch tx.version {
	case 2:
		m["from"] = tx.from.String()
		m["to"] = tx.to.String()
		m["value"] = common.HexInt{*tx.value}.String()
		m["fee"] = common.HexInt{*tx.stepLimit}.String()
		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
		m["nonce"] = common.HexInt64{tx.nonce}.String()
	case 3:
		m["version"] = common.HexInt16{int16(tx.version)}.String()
		m["from"] = tx.from.String()
		m["to"] = tx.to.String()
		m["value"] = common.HexInt{*tx.value}.String()
		m["stepLimit"] = common.HexInt{*tx.stepLimit}.String()
		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
		m["nonce"] = common.HexInt64{tx.nonce}.String()
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
	idFrom := rand.Int() % TEST_ACCOUNTS_NUM
	//tx.hash = []byte{id}
	// valid 하도록 만든다. 기존에 없는 ID, time 등을 이용하도록.
	// insert transaction to valid transaction (expected txPool).
	// ID map, time map 사용.
	// 중복될 경우 새로운 ID, time을 생성한다.
	idTo := rand.Int() % TEST_ACCOUNTS_NUM
	for idTo == idFrom {
		idTo = rand.Int() % TEST_ACCOUNTS_NUM
	}
	//tx.to = addresses[toId]
	stringFrom := string(testAddresses[idFrom].Bytes())
	stringTo := string(testAddresses[idTo].Bytes())
	walletFrom := testWallets[idFrom]
	to := testAddresses[idTo]
	value := big.NewInt(int64(rand.Int() % 300000))

	var timestamp int64
	if valid {
		// TODO: 먼저 from에서 이체 가능금액 확인 & 이체
		balance := resultMap[stringFrom]
		if balance != nil && balance.Cmp(value) > 0 {
			resultMap[stringFrom] = balance.Mul(balance, value)
			if _, ok := resultMap[stringTo]; ok == false {
				resultMap[stringTo] = big.NewInt(0)
			}
			resultMap[stringTo].Add(resultMap[stringTo], value)
		}

		timestamp = time + 1000 + int64(rand.Int()%100)
		// TODO: check value type. no
	} else {
		timestamp = time - service.TestTxLiveDuration() - 1000 - int64(rand.Int()%10)
	}

	return createTxInst(&walletFrom, to, value, timestamp)
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

// create wallet(private/public keys & address) and set balance
// then set addresses and accounts to trie
func initTestWallets(testWalletNum int, db db.Database, mpts ...trie.Mutable) {
	wallet := createWallet(testWalletNum)
	//testAddresses = make([][]byte, testWalletNum)
	for i := 0; i < testWalletNum; i++ {
		testAddresses[i] = wallet[i].address
		accountState := service.T_NewAccountState(db)
		accountState.SetBalance(big.NewInt(deposit))
		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())
		for _, mpt := range mpts {
			mpt.Set(testAddresses[i].Bytes(), serializedAccount)
		}
	}
	testWallets = wallet
}

func TestServiceManager(t *testing.T) {
	// initialize leader trie
	leaderDB := db.NewMapDB()
	leaderTrie := trie_manager.NewMutable(leaderDB, nil)

	// initialize validator trie
	validatorDB := db.NewMapDB()
	validatorTrie := trie_manager.NewMutable(validatorDB, nil)

	// initialize wallets for test and set default balance and apply it to trie
	initTestWallets(TEST_ACCOUNTS_NUM, leaderDB, leaderTrie, validatorTrie)

	// request transactions
	requestCh := make(chan bool)
	leaderServiceManager := service.NewManager(leaderDB)
	go requestTx(TEST_VALID_REQUEST_TX_NUM, leaderServiceManager, requestCh)

	//run service manager for leader
	snapshot := leaderTrie.GetSnapshot()
	snapshot.Flush()
	leaderResult := make([]byte, 64)
	copy(leaderResult, snapshot.Hash())
	initTrs, err := leaderServiceManager.CreateInitialTransition(leaderResult, nil, 0)
	if err != nil {
		log.Panicf("Faile to create initial transition. result = %x, err = %s\n", leaderResult, err)
	}
	parentTrs := initTrs
	txListChan := make(chan module.TransactionList)
	// propose transition
	go func() {
		for {
			transition, err := leaderServiceManager.ProposeTransition(parentTrs)
			if err != nil {
				log.Panicf("Failed to propose transition!, err = %s\n", err)
			}
			txListChan <- transition.NormalTransactions()
			<-txListChan
			cb := &transitionCb{exeDone: make(chan bool)}
			transition.Execute(cb)
			leaderServiceManager.Finalize(transition, module.FinalizeNormalTransaction)
			leaderServiceManager.Finalize(transition, module.FinalizeResult)
			// get result then run below
			<-cb.exeDone
			transition = parentTrs
			// TODO when is done?
		}
	}()

	// validator
	validatorSnapshot := validatorTrie.GetSnapshot()
	validatorSnapshot.Flush()
	validatorResult := make([]byte, 64)
	copy(validatorResult, validatorSnapshot.Hash())
	validatorServiceManager := service.NewManager(validatorDB)
	initVTrs, err := validatorServiceManager.CreateInitialTransition(validatorResult, nil, 0)
	if err != nil {
		log.Panicf("Faile to create initial transition. result = %x, err = %s\n", validatorResult, err)
	}
	parentVTransition := initVTrs
	executedTxNum := 0
	endChan := make(chan bool)
	go func() {
		for {
			txList := <-txListChan
			for iter := txList.Iterator(); iter.Has(); iter.Next() {
				executedTxNum += 1
			}
			vTransition, err := validatorServiceManager.CreateTransition(parentVTransition, txList)
			if err != nil {
				log.Panicf("Failed to create transition for validator")
			}
			cb := &transitionCb{exeDone: make(chan bool)}
			vTransition.Execute(cb)
			<-cb.exeDone
			validatorServiceManager.Finalize(vTransition, module.FinalizeNormalTransaction)
			validatorServiceManager.Finalize(vTransition, module.FinalizeResult)
			parentVTransition = vTransition
			if executedTxNum >= TEST_VALID_REQUEST_TX_NUM {
				endChan <- true
			}
			txListChan <- nil
		}
	}()

	<-requestCh
	<-endChan
	leaderSanpshot := leaderTrie.GetSnapshot()
	validatorSnapdhot := validatorTrie.GetSnapshot()

	if bytes.Compare(leaderSanpshot.Hash(), validatorSnapdhot.Hash()) != 0 {
		log.Panicf("Failed to compare hashes. leadHash : %x, validatorHash : %x\n", leaderSanpshot.Hash(), validatorSnapdhot.Hash())
	}

	service.T_Result(resultMap, leaderSanpshot)
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
