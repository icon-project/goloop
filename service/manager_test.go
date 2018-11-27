package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
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
	From      common.Address  `json:"from"`
	To        common.Address  `json:"to"`
	Value     common.HexUint  `json:"value"`
	Fee       common.HexUint  `json:"fee"`
	Timestamp common.HexInt64 `json:"timestamp"`
	Nonce     common.HexInt64 `json:"nonce"`
	Signature common.HexBytes `json:"signature"`
}

type txTestV3 struct {
	Version   common.HexInt16 `json:"version"`
	From      common.Address  `json:"from"`
	To        common.Address  `json:"to"`
	Value     common.HexUint  `json:"value"`
	StepLimit common.HexUint  `json:"stepLimit"`
	Timestamp common.HexInt64 `json:"timestamp"`
	Nid       common.HexInt16 `json:"nid"`
	Nonce     common.HexInt64 `json:"nonce"`
	Signature common.HexBytes `json:"signature"`
}

func (tx *txTest) ToJSON(version int) (interface{}, error) {
	panic("implement me")
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
var testAddresses [TEST_ACCOUNTS_NUM]module.Address
var testWallets [TEST_ACCOUNTS_NUM]module.Wallet

func createTxInst(wallet module.Wallet, to module.Address, value *big.Int, timestamp int64) module.Transaction {
	r := rand.Int63()
	ver := int((r % 2) + 2)

	tx := txTest{}
	tx.group = module.TransactionGroupNormal
	tx.version = ver
	tx.from = wallet.Address().(*common.Address)
	tx.to = to.(*common.Address)
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
		m["value"] = common.HexUint{*tx.value}.String()
		m["fee"] = common.HexUint{*tx.stepLimit}.String()
		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
		m["nonce"] = common.HexInt64{tx.nonce}.String()
	case 3:
		m["version"] = common.HexInt16{int16(tx.version)}.String()
		m["from"] = tx.from.String()
		m["to"] = tx.to.String()
		m["value"] = common.HexUint{*tx.value}.String()
		m["stepLimit"] = common.HexUint{*tx.stepLimit}.String()
		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
		m["nid"] = common.HexInt16{int16(tx.nid)}.String()
		m["nonce"] = common.HexInt64{tx.nonce}.String()
	default:
		log.Fatalln("unknown transaction version:", tx.version)
	}
	bs, err := SerializeMap(m, map[string]bool(nil), map[string]bool(nil))
	if err != nil {
		log.Fatalln("fail to create transaction bytes")
	}
	bs = append([]byte("icx_sendTransaction."), bs...)
	h := crypto.SHA3Sum256(bs)
	sig, err := wallet.Sign(h)
	if err != nil {
		log.Fatalln("fail to create a signature")
	}
	tx.signature = sig

	// create bytes
	tx.bytes = marshalTx(&tx)
	return &tx
}

func marshalTx(tx *txTest) []byte {
	var i interface{}
	switch tx.version {
	case 2:
		ti := txTestV2{
			From:      *tx.from,
			To:        *tx.to,
			Value:     common.HexUint{*tx.value},
			Fee:       common.HexUint{*tx.stepLimit},
			Timestamp: common.HexInt64{tx.timestamp},
			Nonce:     common.HexInt64{tx.nonce},
			Signature: tx.signature,
		}
		i = ti
	case 3:
		ti := txTestV3{
			Version:   common.HexInt16{int16(tx.version)},
			From:      *tx.from,
			To:        *tx.to,
			Value:     common.HexUint{*tx.value},
			StepLimit: common.HexUint{*tx.stepLimit},
			Timestamp: common.HexInt64{tx.timestamp},
			Nid:       common.HexInt16{int16(tx.nid)},
			Nonce:     common.HexInt64{tx.nonce},
			Signature: tx.signature,
		}
		i = ti
	}
	b, _ := json.Marshal(&i)
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
		timestamp = time - TestTxLiveDuration() - 1000 - int64(rand.Int()%10)
	}

	return createTxInst(walletFrom, to, value, timestamp)
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
	for i := 0; i < testWalletNum; i++ {
		testWallets[i] = common.NewWallet()
		testAddresses[i] = testWallets[i].Address()
		accountState := newAccountState(db, nil)
		accountState.SetBalance(big.NewInt(deposit))
		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())
		for _, mpt := range mpts {
			mpt.Set(testAddresses[i].Bytes(), serializedAccount)
		}
	}
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
	leaderServiceManager := NewManager(leaderDB)
	go requestTx(TEST_VALID_REQUEST_TX_NUM, leaderServiceManager, requestCh)

	//run service manager for leader
	snapshot := leaderTrie.GetSnapshot()
	snapshot.Flush()
	leaderResult := make([]byte, 96)
	copy(leaderResult, snapshot.Hash())
	leaderValidator, _ := ValidatorListFromSlice(leaderDB, nil)
	initTrs, err := leaderServiceManager.CreateInitialTransition(leaderResult, leaderValidator, 0)
	if err != nil {
		log.Panicf("Failed to create initial transition. result = %x, err = %s\n", leaderResult, err)
	}
	parentTrs := initTrs
	txListChan := make(chan module.TransactionList)
	// propose transition
	go func() {
		for {
			executedTxNum := 0

			transition, err := leaderServiceManager.ProposeTransition(parentTrs)
			if err != nil {
				log.Panicf("Failed to propose transition!, err = %s\n", err)
			}
			txList := transition.NormalTransactions()
			for iter := txList.Iterator(); iter.Has(); iter.Next() {
				executedTxNum += 1
			}
			txListChan <- txList
			<-txListChan
			cb := &transitionCb{exeDone: make(chan bool)}
			transition.Execute(cb)
			<-cb.exeDone
			leaderServiceManager.Finalize(transition, module.FinalizeNormalTransaction)
			leaderServiceManager.Finalize(transition, module.FinalizeResult)
			// get result then run below
			transition = parentTrs
			if executedTxNum >= TEST_VALID_REQUEST_TX_NUM {
				log.Printf("Proposed transactions %d\n", executedTxNum)
				return
			}
		}
	}()

	// validator
	validatorSnapshot := validatorTrie.GetSnapshot()
	validatorSnapshot.Flush()
	validatorResult := make([]byte, 96)
	copy(validatorResult, validatorSnapshot.Hash())
	validatorServiceManager := NewManager(validatorDB)
	validatorValidator, _ := ValidatorListFromSlice(leaderDB, nil)
	initVTrs, err := validatorServiceManager.CreateInitialTransition(validatorResult, validatorValidator, 0)
	if err != nil {
		log.Panicf("Failed to create initial transition. result = %x, err = %s\n", validatorResult, err)
	}
	parentVTransition := initVTrs
	endChan := make(chan bool)
	go func() {
		executedTxNum := 0
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
		log.Panicf("Failed to compare hashes. leadHash : %x, validatorHash : %x\n",
			leaderSanpshot.Hash(), validatorSnapdhot.Hash())
	}

	for k, v := range resultMap {
		leaderSanpshot.Get([]byte(k))
		if serializedAccount, err := leaderSanpshot.Get([]byte(k)); err == nil && len(serializedAccount) != 0 {
			var accInfo accountSnapshotImpl
			if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &accInfo); err != nil {
				log.Panicf("Failed to unmarshal")
			}
			if accInfo.GetBalance().Cmp(v) != 0 {
				log.Panicf("Not same value for %x, trie %v, map %v \n",
					[]byte(k), accInfo.GetBalance(), v)
			}
		}
	}
}
