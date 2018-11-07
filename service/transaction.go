package service

import (
	"bytes"
	"io"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

func newTransactionFromBytes(b []byte) (module.Transaction, error) {
	// TODO It assumes JSON string for transaction. When new transaction
	// serialized format is defined, this should be changed by determining
	// version of serialized format.
	return newTransaction(io.Reader(bytes.NewReader(b)))
}

// TODO module.Transaction의 return type 검토 필요
type transactionV2V3 struct {
}

type transaction struct {
	// added by KN.KIM for transactionPool test
	timestamp int64
	data      []byte
}

// TODO define
type TransactionV4 struct {
	isPatch bool // patch: true, normal: false
}

// TODO 효과적으로 JSON-RPC가 serialization을 할 수 있는 방법을 고민해 보자.
// 당장은 아래와 같이 struct를 외부에 바로 노출할 수 있는 workaround를 제공해 줄 것으로
// 고려해 본다. 그렇게 하기 위해서 public하게 대문자로 변수를 선언했다.
// TODO data type을 어떻게 해야 하는지 정리 필요
// TODO change it to TransactionV3. Just avoid collision.
type Transaction3 struct {
	*transaction

	isPatch bool // patch: true, normal: false

	Version   common.HexInt16  `json:"version"`
	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     common.HexInt    `json:"value"`
	StepLimit common.HexInt    `json:"stepLimit"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	NID       common.HexInt16  `json:"nid"`
	Nonce     common.HexInt64  `json:"nonce"`
	Hash      common.HexBytes  `json:"txHash"`
	Signature common.Signature `json:"signature"`
	Data      TransactionData  `json:"data"`
}

type TransactionData struct {
	Method string `json:"method"`
	// TODO 이건 어떻게 할 건가?
	Params map[string]interface{} `json:"params"`
}

// TODO change it to TransactionV2. Just avoid collision.
type Transaction2 struct {
	isPatch bool // patch: true, normal: false

	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     common.HexInt    `json:"value"`
	Fee       common.HexInt    `json:"fee"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	Nonce     common.HexInt64  `json:"nonce"`
	Hash      common.HexBytes  `json:"tx_hash"`
	Signature common.Signature `json:"signature"`
	Params    []byte
}

func newTransaction(r io.Reader) (*transaction, error) {
	if r == nil {
		return nil, common.ErrIllegalArgument
	}
	// TODO impl
	return nil, nil
}

func (tx *transaction) ID() []byte {
	// added by KN.KIM for transactionPool test
	return tx.data
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

// TODO
func (tx *transaction) From() module.Address {
	return nil
}

// TODO
func (tx *transaction) To() module.Address {
	return nil
}

// TODO
func (tx *transaction) Value() int {
	return -1
}

// TODO
func (tx *transaction) StepLimit() int {
	return -1
}

// TODO
func (tx *transaction) TimeStamp() int64 {
	// added by KN.KIM for transactionPool test
	return tx.timestamp
}

// TODO
func (tx *transaction) NID() int {
	return -1
}

// TODO
func (tx *transaction) Nonce() int64 {
	return -1
}

// TODO
func (tx *transaction) Hash() []byte {
	return nil
}

// TODO
func (tx *transaction) Signature() []byte {
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

// tx pool에 들어가기 전에 체크
// TODO 뭘 해야 하는지 확인 필요
// TODO 이건 안 하는 게 좋지 않을까 생각. 일단 GC 방법이 결정되면 검토 필요
func (tx *scoreCallTx) check() error {
	return nil
}

// TODO 뭘 해야 하는지 확인 필요
func (tx *scoreCallTx) validate(state trie.Mutable) error {
	return nil
}

func (tx *scoreCallTx) execute(state *transitionState) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	return nil
}

func (tx *scoreCallTx) cancel() {
}

type scoreDeployTx struct {
	transaction
}
