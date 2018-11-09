package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

// TODO consider how to provide a convenient way of JSON string conversion
// for JSON-RPC (But still it's optional)
// TODO refactoring for the variation of serialization format and TX API version
func newTransaction(b []byte) (module.Transaction, error) {
	if len(b) < 1 {
		return nil, common.ErrIllegalArgument
	}

	// Check serialization format
	// We assumes the legacy JSON format starts with '{'
	// Conceptually, serialization format version must be specified
	// from external modules.
	if b[0] == '{' {
		return newTransactionLegacy(b)
	} else {
		return nil, nil
	}
}

type source interface {
	bytes() []byte
	hash() []byte
	verify() error
}

type transaction struct {
	source

	isPatch bool

	version int
	// TODO type check
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

func (tx *transaction) ID() []byte {
	return tx.hash
}
func (tx *transaction) Version() int {
	return tx.version
}

func (tx *transaction) Bytes() ([]byte, error) {
	if tx.bytes == nil {
		tx.bytes = tx.source.bytes()
	}
	return tx.bytes, nil
}

func (tx *transaction) Verify() error {
	// TODO handler별로 check할 게 있을까? 예를 들어 tx format?
	// TODO JSON RPC에서도 이것을 호출하면 그 때는 balance check를 해야 할 수 있다.
	// 현재는 block에서 호출한다.
	return tx.source.verify()
}

func (tx *transaction) From() module.Address {
	return module.Address(&tx.from)
}

func (tx *transaction) To() module.Address {
	return module.Address(&tx.to)
}

func (tx *transaction) Value() *big.Int {
	return tx.value
}

func (tx *transaction) StepLimit() *big.Int {
	return tx.stepLimit
}

func (tx *transaction) Timestamp() int64 {
	return tx.timestamp
}

func (tx *transaction) NID() int {
	return tx.nid
}

func (tx *transaction) Nonce() int64 {
	return tx.nonce
}

func (tx *transaction) Hash() []byte {
	if tx.hash == nil {
		tx.hash = tx.source.hash()
	}
	return tx.hash
}

func (tx *transaction) Signature() []byte {
	return tx.signature
}

func (tx *transaction) validate(state trie.Mutable) error {
	// TODO TX index DB를 확인하여 이미 block에 들어가 있는 것인지 확인
	// TODO signature check
	// TODO balance가 충분한지 확인. 그런데 여기에서는 이전 tx의 처리 결과를 감안하여
	// 아직 balance가 충분한지 확인해야 함.
	return nil
}

type transferTx struct {
	transaction
}

func (tx *transaction) execute(state *transitionState) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	return nil
}

type scoreCallTx struct {
	transaction
}

// TODO 뭘 해야 하는지 확인 필요
func (tx *scoreCallTx) validate(state trie.Mutable) error {
	return nil
}

func (tx *scoreCallTx) execute(state *transitionState) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다. 만약 지정된 시간을 초과하게 되면
	// 중간에 score engine에게 멈추도록 요청해야 한다.
	return nil
}

type scoreDeployTx struct {
	transaction
}
