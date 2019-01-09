package service

import (
	"bytes"
	"log"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
	"golang.org/x/crypto/sha3"
)

const (
	AccountVersion1 = iota + 1
	AccountVersion  = AccountVersion1
)

// AccountSnapshot represents immutable account state
// It can be get from AccountState or WorldSnapshot.
type AccountSnapshot interface {
	trie.Object
	Version() int
	GetBalance() *big.Int
	IsContract() bool
	Empty() bool
	GetValue(k []byte) ([]byte, error)
	StorageChangedAfter(snapshot AccountSnapshot) bool

	IsContractOwner(owner module.Address) bool
	APIInfo() *scoreapi.Info
	Contract() ContractSnapshot
	ActiveContract() ContractSnapshot
	NextContract() ContractSnapshot
	IsDisabled() bool
	IsBlacklisted() bool
	ContractOwner() module.Address
}

// AccountState represents mutable account state.
// You may change account state with this object. It can be get from
// WorldState. Changes in this object will be retrieved by WorldState.
// Of course, it also can be changed by WorldState.
type AccountState interface {
	Version() int
	GetBalance() *big.Int
	IsContract() bool
	GetValue(k []byte) ([]byte, error)
	SetBalance(v *big.Int)
	SetValue(k, v []byte) error
	DeleteValue(k []byte) error
	GetSnapshot() AccountSnapshot
	Reset(snapshot AccountSnapshot) error

	IsContractOwner(owner module.Address) bool
	InitContractAccount(address module.Address)
	DeployContract(code []byte, eeType string, contentType string,
		params []byte, txHash []byte)
	APIInfo() *scoreapi.Info
	SetAPIInfo(*scoreapi.Info)
	AcceptContract(codeHash []byte, auditTxHash []byte) error
	RejectContract(codeHash []byte, auditTxHash []byte) error
	Contract() Contract
	ActiveContract() Contract
	NextContract() Contract
	IsDisabled() bool
	IsBlacklisted() bool
	Disable(b bool)
	Blacklist(b bool)
	ContractOwner() module.Address
}

type accountSnapshotImpl struct {
	version     int
	balance     common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database

	contractOwner *common.Address
	apiInfo       *scoreapi.Info
	curContract   *contractSnapshotImpl
	nextContract  *contractSnapshotImpl
}

func (s *accountSnapshotImpl) ContractOwner() module.Address {
	return s.contractOwner
}

func (s *accountSnapshotImpl) Version() int {
	return s.version
}

func (s *accountSnapshotImpl) ActiveContract() ContractSnapshot {
	if s.curContract != nil && s.curContract.status == csActive {
		return s.curContract
	}
	return nil
}

func (s *accountSnapshotImpl) IsDisabled() bool {
	if s.curContract.status&csDisable != 0 {
		return true
	}
	return false
}

func (s *accountSnapshotImpl) IsBlacklisted() bool {
	if s.curContract.status&csBlacklist != 0 {
		return true
	}
	return false
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
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountSnapshotImpl) Empty() bool {
	return s.balance.BitLen() == 0 && s.store == nil && s.contractOwner == nil
}

func (s *accountSnapshotImpl) Bytes() []byte {
	b, err := codec.MP.MarshalToBytes(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *accountSnapshotImpl) Contract() ContractSnapshot {
	if s.curContract == nil {
		return nil
	}
	return s.curContract
}

func (s *accountSnapshotImpl) NextContract() ContractSnapshot {
	if s.nextContract == nil {
		return nil
	}
	return s.nextContract
}

func (s *accountSnapshotImpl) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.MP.UnmarshalFromBytes(data, s)
	return err
}

func (s *accountSnapshotImpl) Flush() error {
	if s.curContract != nil {
		if err := s.curContract.flush(); err != nil {
			return err
		}
	}
	if s.nextContract != nil {
		if err := s.nextContract.flush(); err != nil {
			return err
		}
	}
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
		if s.contractOwner.Equal(s2.contractOwner) == false {
			return false
		}
		if s.curContract.Equal(s2.curContract) == false {
			return false
		}
		if s.nextContract.Equal(s2.nextContract) == false {
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

func (s *accountSnapshotImpl) Resolve(bd merkle.Builder) error {
	if s.store != nil {
		if err := s.store.Resolve(bd); err != nil {
			return err
		}
	}
	if s.curContract != nil {
		if err := s.curContract.Resolve(bd); err != nil {
			return err
		}
	}
	if s.nextContract != nil {
		if err := s.nextContract.Resolve(bd); err != nil {
			return err
		}
	}
	return nil
}

func (s *accountSnapshotImpl) StorageChangedAfter(ass AccountSnapshot) bool {
	if s2, ok := ass.(*accountSnapshotImpl); ok {
		if s.store == nil && s2.store == nil {
			return true
		}
		if s.store == nil || s2.store == nil {
			return false
		}
		if s2.store.Equal(s.store, false) {
			return true
		}
	}
	return false
}

func (s *accountSnapshotImpl) IsContractOwner(owner module.Address) bool {
	if s.fIsContract == false {
		return false
	}
	return s.contractOwner.Equal(owner)
}

func (s *accountSnapshotImpl) APIInfo() *scoreapi.Info {
	return s.apiInfo
}

func (s *accountSnapshotImpl) CodecEncodeSelf(e *ugorji.Encoder) {
	e.MustEncode(s.version)
	e.MustEncode(s.balance)
	e.MustEncode(s.fIsContract)
	if s.store != nil {
		e.MustEncode(s.store.Hash())
	} else {
		e.MustEncode(nil)
	}
	e.MustEncode(s.contractOwner)
	e.MustEncode(s.apiInfo)
	e.MustEncode(s.curContract)
	e.MustEncode(s.nextContract)
}

func (s *accountSnapshotImpl) CodecDecodeSelf(d *ugorji.Decoder) {
	d.MustDecode(&s.version)
	d.MustDecode(&s.balance)
	d.MustDecode(&s.fIsContract)
	var hash []byte
	d.MustDecode(&hash)
	if len(hash) == 0 {
		s.store = nil
	} else {
		s.store = trie_manager.NewImmutable(s.database, hash)
	}
	d.MustDecode(&s.contractOwner)
	d.MustDecode(&s.apiInfo)
	d.MustDecode(&s.curContract)
	if s.curContract != nil {
		s.curContract.bk, _ = s.database.GetBucket(db.BytesByHash)
	}
	d.MustDecode(&s.nextContract)
	if s.nextContract != nil {
		s.nextContract.bk, _ = s.database.GetBucket(db.BytesByHash)
	}
}

type accountStateImpl struct {
	version    int
	database   db.Database
	balance    common.HexInt
	isContract bool

	contractOwner module.Address
	apiInfo       *scoreapi.Info
	curContract   *contractImpl
	nextContract  *contractImpl
	store         trie.Mutable
}

func (s *accountStateImpl) ContractOwner() module.Address {
	return s.contractOwner
}

func (s *accountStateImpl) Version() int {
	return s.version
}

func (s *accountStateImpl) ActiveContract() Contract {
	if s.curContract != nil && s.curContract.status == csActive {
		return s.curContract
	}
	return nil
}

func (s *accountStateImpl) IsDisabled() bool {
	if s.curContract != nil && s.curContract.status&csDisable != 0 {
		return true
	}
	return false
}

func (s *accountStateImpl) IsBlacklisted() bool {
	if s.curContract != nil && s.curContract.status&csBlacklist != 0 {
		return true
	}
	return false
}

func (s *accountStateImpl) Disable(b bool) {
	if s.curContract != nil {
		status := s.curContract.status & csBlacklist
		if b == true {
			s.curContract.status = status | csDisable
		} else {
			s.curContract.status = status
		}
	}
}

func (s *accountStateImpl) Blacklist(b bool) {
	if s.curContract != nil {
		status := s.curContract.status & csDisable
		if b == true {
			s.curContract.status = status | csBlacklist
		} else {
			s.curContract.status = status
		}
	}
}

func (s *accountStateImpl) IsContractOwner(owner module.Address) bool {
	if s.isContract == false {
		return false
	}
	return s.contractOwner.Equal(owner)
}

func (s *accountStateImpl) InitContractAccount(address module.Address) {
	if s.isContract == true {
		log.Printf("already Contract account")
		return
	}
	s.isContract = true
	s.contractOwner = address
}

func (s *accountStateImpl) DeployContract(code []byte,
	eeType string, contentType string, params []byte, txHash []byte) {
	if s.isContract == false {
		return
	}
	status := csInactive
	if s.curContract != nil {
		status = csPending
	}
	codeHash := sha3.Sum256(code)
	bk, err := s.database.GetBucket(db.BytesByHash)
	if err != nil {
		log.Printf("Failed to get bucket")
		return
	}
	s.nextContract = &contractImpl{contractSnapshotImpl{
		bk: bk, isNew: true, status: status, contentType: contentType,
		eeType: eeType, deployTxHash: txHash, codeHash: codeHash[:],
		params: params, code: code},
	}
}

func (s *accountStateImpl) AcceptContract(
	txHash []byte, auditTxHash []byte) error {
	if s.isContract == false || s.nextContract == nil {
		return errors.New("Wrong contract status")
	}
	if bytes.Equal(txHash, s.nextContract.deployTxHash) == false {
		return errors.New("Wrong txHash")
	}
	s.curContract = s.nextContract
	s.curContract.status = csActive
	s.curContract.auditTxHash = auditTxHash
	s.nextContract = nil
	return nil
}

func (s *accountStateImpl) RejectContract(
	txHash []byte, auditTxHash []byte) error {
	if s.isContract == false || s.nextContract == nil {
		return errors.New("Wrong contract status")
	}
	if bytes.Equal(txHash, s.nextContract.deployTxHash) == false {
		return errors.New("Wrong txHash")
	}
	s.nextContract.status = csRejected
	s.nextContract.auditTxHash = auditTxHash
	return nil
}

func (s *accountStateImpl) APIInfo() *scoreapi.Info {
	return s.apiInfo
}

func (s *accountStateImpl) SetAPIInfo(apiInfo *scoreapi.Info) {
	s.apiInfo = apiInfo
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

	var curContract *contractSnapshotImpl
	if s.curContract != nil {
		curContract = s.curContract.getSnapshot()
	}
	var nextContract *contractSnapshotImpl
	if s.nextContract != nil {
		nextContract = s.nextContract.getSnapshot()
	}
	return &accountSnapshotImpl{
		database:      s.database,
		version:       s.version,
		balance:       s.balance.Clone(),
		fIsContract:   s.isContract,
		store:         store,
		contractOwner: contractOwner,
		apiInfo:       s.apiInfo,
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
	s.version = snapshot.version
	s.apiInfo = snapshot.apiInfo

	if snapshot.contractOwner != nil {
		s.contractOwner = common.NewAccountAddress(snapshot.contractOwner.Bytes())
	}
	if snapshot.curContract != nil {
		s.curContract = new(contractImpl)
		s.curContract.reset(snapshot.curContract)
	}
	if snapshot.nextContract != nil {
		s.nextContract = new(contractImpl)
		s.nextContract.reset(snapshot.nextContract)
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

func (s *accountStateImpl) Contract() Contract {
	if s.curContract == nil {
		return nil
	}
	return s.curContract
}

func (s *accountStateImpl) NextContract() Contract {
	if s.nextContract == nil {
		return nil
	}
	return s.nextContract
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl) AccountState {
	s := new(accountStateImpl)
	s.database = database
	if snapshot != nil {
		if err := s.Reset(snapshot); err != nil {
			return nil
		}
	} else {
		s.version = AccountVersion
	}
	return s
}

type accountROState struct {
	AccountSnapshot
	curContract  Contract
	nextContract Contract
}

func (a *accountROState) Contract() Contract {
	return a.curContract
}

func (a *accountROState) ActiveContract() Contract {
	if active := a.AccountSnapshot.ActiveContract(); active != nil {
		return newContractROState(active)
	}
	return nil
}

func (a *accountROState) NextContract() Contract {
	return a.nextContract
}

func (a *accountROState) Disable(b bool) {
	log.Panicf("accountROState().Disable() is invoked")
}

func (a *accountROState) Blacklist(b bool) {
	log.Panicf("accountROState().Blacklist() is invoked")
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

func (a *accountROState) SetAPIInfo(*scoreapi.Info) {
	log.Panicf("accountROState().SetApiInfo() is invoked")
}

func (a *accountROState) InitContractAccount(address module.Address) {
	log.Panicf("accountROState().InitContractAccount() is invoked")
}

func (a *accountROState) DeployContract(code []byte,
	eeType string, contentType string, params []byte, txHash []byte) {
	log.Panicf("accountROState().DeployContract() is invoked")
}

func (a *accountROState) AcceptContract(
	txHash []byte, auditTxHash []byte) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) RejectContract(
	txHash []byte, auditTxHash []byte) error {
	return errors.New("ReadOnlyState")
}

func newAccountROState(snapshot AccountSnapshot) AccountState {
	return &accountROState{snapshot,
		newContractROState(snapshot.Contract()),
		newContractROState(snapshot.NextContract())}
}
