package service

import (
	"fmt"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"log"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	mp "github.com/ugorji/go/codec"
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

type accountInfo struct {
	contract bool // true if this account is contract
	balance  *common.HexInt
	// storageRoot 	trie.Mutable
	// contractHash	[]byte
}

func (s *accountInfo) CodecEncodeSelf(e *mp.Encoder) {
	e.Encode(s.balance)
	e.Encode(s.contract)
}

func (s *accountInfo) CodecDecodeSelf(d *mp.Decoder) {
	if err := d.Decode(&s.balance); err != nil {
		log.Fatalf("Fail to decode balance in account")
	}
	if err := d.Decode(&s.contract); err != nil {
		log.Fatalf("Fail to decode isContract in account")
	}
}

func (tx *transaction) execute(state *transitionState) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	// TODO: Change accountInfo to accountState
	stateTrie := state.state
	var account [2]accountInfo // 0 is from, 1 is to
	var addr [2][]byte

	addr[0] = tx.From().Bytes()
	if serializedAccount, err := stateTrie.Get(addr[0]); err == nil {
		if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &account[0]); err != nil {
			fmt.Println("Failed to unmarshal")
			return err
		}
	} else {
		fmt.Println("Failed to get address")
		return err
	}

	txValue := tx.Value()

	if account[0].balance.Cmp(txValue) < 0 {
		//return NotEnoughBalance
		fmt.Println("Not enough balance. ", account[0].balance, ", value ", txValue)
		return nil
	}

	addr[1] = tx.To().Bytes()
	if serializedAccount, err := stateTrie.Get(addr[1]); err == nil {
		if serializedAccount != nil {
			if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &account[1]); err != nil {
				return err
			}
		} else {
			account[1] = accountInfo{balance: &common.HexInt{*big.NewInt(0)}}
		}
	}

	account[0].balance.Sub(&account[0].balance.Int, txValue)
	account[1].balance.Add(&account[1].balance.Int, txValue)

	snapshot := stateTrie.GetSnapshot()
	for i, account := range account {
		if serializedAccount, err := codec.MP.MarshalToBytes(account); err == nil {
			if err = stateTrie.Set(addr[i], serializedAccount); err != nil {
				stateTrie.Reset(snapshot)
				return err
			}
		} else {
			stateTrie.Reset(snapshot)
			return err
		}
	}

	return nil
}

func TestExecute() {
	address := [2]common.Address{
		*common.NewAccountAddress([]byte("HELLO")),
		*common.NewAccountAddress([]byte("HI")),
	}
	account := newAccountState(nil, nil)
	account.setBalance(big.NewInt(900000))
	serializedAccount, _ := codec.MP.MarshalToBytes(account)

	tx := transaction{from: address[0], to: address[1], value: big.NewInt(100000)}
	db := db.NewMapDB()
	mgr := trie_manager.New(db)
	trie := mgr.NewMutable(nil)
	trie.Set(address[0].Bytes(), serializedAccount)
	transitionState := &transitionState{state: trie}
	tx.execute(transitionState)

	var result accountInfo
	for _, v := range address {
		serializedAccount, _ = trie.Get(v.Bytes())
		if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &result); err != nil {
			fmt.Println("ERROR!!!!!")
			return
		}
		fmt.Println("result.balance : ", result.balance)
	}
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
