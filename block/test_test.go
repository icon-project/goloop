package block

import (
	"bytes"
	"math/big"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

const gheight int64 = 0
const defaultValidators = 1

type testChain struct {
	module.Chain
	wallet   module.Wallet
	database db.Database
	gtx      *testTransaction
	gs       *testGenesisStorage
	vld      module.CommitVoteSetDecoder
	sm       *testServiceManager
	bm       module.BlockManager
}

func (c *testChain) DefaultWaitTimeout() time.Duration {
	return 0
}

func (c *testChain) MaxWaitTimeout() time.Duration {
	return 0
}

func (c *testChain) Database() db.Database {
	return c.database
}

func (c *testChain) Wallet() module.Wallet {
	return c.wallet
}

func (c *testChain) Genesis() []byte {
	return c.gtx.Bytes()
}

func (c *testChain) GenesisStorage() module.GenesisStorage {
	return c.gs
}

func (c *testChain) NID() int {
	return 1
}

func (c *testChain) CID() int {
	return 1
}

func (c *testChain) NetID() int {
	return 1
}

func (c *testChain) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return c.vld
}

func (c *testChain) BlockManager() module.BlockManager {
	return c.bm
}

func (c *testChain) ServiceManager() module.ServiceManager {
	return c.sm
}

func (c *testChain) Logger() log.Logger {
	return log.GlobalLogger()
}

type testError struct {
}

func (e *testError) Error() string {
	return "testError"
}

type testTransactionEffect struct {
	WorldState     []byte
	NextValidators *testValidatorList
	LogsBloom      txresult.LogsBloom
}

type testReceiptData struct {
	To                 *common.Address
	CumulativeStepUsed *big.Int
	StepPrice          *big.Int
	StepUsed           *big.Int
	Status             module.Status
	SCOREAddress       *common.Address
}

type testReceipt struct {
	module.Receipt
	Data testReceiptData
}

func (r *testReceipt) Bytes() []byte {
	return codec.MustMarshalToBytes(r)
}

func (r *testReceipt) To() module.Address {
	return r.Data.To
}

func (r *testReceipt) CumulativeStepUsed() *big.Int {
	return r.Data.CumulativeStepUsed
}

func (r *testReceipt) StepPrice() *big.Int {
	return r.Data.StepPrice
}

func (r *testReceipt) StepUsed() *big.Int {
	return r.Data.StepUsed
}

func (r *testReceipt) Status() module.Status {
	return r.Data.Status
}

func (r *testReceipt) SCOREAddress() module.Address {
	return r.Data.SCOREAddress
}

func (r *testReceipt) LogsBloomDisabled() bool {
	return false
}

func (r *testReceipt) Check(r2 module.Receipt) error {
	rct2, ok := r2.(*testReceipt)
	if !ok {
		return errors.Errorf("IncompatibleReceipt")
	}
	if !reflect.DeepEqual(&r.Data, &rct2.Data) {
		return errors.Errorf("DataIsn'tEqual")
	}
	return nil
}

type testTransactionData struct {
	Group                 module.TransactionGroup
	CreateError           *testError
	TransitionCreateError *testError
	ValidateError         *testError
	ExecuteError          *testError
	Effect                testTransactionEffect
	Receipt               testReceipt
	Nonce                 int64
}

type testTransaction struct {
	module.Transaction
	Data testTransactionData
}

var gnonce int64

func newTestTransaction() *testTransaction {
	tx := &testTransaction{}
	tx.Data.Group = module.TransactionGroupNormal
	tx.Data.Nonce = atomic.LoadInt64(&gnonce)
	atomic.AddInt64(&gnonce, 1)
	return tx
}

func (tx *testTransaction) Group() module.TransactionGroup {
	return tx.Data.Group
}

func (tx *testTransaction) ID() []byte {
	return crypto.SHA3Sum256(tx.Bytes())
}

func (tx *testTransaction) Bytes() []byte {
	return codec.MustMarshalToBytes(tx)
}

func (tx *testTransaction) Verify() error {
	return nil
}

func (tx *testTransaction) Version() int {
	return 0
}

func (tx *testTransaction) ValidateNetwork(nid int) bool {
	return true
}

type testTransactionIterator struct {
	*testTransactionList
	i int
}

func (it *testTransactionIterator) Has() bool {
	return it.i < len(it.Transactions)
}

func (it *testTransactionIterator) Next() error {
	if !it.Has() {
		return errors.Errorf("no more index=%v\n", it.i)
	}
	it.i++
	return nil
}

func (it *testTransactionIterator) Get() (module.Transaction, int, error) {
	if !it.Has() {
		return nil, -1, errors.Errorf("no more index=%v\n", it.i)
	}
	return it.Transactions[it.i], it.i, nil
}

type testTransactionList struct {
	module.TransactionList
	Transactions []*testTransaction
	_effect      *testTransactionEffect
	_receipts    []*testReceipt
}

func newTestTransactionList(txs []*testTransaction) *testTransactionList {
	l := &testTransactionList{}
	l.Transactions = make([]*testTransaction, len(txs))
	copy(l.Transactions, txs)
	l.updateCache()
	return l
}

func (l *testTransactionList) effect() *testTransactionEffect {
	l.updateCache()
	return l._effect
}

func (l *testTransactionList) updateCache() {
	if l._effect == nil {
		l._effect = &testTransactionEffect{}
		for _, tx := range l.Transactions {
			if tx.Data.Effect.WorldState != nil {
				l._effect.WorldState = tx.Data.Effect.WorldState
			}
			if tx.Data.Effect.NextValidators != nil {
				l._effect.NextValidators = tx.Data.Effect.NextValidators
			}
			l._effect.LogsBloom.Merge(&tx.Data.Effect.LogsBloom)
			l._receipts = append(l._receipts, &tx.Data.Receipt)
		}
	}
}

func (l *testTransactionList) Len() int {
	return len(l.Transactions)
}

func (l *testTransactionList) Get(i int) (module.Transaction, error) {
	if 0 <= i && i < len(l.Transactions) {
		return l.Transactions[i], nil
	}
	return nil, errors.Errorf("bad index=%v\n", i)
}

func (l *testTransactionList) Iterator() module.TransactionIterator {
	return &testTransactionIterator{l, 0}
}

func (l *testTransactionList) Hash() []byte {
	return crypto.SHA3Sum256(codec.MustMarshalToBytes(l))
}

func (l *testTransactionList) Equal(l2 module.TransactionList) bool {
	if tl, ok := l2.(*testTransactionList); ok {
		if len(l.Transactions) != len(tl.Transactions) {
			return false
		}
		for i := 0; i < len(l.Transactions); i++ {
			if !bytes.Equal(l.Transactions[i].Bytes(), tl.Transactions[i].Bytes()) {
				return false
			}
		}
		return true
	}
	it := l.Iterator()
	it2 := l2.Iterator()
	for {
		if it.Has() != it2.Has() {
			return false
		}
		if !it.Has() {
			return true
		}
		t, _, _ := it.Get()
		t2, _, _ := it2.Get()
		if !bytes.Equal(t.Bytes(), t2.Bytes()) {
			return false
		}
	}
}

type transitionStep int

//goland:noinspection GoUnusedConst
const (
	transitionStepUnexecuted transitionStep = iota
	transitionStepExecuting
	transitionStepSucceed
	transitionStepFailed
)

type testTransitionResult struct {
	WorldState          []byte
	PatchTXReceiptHash  []byte
	NormalTXReceiptHash []byte
}

type testTransition struct {
	module.Transition
	patchTransactions  *testTransactionList
	normalTransactions *testTransactionList
	baseValidators     *testValidatorList
	_result            []byte
	_logsBloom         *txresult.LogsBloom
	_bi                module.BlockInfo
	_csi               module.ConsensusInfo

	sync.Mutex
	step     transitionStep
	_exeChan chan struct{}
}

func (tr *testTransition) setExeChan(ch chan struct{}) {
	tr._exeChan = ch
}

func (tr *testTransition) PatchTransactions() module.TransactionList {
	tr.Lock()
	defer tr.Unlock()

	return tr.patchTransactions
}

func (tr *testTransition) NormalTransactions() module.TransactionList {
	tr.Lock()
	defer tr.Unlock()

	return tr.normalTransactions
}

func (tr *testTransition) PatchReceipts() module.ReceiptList {
	return nil
}

func (tr *testTransition) NormalReceipts() module.ReceiptList {
	return nil
}

func (tr *testTransition) EffectiveTransactions() *testTransactionList {
	if tr.patchTransactions.Len() > 0 {
		return tr.patchTransactions
	}
	return tr.normalTransactions
}

func (tr *testTransition) getErrors() (error, error) {
	for _, tx := range tr.patchTransactions.Transactions {
		if tx.Data.ValidateError != nil {
			return tx.Data.ValidateError, nil
		}
	}
	for _, tx := range tr.normalTransactions.Transactions {
		if tx.Data.ValidateError != nil {
			return tx.Data.ValidateError, nil
		}
	}
	for _, tx := range tr.patchTransactions.Transactions {
		if tx.Data.ExecuteError != nil {
			return nil, tx.Data.ExecuteError
		}
	}
	for _, tx := range tr.normalTransactions.Transactions {
		if tx.Data.ExecuteError != nil {
			return nil, tx.Data.ExecuteError
		}
	}
	return nil, nil
}

func (tr *testTransition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	tr.Lock()
	defer tr.Unlock()

	if tr.step >= transitionStepExecuting {
		return nil, errors.Errorf("already executed")
	}
	verr, eerr := tr.getErrors()
	go func() {
		tr.Lock()
		defer tr.Unlock()

		if verr != nil {
			tr.step = transitionStepFailed
			return
		}

		if tr._exeChan != nil {
			tr.Unlock()
			<-tr._exeChan
			cb.OnValidate(tr, verr)
			tr.Lock()
		} else {
			tr.Unlock()
			cb.OnValidate(tr, verr)
			tr.Lock()
		}

		if tr.step == transitionStepFailed {
			// canceled
			return
		}
		if eerr != nil {
			tr.step = transitionStepFailed
			return
		}
		tr.step = transitionStepSucceed
		if tr._exeChan != nil {
			tr.Unlock()
			<-tr._exeChan
			cb.OnExecute(tr, eerr)
			tr.Lock()
		} else {
			tr.Unlock()
			cb.OnExecute(tr, eerr)
			tr.Lock()
		}
	}()
	return func() bool {
		tr.Lock()
		defer tr.Unlock()

		if tr.step == transitionStepExecuting {
			tr.step = transitionStepFailed
			return true
		}
		return false
	}, nil
}

func (tr *testTransition) ExecuteForTrace(ti module.TraceInfo) (canceler func() bool, err error) {
	return nil, errors.ErrUnsupported
}

func (tr *testTransition) Result() []byte {
	tr.Lock()
	defer tr.Unlock()

	if tr.step == transitionStepSucceed {
		if tr._result == nil {
			result := &testTransitionResult{}
			result.WorldState = tr.EffectiveTransactions().effect().WorldState
			tr._result = codec.MustMarshalToBytes(result)
		}
		return tr._result
	}
	return nil
}

func (tr *testTransition) NextValidators() module.ValidatorList {
	tr.Lock()
	defer tr.Unlock()

	if tr.step == transitionStepSucceed {
		nv := tr.EffectiveTransactions().effect().NextValidators
		if nv != nil {
			return nv
		}
		return tr.baseValidators
	}
	return nil
}

func (tr *testTransition) LogsBloom() module.LogsBloom {
	tr.Lock()
	defer tr.Unlock()

	if tr.step == transitionStepSucceed {
		if tr._logsBloom == nil {
			tr._logsBloom = txresult.NewLogsBloom(nil)
			tr._logsBloom.Merge(&tr.patchTransactions.effect().LogsBloom)
			tr._logsBloom.Merge(&tr.normalTransactions.effect().LogsBloom)
		}
		return tr._logsBloom
	}
	return nil
}

func (tr *testTransition) BlockInfo() module.BlockInfo {
	return tr._bi
}

func (tr *testTransition) Equal(t2 module.Transition) bool {
	tr2 := t2.(*testTransition)
	if tr == tr2 {
		return true
	}
	return tr.patchTransactions.Equal(tr2.patchTransactions) &&
		tr.normalTransactions.Equal(tr2.normalTransactions) &&
		common.BlockInfoEqual(tr._bi, tr2._bi) &&
		common.ConsensusInfoEqual(tr._csi, tr2._csi)
}

func (tr *testTransition) BTPSection() module.BTPSection {
	return btp.ZeroBTPSection
}

type testServiceManager struct {
	module.ServiceManager
	transactions [][]*testTransaction
	bucket       *db.CodedBucket
	exeChan      chan struct{}
}

func newTestServiceManager(database db.Database) *testServiceManager {
	sm := &testServiceManager{}
	sm.transactions = make([][]*testTransaction, 2)
	sm.bucket, _ = db.NewCodedBucket(database, db.BytesByHash, nil)
	return sm
}

func (sm *testServiceManager) setTransitionExeChan(ch chan struct{}) {
	sm.exeChan = ch
}

func (sm *testServiceManager) GetNetworkID(result []byte) (int64, error) {
	return 1, nil
}

func (sm *testServiceManager) GetChainID(result []byte) (int64, error) {
	return 0, errors.ErrNotFound
}

func (sm *testServiceManager) ProposeTransition(
	parent module.Transition,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
) (module.Transition, error) {
	tr := &testTransition{}
	tr.baseValidators = parent.NextValidators().(*testValidatorList)
	tr.patchTransactions = newTestTransactionList(sm.transactions[module.TransactionGroupPatch])
	tr.normalTransactions = newTestTransactionList(sm.transactions[module.TransactionGroupNormal])
	tr._bi = bi
	tr._csi = csi
	if sm.exeChan != nil {
		tr.setExeChan(sm.exeChan)
	}
	return tr, nil
}

func (sm *testServiceManager) CreateInitialTransition(result []byte, nextValidators module.ValidatorList) (module.Transition, error) {
	if nextValidators == nil {
		nextValidators = newTestValidatorList(nil)
	}
	nvl, ok := nextValidators.(*testValidatorList)
	if !ok {
		return nil, errors.Errorf("bad validator list type")
	}
	tr := &testTransition{}
	tr.patchTransactions = newTestTransactionList(nil)
	tr.normalTransactions = newTestTransactionList(nil)
	tr.normalTransactions._effect.WorldState = result
	tr.normalTransactions._effect.NextValidators = nvl
	tr._bi = common.NewBlockInfo(-1, 0)
	tr.step = transitionStepSucceed
	return tr, nil
}

func (sm *testServiceManager) CreateTransition(
	parent module.Transition,
	txs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	validate bool,
) (module.Transition, error) {
	if ttxl, ok := txs.(*testTransactionList); ok {
		for _, ttx := range ttxl.Transactions {
			if ttx.Data.TransitionCreateError != nil {
				return nil, errors.Errorf("bad transaction list")
			}
		}
		tr := &testTransition{}
		tr.baseValidators = parent.NextValidators().(*testValidatorList)
		tr.patchTransactions = newTestTransactionList(nil)
		tr.normalTransactions = ttxl
		tr._bi = bi
		tr._csi = csi
		if sm.exeChan != nil {
			tr.setExeChan(sm.exeChan)
		}
		return tr, nil
	}
	return nil, errors.Errorf("bad type")
}

func (sm *testServiceManager) GetPatches(transition module.Transition, bi module.BlockInfo) module.TransactionList {
	return newTestTransactionList(sm.transactions[module.TransactionGroupPatch])
}

func (sm *testServiceManager) PatchTransition(transition module.Transition,
	patches module.TransactionList,
	bi module.BlockInfo,
) module.Transition {
	var ttxl *testTransactionList
	var ttr *testTransition
	var ok bool
	if ttxl, ok = patches.(*testTransactionList); !ok {
		return nil
	}
	if ttr, ok = transition.(*testTransition); !ok {
		return nil
	}
	tr := &testTransition{}
	tr.baseValidators = ttr.baseValidators
	tr.patchTransactions = ttxl
	tr.normalTransactions = ttr.normalTransactions
	tr._bi = transition.(*testTransition)._bi
	tr._csi = transition.(*testTransition)._csi
	return tr
}

func (sm *testServiceManager) Finalize(transition module.Transition, opt int) error {
	var tr *testTransition
	var ok bool
	if tr, ok = transition.(*testTransition); !ok {
		return errors.New("invalid assertion. not testTransition")
	}
	if opt&module.FinalizeNormalTransaction != 0 {
		log.Must(sm.bucket.Put(tr.normalTransactions))
	}
	if opt&module.FinalizePatchTransaction != 0 {
		log.Must(sm.bucket.Put(tr.patchTransactions))
	}
	if opt&module.FinalizeResult != 0 {
		log.Must(sm.bucket.Put(tr.NextValidators()))
	}
	return nil
}

func (sm *testServiceManager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	ttx := &testTransaction{}
	_, err := codec.UnmarshalFromBytes(b, ttx)
	if err != nil {
		return nil, err
	}
	if ttx.Data.CreateError != nil {
		return nil, ttx.Data.CreateError
	}
	return ttx, nil
}

func (sm *testServiceManager) GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	return sm.TransactionFromBytes(b, blockVersion)
}

func (sm *testServiceManager) TransactionListFromHash(hash []byte) module.TransactionList {
	ttxs := &testTransactionList{}
	err := sm.bucket.Get(db.Raw(hash), ttxs)
	if err != nil {
		return nil
	}
	return ttxs
}

func (sm *testServiceManager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	ttxs := make([]*testTransaction, len(txs))
	for i, tx := range txs {
		var ok bool
		ttxs[i], ok = tx.(*testTransaction)
		if !ok {
			return nil
		}
	}
	return newTestTransactionList(ttxs)
}

func (sm *testServiceManager) SendTransaction(result []byte, height int64, tx interface{}) ([]byte, error) {
	if ttx, ok := tx.(*testTransaction); ok {
		if ttx.Data.CreateError != nil {
			return nil, ttx.Data.CreateError
		}
		sm.transactions[ttx.Group()] = append(sm.transactions[ttx.Group()], ttx)
		return ttx.ID(), nil
	}
	return nil, errors.Errorf("bad type")
}

func (sm *testServiceManager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	tvl := &testValidatorList{}
	err := sm.bucket.Get(db.Raw(hash), tvl)
	if err != nil {
		return nil
	}
	return tvl
}

func (sm *testServiceManager) GetNextBlockVersion(result []byte) int {
	return module.BlockVersion2
}

func (sm *testServiceManager) NextProofContextMapFromResult(result []byte) (module.BTPProofContextMap, error) {
	return btp.ZeroProofContextMap, nil
}

func (sm *testServiceManager) BTPSectionFromResult(result []byte) (module.BTPSection, error) {
	return btp.ZeroBTPSection, nil
}

type testValidator struct {
	Address_ *common.Address
}

func newTestValidator(addr module.Address) *testValidator {
	v := &testValidator{}
	v.Address_ = common.AddressToPtr(addr)
	return v
}

func (v *testValidator) Address() module.Address {
	return v.Address_
}

func (v *testValidator) PublicKey() []byte {
	return nil
}

func (v *testValidator) Bytes() []byte {
	return v.Address_.Bytes()
}

type testValidatorList struct {
	module.ValidatorList
	Validators []*testValidator
}

func newTestValidatorList(validators []*testValidator) *testValidatorList {
	vl := &testValidatorList{}
	vl.Validators = make([]*testValidator, len(validators))
	copy(vl.Validators, validators)
	return vl
}

func (vl *testValidatorList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

func (vl *testValidatorList) Bytes() []byte {
	return codec.MustMarshalToBytes(vl)
}

func (vl *testValidatorList) IndexOf(addr module.Address) int {
	for i, v := range vl.Validators {
		if v.Address().Equal(addr) {
			return i
		}
	}
	return -1
}

func (vl *testValidatorList) Len() int {
	return len(vl.Validators)
}

func (vl *testValidatorList) Get(i int) (module.Validator, bool) {
	if i >= 0 && i < len(vl.Validators) {
		return vl.Validators[i], true
	}
	return nil, false
}

type testCommitVoteSet struct {
	zero       bool
	Pass       bool
	Timestamp_ int64
}

func newCommitVoteSetFromBytes(bs []byte) module.CommitVoteSet {
	vs := &testCommitVoteSet{}
	if bs == nil {
		vs.zero = true
		return vs
	}
	_, err := codec.UnmarshalFromBytes(bs, vs)
	if err != nil {
		return nil
	}
	return vs
}

func newCommitVoteSet(pass bool) module.CommitVoteSet {
	return &testCommitVoteSet{Pass: pass}
}

func newCommitVoteSetWithTimestamp(pass bool, ts int64) module.CommitVoteSet {
	return &testCommitVoteSet{Pass: pass, Timestamp_: ts}
}

func (vs *testCommitVoteSet) VerifyBlock(block module.BlockData, validators module.ValidatorList) ([]bool, error) {
	if block.Height() == 0 && vs.zero {
		return nil, nil
	}
	if vs.Pass {
		return nil, nil
	}
	return nil, errors.Errorf("verify error")
}

func (vs *testCommitVoteSet) Bytes() []byte {
	bs, _ := codec.MarshalToBytes(vs)
	return bs
}

func (vs *testCommitVoteSet) Hash() []byte {
	return crypto.SHA3Sum256(vs.Bytes())
}

func (vs *testCommitVoteSet) Timestamp() int64 {
	return vs.Timestamp_
}

func (vs *testCommitVoteSet) VoteRound() int32 {
	return 0
}

func (vs *testCommitVoteSet) BlockVoteSetBytes() []byte {
	return vs.Bytes()
}

func (vs *testCommitVoteSet) NTSDProofCount() int {
	return 0
}

func (vs *testCommitVoteSet) NTSDProofAt(i int) []byte {
	return nil
}

func newRandomTestValidatorList(n int) *testValidatorList {
	wallets := newWallets(n)
	validators := make([]*testValidator, n)
	for i, w := range wallets {
		validators[i] = newTestValidator(w.Address())
	}
	return newTestValidatorList(validators)
}

func newGenesisTX(n int) *testTransaction {
	tx := newTestTransaction()
	wallets := newWallets(n)
	validators := make([]*testValidator, n)
	for i, w := range wallets {
		validators[i] = newTestValidator(w.Address())
	}
	tx.Data.Effect.NextValidators = newTestValidatorList(validators)
	return tx
}

type testGenesisStorage struct {
	gtx *testTransaction
}

func (t testGenesisStorage) CID() (int, error) {
	return 1, nil
}

func (t testGenesisStorage) NID() (int, error) {
	return 1, nil
}

func (t testGenesisStorage) Height() int64 {
	return 0
}

func (t testGenesisStorage) Type() (module.GenesisType, error) {
	return module.GenesisNormal, nil
}

func (t testGenesisStorage) Genesis() []byte {
	return t.gtx.Bytes()
}

func (t testGenesisStorage) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func newTestGenesisStorage(gtx *testTransaction) *testGenesisStorage {
	return &testGenesisStorage{gtx: gtx}
}

func newTestChain(database db.Database, gtx *testTransaction) *testChain {
	if database == nil {
		database = newMapDB()
	}
	if gtx == nil {
		gtx = newGenesisTX(defaultValidators)
	}
	c := &testChain{
		wallet:   wallet.New(),
		database: database,
		gtx:      gtx,
		gs:       newTestGenesisStorage(gtx),
		vld:      newCommitVoteSetFromBytes,
	}
	c.sm = newServiceManager(c)
	return c
}

func newServiceManager(chain module.Chain) *testServiceManager {
	return newTestServiceManager(chain.Database())
}

func newWallets(n int) []module.Wallet {
	wallets := make([]module.Wallet, n)
	for i := range wallets {
		wallets[i] = wallet.New()
	}
	return wallets
}

func newMapDB() db.Database {
	database, err := db.Open("", "mapdb", "")
	if err != nil {
		log.Panicf("Fail to open database err=%+v", err)
	}
	return database
}
