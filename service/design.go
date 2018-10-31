package service

import (
	"io"
	"sync"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

////////////////////
// Service Manager
////////////////////

// TODO State manager와 Service manager를 분리할 것인지 합칠 것인지 고민
// 즉 외부에서 특정 정보를 얻으려고 할 때 그냥 State manager를 통해서 바로 얻어가는 게
// 맞는가?
// TODO 전체적으로 lock을 잘 잡아야 함.
// TODO Receipt와 Validator 쪽은 Service module 내에서 실제 만드는 쪽에서 struct를 정의하는 게 맞을 듯
type manager struct {
	patchTxPool  *txPool
	normalTxPool *txPool

	db db.DB
}

// TODO 아래 function이 interface로 정의되는 게 맞는데, chain manager를 통해서 제공될 수도 있기 때문에 일단 여기에 두자.
func NewManager(db db.DB) module.ServiceManager {
	// TODO 제대로 초기화해야 함.
	return &manager{patchTxPool: new(txPool), normalTxPool: new(txPool)}
}

// ProposeTransition proposes a Transition following the parent Transition.
// Returned Transition always passes validation.
func (m *manager) ProposeTransition(parent module.Transition) (module.Transition, error) {
	// TODO parent가 없거나 result 등의 정보가 잘못되었으면 오류를 발생한다.
	// TODO 적절한 transaction을 normalTxPool에서 얻어와서 validation을 수행한 후
	// 추가한다.
	// TODO 이 때 transaction 개수는 configurable하게 하자. 일단 위에 const로 빼 놓고
	// 이후 configuration이 추가되면 반영하는 방법으로 진행하자.
	return nil, nil
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte, vs []module.Validator) (module.Transition, error) {
	// TODO result가 invalid하거나 validator가 이상하면 오류 발생
	return &transition{result: result, nextValidators: vs}, nil
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
func (m *manager) CreateTransition(parent module.Transition, txs module.TransactionList) (module.Transition, error) {
	// TODO 간단한 parent check를 하여 문제가 있다면 error
	return nil, nil
}

// GetPatches returns all patch transactions based on the parent transition.
// If it doesn't have any patches, it returns nil.
func (m *manager) GetPatches(parent module.Transition) module.TransactionList {
	// TODO patchTxPool에서 patch Tx를 찾아서 리턴한다.
	return nil
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(transition module.Transition, patches module.TransactionList) module.Transition {
	// TODO 기존 transition에서 patch를 덮어 씌운다. invalid한 patch임을 판단할 수 있는
	// 방법이 있는가? 있다면 transition 기준에서 부적절한 patch임을 판단하여 오류 발생.
	// TODO 기존 transition을 그대로 리턴하면 안 되고 새로 생성해서 리턴해야 한다.
	return nil
}

// Finalize finalizes data related to the transition. It usually stores
// data to a persistent storage. opt indicates which data are finalized.
// It should be called for every transition.
func (m *manager) Finalize(transition module.Transition, opt int) {
	// TODO transition instance로 변경하여 처리
}

// TransactionFromReader returns a Transaction instance from bytes
// read by Reader.
func (m *manager) TransactionFromReader(r io.Reader) module.Transaction {
	// TODO Reader를 통해서 읽어 들여서 transaction instance를 만든다.
	return nil
}

////////////////////
// Transition
////////////////////
const (
	// TODO stepValidating과 stepExecuting은 필요없는데 만들까 말까?
	stepInit = iota
	stepValidating
	stepValidated
	stepExecuting
	stepExecuted
)

type transition struct {
	parent *transition

	normalTransactions *txList
	patchTransactions  *txList
	// TODO 아래 result의 구조체를 별도 정의
	result         []byte
	nextValidators []module.Validator
	normalReceipts module.ReceiptList
	patchReceipts  module.ReceiptList
	logBloom       []byte

	cb module.TransitionCallback

	// Execute() 호출 직전 이전 상태의 state trie 상태로 시작해서 tx handling하면서
	// 그 상태가 변한다.
	// Execute()가 호출될 때 parent transition에서 trie를 복사해 오면 된다.
	// state를 변경하는 것들은 모두 service package이기 때문에 부적절한 사용을 하지
	// 않는다고 가정하고 copy가 아닌 pointer를 직접 전달하자.
	state *trie.Mutable

	// internal processing state
	step  int
	mutex sync.Mutex
}

func (t *transition) PatchTransactions() module.TransactionList {
	return t.patchTransactions
}
func (t *transition) NormalTransactions() module.TransactionList {
	return t.normalTransactions
}

// Execute executes this transition.
// The result is asynchronously notified by cb. canceler can be used
// to cancel it after calling Execute. After canceler returns true,
// all succeeding cb functions may not be called back.
// REMARK: It is assumed to be called once. Any additional call returns
// error.
func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	// TODO lock을 잡아라.
	if step > stepInit {
		return nil, common.ErrInvalidState
	}
	// TODO thread를 만들고 executeSync()를 호출해라.
	return nil, nil
}

// Result returns service manager defined result bytes.
func (t *transition) Result() []byte { return t.result }

// NextValidators returns the addresses of validators as a result of
// transaction processing.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) NextValidators() []module.Validator { return t.nextValidators }

// PatchReceipts returns patch receipts.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) PatchReceipts() module.ReceiptList { return t.patchReceipts }

// NormalReceipts returns receipts.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) NormalReceipts() module.ReceiptList { return t.normalReceipts }

// LogBloom returns log bloom filter for this transition.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) LogBloom() []byte { return t.logBloom }

func (t *transition) validate() error {
}

func (t *transition) executeSync() {
}

func (t *transition) finalize(opt int) {
	// TODO 결과를 finalize하게 되면 parent transition instance를 끊어 줘야 한다.
	// 메모리에서 해제될 수 있도록...
	// TODO opt에 따라서 관련 state를 계산하고 persistent storage에 저장한다.
}

////////////////////
// Transaction Pool
////////////////////
// TODO garbage를 정리하는 방법 필요. 간단하게는 removeList()에 넣어두면 되긴 한데...
// add()할 때 개수 체크 및 candidate()에서 정리
// TODO GC 방법은 정리 필요
type txPool struct {
}

// transaction에 넣을 때 간단한 검증이 필요하다면, 검증은 외부에서 해야 함.
func (pool *txPool) add(tx tx) error {
	return nil
}

// 없다면, len()이 0인 TransactionList를 리턴한다. (nil 아님)
func (pool *txPool) candidate(n int) []tx {
	// TODO validate 작업도 필요.
	// TODO ServiceManager에 하나의 pool을 관리하고 candidate를 구할 때 transition
	// 기반으로 사용된 적이 있는 것을 제외하는 방식으로 구현하려고 하는데, unfinalized
	// branch가 긴 것을 감안하면 좀 더 효과적인 구현이 있을지 고민 필요
	return nil
}

// 이것을 사용할 경우 없음.
func (pool *txPool) remove(tx tx) {
}

// 사용할 경우 없음. 이것도 간단한 검증은 외부에서 수행
func (pool *txPool) addList(tx []tx) {

}

// finalize할 때 호출됨.
func (pool *txPool) removeList(tx []tx) {
}

////////////////////
// Transaction
////////////////////
// TODO normal과 patch를 구분해야 하는가? 또한 naming에서 tx vs patch로 정리할까?
type tx struct {
	// TODO 아래는 type integer로 하거나, 혹은 struct를 분리하는 방식도 있음.
	isPatch bool // patch: true, normal: false
}

func (tx *tx) ID() []byte {
	return nil
}
func (tx *tx) Version() int {
	return 0
}
func (tx *tx) Bytes() ([]byte, error) {
	return nil, nil
}

// TODO check()인지 validate()인지 확인 필요.
func (tx *tx) Verify() error {
	return nil
}

// tx pool에 들어가기 전에 체크
// TODO 뭘 해야 하는지 확인 필요
// TODO 이건 안 하는 게 좋지 않을까 생각. 일단 GC 방법이 결정되면 검토 필요
func (tx *tx) check() error {
	return nil
}

// TODO 뭘 해야 하는지 확인 필요
func (tx *tx) validate(state *trie.Mutable) error {
	return nil
}

// TODO state를 공통으로 사용하면 external handler가 그 값을 이상하게 바꿀 수 있는데,
// 이런 부분을 대비해야 하나?
func (tx *tx) execute(state *trie.Mutable) error {
	return nil
}

type transferTx struct {
	tx
}

type scoreCallTx struct {
	tx
}

type scoreDeployTx struct {
	tx
}

////////////////////
// Transaction List
////////////////////
type txList struct {
	txs  []tx
	hash []byte
}

func (l *txList) Get(n int) (module.Transaction, error) {
	if n < 0 || n >= len(l.txs) {
		return nil, common.ErrIllegalArgument
	}
	return &l.txs[n], nil
}
func (l *txList) Size() int {
	return len(l.txs)
}

// TODO 구현
func (l *txList) Hash() []byte {
	return nil
}
