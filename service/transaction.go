package service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/icon-project/goloop/common/crypto"
	"log"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/pkg/errors"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type (
	source interface {
		bytes() []byte
	}

	transaction struct {
		source

		group module.TransactionGroup

		version   int
		from      *common.Address
		to        *common.Address
		value     *big.Int
		stepLimit *big.Int
		timestamp int64
		nid       int
		nonce     int64
		signature *common.Signature

		bytes []byte
		hash  []byte
	}
)

// TODO consider how to provide a convenient way of JSON string conversion
// for JSON-RPC (But still it's optional)
// TODO refactoring for the variation of serialization format and TX API version
func newTransaction(b []byte) (*transaction, error) {
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

// TODO redesign
// newTransactionFromObject copies or creates transaction instance.
func newTransactionFromObject(tx module.Transaction) (*transaction, error) {
	if t, ok := tx.(*transaction); ok {
		// TODO deep copy or shallow copy?
		return t, nil
	}

	var t transaction
	if tx.Group() != module.TransactionGroupNormal &&
		tx.Group() != module.TransactionGroupPatch {
		return nil, common.ErrIllegalArgument
	}
	t.group = tx.Group()
	t.version = tx.Version()
	if t.version != 3 {
		t.version = 2
	}
	if tx.From() == nil {
		return nil, common.ErrIllegalArgument
	}
	t.from = common.NewAddressFromString(tx.From().String())
	if tx.To() == nil {
		return nil, common.ErrIllegalArgument
	}
	t.to = common.NewAddressFromString(tx.To().String())
	t.value = new(big.Int)
	t.value.Set(tx.Value())
	t.stepLimit = new(big.Int)
	t.stepLimit.Set(tx.StepLimit())
	t.timestamp = tx.Timestamp()
	t.nid = tx.NID()
	t.nonce = tx.Nonce()
	sig, err := crypto.ParseSignature(tx.Signature())
	if err != nil {
		return nil, err
	}
	t.signature = &common.Signature{sig}
	t.hash = make([]byte, 0)
	if tx.Hash() == nil {
		t.hash = nil
	} else {
		t.hash = append(t.hash, tx.Hash()...)
	}
	if tx.Bytes() == nil {
		t.bytes = nil
	} else {
		t.bytes = append(t.bytes, tx.Bytes()...)
	}
	return &t, nil
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

func (tx *transaction) From() module.Address {
	return module.Address(tx.from)
}

func (tx *transaction) To() module.Address {
	return module.Address(tx.to)
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

func (tx *transaction) Bytes() []byte {
	if tx.bytes == nil {
		tx.bytes = tx.source.bytes()
	}
	return tx.bytes
}

func (tx *transaction) Hash() []byte {
	if tx.hash == nil {
		tx.hash = crypto.SHA3Sum256(tx.Bytes())
	}
	return tx.hash
}

func (tx *transaction) Signature() []byte {
	if tx.signature != nil {
		sig, err := tx.signature.SerializeRSV()
		if err == nil {
			return sig
		}
	}
	return nil
}

// Verify conducts TX syntax check, signature verification, and balance check.
// It is called when JSON-RPC server pre-validates.
func (tx *transaction) Verify() error {
	// TODO What about checking parameters for each tx types? If right,
	// move it to the transferTx, scoreCallTx, and scoreDeployTx.
	// TODO check balance
	return tx.verifySignature()
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

	// check if balance is enough
	return tx.checkBalance(state)
}

// TODO checkBalance should subtract from balance
func (tx *transaction) checkBalance(state trie.Mutable) error {
	addr := tx.From().Bytes()
	if serializedAccount, err := state.Get(addr); err == nil {
		if len(serializedAccount) != 0 {
			var account accountSnapshotImpl
			if _, err = codec.MP.UnmarshalFromBytes(serializedAccount, &account); err == nil {
				var expense big.Int
				expense.Add(tx.value, tx.stepLimit)
				if account.GetBalance().Cmp(&expense) >= 0 {
					return nil
				}
			} else {
				return errors.New("Account info unmarshaling error")
			}
		}
		return errors.New("Not enough balance")
	}
	return errors.New("Account info access error")
}

// TODO: 계정이 없을 경우 계정을 추가해야하는데 newAccount(db) db가 전달되어야 한다. 따라서 db인자를 추가한다. 이후 정리 필요. by KN.KIM
func (tx *transaction) execute(state *transitionState, db db.Database) error {
	// TODO 지정된 시간 이내에 결과가 나와야 한다.
	stateTrie := state.state
	var accSnapshot [2]accountSnapshotImpl // 0 is from, 1 is to
	var accState [2]AccountState           // 0 is from, 1 is to
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

	if accSnapshot[0].GetBalance().Cmp(txValue) < 0 {
		//return NotEnoughBalance
		log.Println("Not enough balance. ", accSnapshot[0].GetBalance(), ", value ", txValue)
		return nil
	}

	addr[1] = tx.To().Bytes()
	if serializedAccount, err := stateTrie.Get(addr[1]); err == nil {
		if serializedAccount != nil {
			if _, err := codec.MP.UnmarshalFromBytes(serializedAccount, &accSnapshot[1]); err != nil {
				log.Println("Failed to unmarshal")
				return err
			}
		}
	}
	accState[1] = newAccountState(db, &accSnapshot[1])

	accState[0].SetBalance(big.NewInt(0).Sub(accSnapshot[0].GetBalance(), txValue))
	accState[1].SetBalance(big.NewInt(0).Add(accSnapshot[1].GetBalance(), txValue))

	for i, account := range accState {
		if resultSnapshot := account.GetSnapshot(); resultSnapshot != nil {
			if serializedAccount, err := codec.MP.MarshalToBytes(resultSnapshot); err == nil {
				if err = stateTrie.Set(addr[i], serializedAccount); err != nil {
					return err
				}
			} else {
				log.Println("Failed to marshal")
				return err
			}
		}
	}
	return nil
}

var (
	v2FieldInclusion = map[string]bool(nil)
	v2FieldExclusion = map[string]bool{
		"method":    true,
		"signature": true,
		"tx_hash":   true,
	}
	v3FieldInclusion = map[string]bool(nil)
	v3FieldExclusion = map[string]bool{
		"signature": true,
		"txHash":    true,
	}
)

func (tx *transaction) verifySignature() error {
	raw := tx.Bytes()

	var data map[string]interface{}
	var err error
	if err = json.Unmarshal(raw, &data); err != nil {
		log.Println("JSON Parse FAILS")
		log.Println("JSON", string(raw))
		return err
	}
	var bs []byte
	var txHash []byte
	if tx.version == 2 {
		bs, err = SerializeMap(data, v2FieldInclusion, v2FieldExclusion)
	} else {
		bs, err = SerializeMap(data, v3FieldInclusion, v3FieldExclusion)
	}
	txHash = tx.Hash()
	if err != nil {
		log.Println("Serialize FAILs")
		log.Println("JSON", string(raw))
		return err
	}
	h := crypto.SHA3Sum256(bs)

	bs = append([]byte("icx_sendTransaction."), bs...)
	if bytes.Compare(h, txHash) != 0 {
		log.Println("Hashes are different")
		log.Println("JSON.TxHash", hex.EncodeToString(txHash))
		log.Println("Calc.TxHash", hex.EncodeToString(h))
		log.Println("TxPhrase", string(bs))
		return errors.New("txHash value is different from real")
	}

	if err != nil {

	}
	if pk, err := tx.signature.RecoverPublicKey(h); err != nil {
		log.Println("FAIL Recovering public key")
		log.Println("Signature", tx.signature)
		return err
	} else {
		addr := common.NewAccountAddressFromPublicKey(pk).String()
		if err != nil {
			log.Println("FAIL to recovering address from public key")
			return err
		}
		if addr != tx.from.String() {
			log.Println("FROM is different from signer")
			log.Println("SIGNER", addr)
			log.Println("FROM", tx.from)
			return errors.New("FROM is different from signer")
		}
	}
	log.Println("TX verified")
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
