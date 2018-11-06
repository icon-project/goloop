package service

import (
	"io"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

// TODO State manager와 Service manager를 분리할 것인지 합칠 것인지 고민
// 즉 외부에서 특정 정보를 얻으려고 할 때 그냥 State manager를 통해서 바로 얻어가는 게
// 맞는가?

////////////////////
// Transaction Pool
////////////////////
// TODO garbage를 정리하는 방법 필요. 간단하게는 removeList()에 넣어두면 되긴 한데...
// add()할 때 개수 체크 및 candidate()에서 정리.
// TODO tx를 버리는 기준 확인 필요
// TODO tx 시간 순으로 정렬 필요
// TODO 당연하지만, lock 잘 잡고...
type transactionPool struct {
}

// transaction에 넣을 때 간단한 검증이 필요하다면, 검증은 외부에서 해야 함.
func (pool *transactionPool) add(tx transaction) error {
	return nil
}

// 없다면, len()이 0인 TransactionList를 리턴한다. (nil 아님)
// It returns all candidates for a negative integer n.
func (pool *transactionPool) candidate(state trie.Mutable, max int) []transaction {
	// TODO state를 전달받더라도 실제 account info는 address를 통해서 바로 찾는 것이
	// 유리할텐데... trie를 통해서 Get하면 비효율적임.
	// TODO max가 음수이면 모든 transaction을 리턴한다. patch pool에 대해서 필요할 것
	// 같음.
	// TODO validate 작업도 필요.
	// TODO ServiceManager에 하나의 pool을 관리하고 candidate를 구할 때 transition
	// 기반으로 사용된 적이 있는 것을 제외하는 방식으로 구현하려고 하는데, unfinalized
	// branch가 긴 것을 감안하면 좀 더 효과적인 구현이 있을지 고민 필요
	return nil
}

// 이것을 사용할 경우 없음.
func (pool *transactionPool) remove(tx transaction) {
}

// 사용할 경우 없음. 이것도 간단한 검증은 외부에서 수행
func (pool *transactionPool) addList(tx []transaction) {

}

// finalize할 때 호출됨.
func (pool *transactionPool) removeList(tx []transaction) {
	// TODO 효과적으로 제거하는 방안 필요
}

////////////////////
// Transaction
////////////////////
// TODO normal과 patch를 구분해야 하는가? 또한 naming에서 transaction vs patch로 정리할까?
type transaction struct {
	// TODO 아래는 type integer로 하거나, 혹은 struct를 분리하는 방식도 있음.
	isPatch bool // patch: true, normal: false
}

func newTransaction(r io.Reader) (*transaction, error) {
	if r == nil {
		return nil, common.ErrIllegalArgument
	}
	// TODO impl
	return nil, nil
}

func (tx *transaction) ID() []byte {
	return nil
}
func (tx *transaction) Version() int {
	return 0
}
func (tx *transaction) Bytes() ([]byte, error) {
	return nil, nil
}

// TODO check()인지 validate()인지 확인 필요.
func (tx *transaction) Verify() error {
	return nil
}

// tx pool에 들어가기 전에 체크
// TODO 뭘 해야 하는지 확인 필요
// TODO 이건 안 하는 게 좋지 않을까 생각. 일단 GC 방법이 결정되면 검토 필요
func (tx *transaction) check() error {
	return nil
}

// TODO 뭘 해야 하는지 확인 필요
func (tx *transaction) validate(state trie.Mutable) error {
	return nil
}

func (tx *transaction) execute(state *transitionState) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	return nil
}

func (tx *transaction) cancel() {
}

type transferTx struct {
	transaction
}

type scoreCallTx struct {
	transaction
}

func (t *scoreCallTx) execute(state *transitionState) error {
	// TODO rollback될 것을 생각해서 항상 처음에 snapshot을 찍어줘야 한다.
	return nil
}

type scoreDeployTx struct {
	transaction
}

////////////////////
// Transaction List
////////////////////
// TODO to avoid name conflict, temporarily take 'list' instead of 'List'
type transactionlist struct {
	txs  []transaction
	hash []byte
}

func (l *transactionlist) Get(n int) (module.Transaction, error) {
	if n < 0 || n >= len(l.txs) {
		return nil, common.ErrIllegalArgument
	}
	return &l.txs[n], nil
}

func (l *transactionlist) Iterator() module.TransactionIterator {
	return nil
}

// TODO 구현
func (l *transactionlist) Hash() []byte {
	return nil
}

func (l *transactionlist) Equal(txList module.TransactionList) bool {
	return true
}

type txIterator struct {
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
			// TODO i가 256를 넘을 경우를 감안한 byte encoding 수정
			bytes, _ := r.Bytes()
			l.trie.Set([]byte{byte(i)}, bytes)
		}
		l.hash = l.trie.GetSnapshot().RootHash()
	}
	return l.hash
}
