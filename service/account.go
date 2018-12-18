package service

import (
	"log"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
)

// AccountSnapshot represents immutable account state
// It can be get from AccountState or WorldSnapshot.
type AccountSnapshot interface {
	trie.Object
	GetBalance() *big.Int
	IsContract() bool
	Empty() bool
	GetValue(k []byte) ([]byte, error)

	GetContractOwner() *common.Address
	GetCurContract() ContractSnapshot
	GetNextContract() ContractSnapshot
}

// AccountState represents mutable account state.
// You may change account state with this object. It can be get from
// WorldState. Changes in this object will be retrieved by WorldState.
// Of course, it also can be changed by WorldState.
type AccountState interface {
	GetBalance() *big.Int
	IsContract() bool
	GetValue(k []byte) ([]byte, error)
	SetBalance(v *big.Int)
	SetValue(k, v []byte) error
	DeleteValue(k []byte) error
	GetSnapshot() AccountSnapshot
	Reset(snapshot AccountSnapshot) error

	GetContractOwner() *common.Address
	SetContractOwner(addr *common.Address)
	GetCurContract() Contract
	SetCurContract(contract Contract)
	GetNextContract() Contract
	SetNextContract(contract Contract)
}

type ContractSnapshot interface {
	GetStatus() contractStatus
	GetDeployTx() []byte
	GetAuditTx() []byte
	GetCodeHash() []byte
	GetApiInfo() []byte
	GetParams() []byte
}

type Contract interface {
	ContractSnapshot
	Reset(snapshot ContractSnapshot) error
	SetStatus(status contractStatus)
	SetDeployTx(tx []byte)
	SetAuditTx(tx []byte)
	SetCodeHash(hash []byte)
	SetApiInfo(info []byte)
	SetParams(params []byte)
	GetSnapshot() ContractSnapshot
}

type accountSnapshotImpl struct {
	balance     common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database

	contractOwner *common.Address
	curContract   ContractSnapshot
	nextContract  ContractSnapshot
}

func (s *accountSnapshotImpl) GetBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountSnapshotImpl) IsContract() bool {
	return s.fIsContract
}

func (s *accountSnapshotImpl) GetValue(k []byte) ([]byte, error) {
	return s.store.Get(k)
}

func (s *accountSnapshotImpl) Empty() bool {
	return s.balance.BitLen() == 0 && s.store == nil
}

func (s *accountSnapshotImpl) Bytes() []byte {
	b, err := codec.MP.MarshalToBytes(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *accountSnapshotImpl) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.MP.UnmarshalFromBytes(data, s)
	return err
}

func (s *accountSnapshotImpl) Flush() error {
	if sp, ok := s.store.(trie.Snapshot); ok {
		return sp.Flush()
	}
	return nil
}

func (s *accountSnapshotImpl) Equal(object trie.Object) bool {
	if s2, ok := object.(*accountSnapshotImpl); ok {
		if s == s2 {
			return true
		}
		if s == nil || s2 == nil {
			return false
		}
		if s.fIsContract != s2.fIsContract ||
			s.balance.Cmp(&s2.balance.Int) != 0 {
			return false
		}
		if s.store == s2.store {
			return true
		}
		if s.store == nil || s2.store == nil {
			return false
		}
		return s.store.Equal(s2.store, false)
	} else {
		log.Panicf("Replacing accountSnapshotImpl with other object(%T)", object)
	}
	return false
}

func (s *accountSnapshotImpl) GetContractOwner() *common.Address {
	if s.contractOwner == nil {
		return nil
	}
	newAddr := new(common.Address)
	_ = newAddr.SetBytes(s.contractOwner.Bytes())
	return newAddr
}

func (s *accountSnapshotImpl) GetCurContract() ContractSnapshot {
	return s.curContract
}

func (s *accountSnapshotImpl) GetNextContract() ContractSnapshot {
	return s.nextContract
}

func (s *accountSnapshotImpl) CodecEncodeSelf(e *ugorji.Encoder) {
	_ = e.Encode(s.balance)
	_ = e.Encode(s.fIsContract)
	if s.store != nil {
		_ = e.Encode(s.store.Hash())
	} else {
		_ = e.Encode(nil)
	}
	_ = e.Encode(s.contractOwner)
	_ = e.Encode(s.curContract)
	_ = e.Encode(s.nextContract)
}

func (s *accountSnapshotImpl) CodecDecodeSelf(d *ugorji.Decoder) {
	if err := d.Decode(&s.balance); err != nil {
		log.Fatalf("Fail to decode balance in account")
	}
	if err := d.Decode(&s.fIsContract); err != nil {
		log.Fatalf("Fail to decode isContract in account")
	}
	var hash []byte
	if err := d.Decode(&hash); err != nil {
		log.Fatalf("Fail to decode hash in account")
	} else {
		if len(hash) == 0 {
			s.store = nil
		} else {
			s.store = trie_manager.NewImmutable(s.database, hash)
		}
	}
	if err := d.Decode(&s.contractOwner); err != nil {
		log.Fatalf("Fail to decode contractOwner in account")
	}
	var cc contractSnapshotImpl
	if err := d.Decode(&cc); err != nil {
		log.Fatalf("Fail to decode curContract in account")
	}
	s.curContract = &cc

	var nc contractSnapshotImpl
	if err := d.Decode(&nc); err != nil {
		log.Fatalf("Fail to decode nextContract in account")
	}
	s.nextContract = &nc
}

type contractStatus int

const (
	csInactive contractStatus = iota
	csActive
	csRejected
	csPending
)

type contractSnapshotImpl struct {
	status      contractStatus
	contentType string
	deployTx    []byte
	auditTx     []byte
	codeHash    []byte
	apiInfo     []byte // API Function Info
	params      []byte
}

type contractImpl struct {
	*contractSnapshotImpl
}

func newContractImpl() *contractImpl {
	return &contractImpl{new(contractSnapshotImpl)}
}

func (c *contractSnapshotImpl) GetStatus() contractStatus {
	return c.status
}

func (c *contractSnapshotImpl) GetDeployTx() []byte {
	return append([]byte(nil), c.deployTx...)
}

func (c *contractSnapshotImpl) GetAuditTx() []byte {
	return append([]byte(nil), c.auditTx...)
}

func (c *contractSnapshotImpl) GetCodeHash() []byte {
	return append([]byte(nil), c.codeHash...)
}

func (c *contractSnapshotImpl) GetApiInfo() []byte {
	return append([]byte(nil), c.apiInfo...)
}

func (c *contractSnapshotImpl) GetParams() []byte {
	return append([]byte(nil), c.params...)
}

func (c *contractImpl) Reset(isnapshot ContractSnapshot) error {
	snapshot, ok := isnapshot.(*contractSnapshotImpl)
	if ok == false {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", c)
	}
	c.status = snapshot.status
	c.contentType = snapshot.contentType
	c.params = append([]byte(nil), snapshot.params...)
	c.apiInfo = append([]byte(nil), snapshot.apiInfo...)
	c.auditTx = append([]byte(nil), snapshot.auditTx...)
	c.deployTx = append([]byte(nil), snapshot.deployTx...)
	c.codeHash = append([]byte(nil), snapshot.codeHash...)
	return nil
}

func (c *contractImpl) SetStatus(status contractStatus) {
	c.status = status
}

func (c *contractImpl) SetDeployTx(tx []byte) {
	c.deployTx = append([]byte(nil), tx...)
}

func (c *contractImpl) SetAuditTx(tx []byte) {
	c.auditTx = append([]byte(nil), tx...)
}

func (c *contractImpl) SetCodeHash(hash []byte) {
	c.codeHash = append([]byte(nil), hash...)
}

func (c *contractImpl) SetApiInfo(info []byte) {
	c.apiInfo = append([]byte(nil), info...)
}

func (c *contractImpl) SetParams(params []byte) {
	c.params = append([]byte(nil), params...)
}

func (c *contractImpl) GetSnapshot() ContractSnapshot {
	return &contractSnapshotImpl{
		status:      c.status,
		contentType: c.contentType,
		deployTx:    append([]byte(nil), c.deployTx...),
		auditTx:     append([]byte(nil), c.auditTx...),
		codeHash:    append([]byte(nil), c.codeHash...),
		apiInfo:     append([]byte(nil), c.apiInfo...),
		params:      append([]byte(nil), c.params...),
	}
}

func (c *contractSnapshotImpl) CodecEncodeSelf(e *ugorji.Encoder) {
	_ = e.Encode(c.status)
	_ = e.Encode(c.contentType)
	_ = e.Encode(c.deployTx)
	_ = e.Encode(c.auditTx)
	_ = e.Encode(c.codeHash)
	_ = e.Encode(c.apiInfo)
	_ = e.Encode(c.params)
}

func (c *contractSnapshotImpl) CodecDecodeSelf(d *ugorji.Decoder) {
	if err := d.Decode(&c.status); err != nil {
		log.Fatalf("Fail to decode status in account")
	}
	if err := d.Decode(&c.contentType); err != nil {
		log.Fatalf("Fail to decode contentType in account")
	}
	if err := d.Decode(&c.deployTx); err != nil {
		log.Fatalf("Fail to decode deployTx in account")
	}
	if err := d.Decode(&c.auditTx); err != nil {
		log.Fatalf("Fail to decode auditTx in account")
	}
	if err := d.Decode(&c.codeHash); err != nil {
		log.Fatalf("Fail to decode codeHash in account")
	}
	if err := d.Decode(&c.apiInfo); err != nil {
		log.Fatalf("Fail to decode apiInfo in account")
	}
	if err := d.Decode(&c.params); err != nil {
		log.Fatalf("Fail to decode params in account")
	}
}

type accountStateImpl struct {
	database   db.Database
	balance    common.HexInt
	isContract bool

	contractOwner *common.Address
	curContract   Contract
	nextContract  Contract
	store         trie.Mutable
}

func (s *accountStateImpl) GetContractOwner() *common.Address {
	return s.contractOwner
}

func (s *accountStateImpl) GetCurContract() Contract {
	return s.curContract
}

func (s *accountStateImpl) GetNextContract() Contract {
	return s.nextContract
}

func (s *accountStateImpl) SetContractOwner(addr *common.Address) {
	s.contractOwner = addr
}

func (s *accountStateImpl) SetCurContract(contract Contract) {
	s.curContract = contract
}

func (s *accountStateImpl) SetNextContract(contract Contract) {
	s.nextContract = contract
}

func (s *accountStateImpl) GetBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountStateImpl) SetBalance(v *big.Int) {
	s.balance.Set(v)
}

func (s *accountStateImpl) IsContract() bool {
	return s.isContract
}

func (s *accountStateImpl) GetSnapshot() AccountSnapshot {
	var store trie.Immutable
	if s.store != nil {
		store = s.store.GetSnapshot()
		if store.Empty() {
			store = nil
		}
	}

	var contractOwner *common.Address
	if s.contractOwner != nil {
		contractOwner = common.NewAccountAddress(s.contractOwner.Bytes())
	}

	var curContract ContractSnapshot
	var nextContract ContractSnapshot
	if s.curContract != nil {
		curContract = s.curContract.GetSnapshot()
	}
	if s.nextContract != nil {
		nextContract = s.nextContract.GetSnapshot()
	}

	return &accountSnapshotImpl{
		balance:       s.balance.Clone(),
		fIsContract:   s.isContract,
		store:         store,
		contractOwner: contractOwner,
		curContract:   curContract,
		nextContract:  nextContract,
	}
}

func (s *accountStateImpl) Reset(isnapshot AccountSnapshot) error {
	snapshot, ok := isnapshot.(*accountSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	s.balance.Set(&snapshot.balance.Int)
	s.isContract = snapshot.fIsContract

	if snapshot.contractOwner != nil {
		s.contractOwner = common.NewAccountAddress(snapshot.contractOwner.Bytes())
	}
	if snapshot.curContract != nil {
		s.curContract = newContractImpl()

		_ = s.curContract.Reset(snapshot.curContract)
	}
	if snapshot.nextContract != nil {
		s.nextContract = newContractImpl()
		_ = s.nextContract.Reset(snapshot.nextContract)
	}
	if s.store == nil && snapshot.store == nil {
		return nil
	}
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	if snapshot.store == nil {
		s.store = nil
	} else {
		if err := s.store.Reset(snapshot.store); err != nil {
			log.Panicf("Fail to make accountStateImpl err=%v", err)
		}
	}

	return nil
}

func (s *accountStateImpl) GetValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountStateImpl) SetValue(k, v []byte) error {
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	return s.store.Set(k, v)
}

func (s *accountStateImpl) DeleteValue(k []byte) error {
	if s.store == nil {
		return nil
	}
	return s.store.Delete(k)
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl) AccountState {
	s := new(accountStateImpl)
	s.database = database
	if snapshot != nil {
		if err := s.Reset(snapshot); err != nil {
			return nil
		}
	}
	return s
}

type contractROState struct {
	*contractSnapshotImpl
}

func (c *contractROState) SetStatus(status contractStatus) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) SetDeployTx(tx []byte) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) SetAuditTx(tx []byte) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) SetCodeHash(hash []byte) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) SetApiInfo(info []byte) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) SetParams(params []byte) {
	log.Printf("ReadOnlyState")
}

func (c *contractROState) GetSnapshot() ContractSnapshot {
	return c.contractSnapshotImpl
}

func (c *contractROState) Reset(snapshot ContractSnapshot) error {
	return errors.New("ReadOnlyState")
}

func newContractROState(isnapshot ContractSnapshot) Contract {
	snapshot, ok := isnapshot.(*contractSnapshotImpl)
	if ok == false {
		return nil
	}
	return &contractROState{snapshot}
}

type accountROState struct {
	AccountSnapshot

	contractOwner *common.Address
	curContract   Contract
	nextContract  Contract
}

func (a *accountROState) SetBalance(v *big.Int) {
	log.Panicf("accountROState().SetBalance() is invoked")
}

func (a *accountROState) SetValue(k, v []byte) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) DeleteValue(k []byte) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) GetSnapshot() AccountSnapshot {
	return a.AccountSnapshot
}

func (a *accountROState) Reset(snapshot AccountSnapshot) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) SetContractOwner(addr *common.Address) {
	log.Panicf("accountROState().SetContractOwner() is invoked")
}

func (a *accountROState) SetCurContract(contract Contract) {
	log.Panicf("accountROState().SetCurContract() is invoked")
}

func (a *accountROState) SetNextContract(contract Contract) {
	log.Panicf("accountROState().SetNextContract() is invoked")
}

func (a *accountROState) GetCurContract() Contract {
	return a.curContract
}

func (a *accountROState) GetNextContract() Contract {
	return a.nextContract
}

func newAccountROState(snapshot AccountSnapshot) AccountState {
	return &accountROState{snapshot,
		snapshot.GetContractOwner(),
		newContractROState(snapshot.GetCurContract()),
		newContractROState(snapshot.GetNextContract())}
}
