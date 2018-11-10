package service

import (
	"encoding/hex"
	"log"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/pkg/errors"

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
	verifySignature() error
}

type transaction struct {
	source

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

func (tx *transaction) Group() module.TransactionGroup {
	return tx.group
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

// Verify conducts TX syntax check, signature verification, and balance check.
// It is called when JSON-RPC server pre-validates.
func (tx *transaction) Verify() error {
	// TODO What about checking parameters for each tx types? If right,
	// move it to the transferTx, scoreCallTx, and scoreDeployTx.
	// TODO check balance
	return tx.source.verifySignature()
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

func (tx *transaction) validate(state trie.Mutable, txdb db.Bucket) error {
	// check if it's already handled in a block
	if loc, err := txdb.Get(tx.ID()); loc != nil || err != nil {
		if err != nil {
			return errors.New("TX validation failed due to Transaction Index DB failure")
		}
		errors.New("Already handled TX: " + hex.EncodeToString(tx.ID()))
	}

	// verify a signature
	// TODO transaction execute테스트를 위해 임시로 comment out by KN.KIM
	//tx.source.verifySignature()

	// TODO balance가 충분한지 확인. 그런데 여기에서는 이전 tx의 처리 결과를 감안하여
	// 아직 balance가 충분한지 확인해야 함.
	return nil
}

// TODO: 계정이 없을 경우 계정을 추가해야하는데 newAccount(db) db가 전달되어야 한다. 따라서 db인자를 추가한다. 이후 정리 필요. by KN.KIM
func (tx *transaction) execute(state *transitionState, db db.Database) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	stateTrie := state.state
	var accSnapshot [2]accountSnapshotImpl // 0 is from, 1 is to
	var accState [2]accountState           // 0 is from, 1 is to
	var addr [2][]byte

	addr[0] = tx.From().Bytes()
	if serializedAccount, err := stateTrie.Get(addr[0]); err == nil && len(serializedAccount) != 0 {
		if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &accSnapshot[0]); err != nil {
			log.Println("Failed to unmarshal")
			return err
		}
		accState[0] = newAccountState(db, &accSnapshot[0])
	} else {
		log.Println("Failed to get address")
		return err
	}

	txValue := tx.Value()

	if accSnapshot[0].getBalance().Cmp(txValue) < 0 {
		//return NotEnoughBalance
		log.Println("Not enough balance. ", accSnapshot[0].getBalance(), ", value ", txValue)
		return nil
	}

	addr[1] = tx.To().Bytes()
	if serializedAccount, err := stateTrie.Get(addr[1]); err == nil {
		if serializedAccount != nil {
			if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &accSnapshot[1]); err != nil {
				log.Println("Failed to unmarshal")
				return err
			}
			accState[1] = newAccountState(db, &accSnapshot[1])
		} else {
			accState[1] = newAccountState(db, nil)
		}
	}

	accState[0].setBalance(big.NewInt(0).Sub(accSnapshot[0].getBalance(), txValue))
	accState[1].setBalance(big.NewInt(0).Add(accSnapshot[1].getBalance(), txValue))

	stateTrieSs := stateTrie.GetSnapshot()
	for i, account := range accState {
		if resultSnapshot := account.getSnapshot(); resultSnapshot != nil {
			if serializedAccount, err := codec.MP.MarshalToBytes(resultSnapshot); err == nil {
				if err = stateTrie.Set(addr[i], serializedAccount); err != nil {
					stateTrie.Reset(stateTrieSs)
					return err
				}
			} else {
				stateTrie.Reset(stateTrieSs)
				log.Println("Failed to marshal")
				return err
			}
		}
	}
	return nil
}

type transferTx struct {
	transaction
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
