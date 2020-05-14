package state

import (
	"bytes"
	"io"
	"math/big"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/common/trie/ompt"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
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
	IsEmpty() bool
	GetValue(k []byte) ([]byte, error)
	StorageChangedAfter(snapshot AccountSnapshot) bool

	IsContractOwner(owner module.Address) bool
	APIInfo() *scoreapi.Info
	Contract() ContractSnapshot
	ActiveContract() ContractSnapshot
	NextContract() ContractSnapshot
	IsDisabled() bool
	IsBlocked() bool
	ContractOwner() module.Address

	GetObjGraph(flags bool) (int, []byte, []byte, error)
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
	SetValue(k, v []byte) ([]byte, error)
	DeleteValue(k []byte) ([]byte, error)
	GetSnapshot() AccountSnapshot
	Reset(snapshot AccountSnapshot) error
	Clear()

	IsContractOwner(owner module.Address) bool
	InitContractAccount(address module.Address) bool
	DeployContract(code []byte, eeType EEType, contentType string, params []byte, txHash []byte) ([]byte, error)
	APIInfo() *scoreapi.Info
	SetAPIInfo(*scoreapi.Info)
	AcceptContract(txHash []byte, auditTxHash []byte) error
	RejectContract(txHash []byte, auditTxHash []byte) error
	Contract() Contract
	ActiveContract() Contract
	NextContract() Contract
	SetDisable(b bool)
	IsDisabled() bool
	SetBlock(b bool)
	IsBlocked() bool
	ContractOwner() module.Address

	GetObjGraph(flags bool) (int, []byte, []byte, error)
	SetObjGraph(flags bool, nextHash int, objGraph []byte) error
}

type accountSnapshotImpl struct {
	version     int
	balance     common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database

	state         int
	contractOwner *common.Address
	apiInfo       *scoreapi.Info
	curContract   *contractSnapshotImpl
	nextContract  *contractSnapshotImpl

	objGraph *objectGraph
}

func (s *accountSnapshotImpl) ContractOwner() module.Address {
	return s.contractOwner
}

func (s *accountSnapshotImpl) Version() int {
	return s.version
}

func (s *accountSnapshotImpl) ActiveContract() ContractSnapshot {
	if s.state == ASActive && s.curContract != nil && s.curContract.state == CSActive {
		return s.curContract
	}
	return nil
}

func (s *accountSnapshotImpl) IsDisabled() bool {
	if s.state&ASDisabled != 0 {
		return true
	}
	return false
}

func (s *accountSnapshotImpl) IsBlocked() bool {
	if s.state&ASBlocked != 0 {
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

func (s *accountSnapshotImpl) IsEmpty() bool {
	return s.balance.BitLen() == 0 && s.store == nil && s.contractOwner == nil
}

func (s *accountSnapshotImpl) Bytes() []byte {
	b, err := codec.MarshalToBytes(s)
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
	_, err := codec.UnmarshalFromBytes(data, s)
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
	if s.objGraph != nil {
		if err := s.objGraph.flush(); err != nil {
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
			s.balance.Cmp(&s2.balance.Int) != 0 || s.state != s2.state {
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
		if s.apiInfo.Equal(s2.apiInfo) == false {
			return false
		}
		if s.objGraph.Equal(s2.objGraph) == false {
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
		s.store.Resolve(bd)
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

func (s *accountSnapshotImpl) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	var obj *objectGraph
	obj = s.objGraph
	if flags == false {
		return obj.nextHash, obj.graphHash, nil, nil
	} else {
		if obj.graphData == nil && obj.graphHash != nil {
			bk, err := s.database.GetBucket(db.BytesByHash)
			if err != nil {
				err = errors.CriticalIOError.Wrap(err, "FailToGetBucket")
				return 0, nil, nil, err
			}
			v, err := bk.Get(obj.graphHash)
			if err != nil {
				return 0, nil, nil, err
			}
			if v == nil {
				return 0, nil, nil, errors.NotFoundError.Errorf(
					"FAIL to find graphData by graphHash(%x)", obj.graphHash)
			}
			obj.graphData = v
		}
	}
	log.Tracef("GetObjGraph flag(%t), nextHash(%d), graphHash(%#x), lenOfObjGraph(%d)\n",
		flags, obj.nextHash, obj.graphHash, len(obj.graphData))

	return obj.nextHash, obj.graphHash, obj.graphData, nil
}

const (
	accountSnapshotImplEntries     = 9
	accountSnapshotIncludeObjGraph = 11 // include object graph
)

func (s *accountSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	var storeHash []byte
	if s.store != nil {
		storeHash = s.store.Hash()
	}

	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		s.version,
		&s.balance,
		s.fIsContract,
		storeHash,
		s.state,
		s.contractOwner,
		s.apiInfo,
		s.curContract,
		s.nextContract,
	); err != nil {
		return err
	}
	if s.objGraph != nil {
		if err := e2.EncodeMulti(
			s.objGraph.nextHash,
			s.objGraph.graphHash,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *accountSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	var storeHash []byte
	var objGraph objectGraph
	if n, err := d2.DecodeMulti(
		&s.version,
		&s.balance,
		&s.fIsContract,
		&storeHash,
		&s.state,
		&s.contractOwner,
		&s.apiInfo,
		&s.curContract,
		&s.nextContract,
		&objGraph.nextHash,
		&objGraph.graphHash,
	); err == nil || err == io.EOF {
		if n == accountSnapshotIncludeObjGraph {
			s.objGraph = &objGraph
		} else if n == accountSnapshotImplEntries {
			s.objGraph = nil
		} else {
			return codec.ErrInvalidFormat
		}
	} else {
		return err
	}
	if len(storeHash) > 0 {
		s.store = trie_manager.NewImmutable(s.database, storeHash)
	}
	if s.curContract != nil {
		s.curContract.bk, _ = s.database.GetBucket(db.BytesByHash)
	}
	if s.nextContract != nil {
		s.nextContract.bk, _ = s.database.GetBucket(db.BytesByHash)
	}
	return nil
}

func (s *accountSnapshotImpl) ClearCache() {
	if s.store != nil {
		s.store.ClearCache()
	}
}

type accountStateImpl struct {
	cacheID []byte

	version    int
	database   db.Database
	balance    common.HexInt
	isContract bool

	state         int
	contractOwner module.Address
	apiInfo       *scoreapi.Info
	curContract   *contractImpl
	nextContract  *contractImpl
	store         trie.Mutable

	objGraph *objectGraph
}

type objectGraph struct {
	bk        db.Bucket
	nextHash  int
	graphHash []byte
	graphData []byte
}

func (o *objectGraph) flush() error {
	if o.bk == nil || o.graphData == nil {
		return nil
	}
	prevData, err := o.bk.Get(o.graphHash)
	if err != nil {
		return err
	}
	// already exists
	if prevData != nil {
		return nil
	}
	if err := o.bk.Set(o.graphHash, o.graphData); err != nil {
		return err
	}
	return nil
}

func (o *objectGraph) Equal(o2 *objectGraph) bool {
	if o == o2 {
		return true
	}
	if o == nil || o2 == nil {
		return false
	}
	if o.nextHash != o2.nextHash {
		return false
	}
	if !bytes.Equal(o.graphHash, o2.graphHash) {
		return false
	}
	return true
}

func (s *accountStateImpl) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	var obj *objectGraph
	obj = s.objGraph
	if flags == false {
		return obj.nextHash, obj.graphHash, nil, nil
	} else {
		if obj.graphData == nil && obj.graphHash != nil {
			bk, err := s.database.GetBucket(db.BytesByHash)
			if err != nil {
				err = errors.CriticalIOError.Wrap(err, "FailToGetBucket")
				return 0, nil, nil, err
			}
			v, err := bk.Get(obj.graphHash)
			if err != nil {
				return 0, nil, nil, err
			}
			if v == nil {
				return 0, nil, nil, errors.NotFoundError.Errorf(
					"FAIL to find graphData by graphHash(%x)", obj.graphHash)
			}
			obj.graphData = v
		}
	}
	log.Tracef("GetObjGraph flag(%t), nextHash(%d), graphHash(%#x), lenOfObjGraph(%d)\n",
		flags, obj.nextHash, obj.graphHash, len(obj.graphData))

	return obj.nextHash, obj.graphHash, obj.graphData, nil
}

func (s *accountStateImpl) SetObjGraph(flags bool, nextHash int, graphData []byte) error {
	log.Tracef("SetObjGraph flags(%t), nextHash(%d), lenOfObjGraph(%d)\n", flags, nextHash, len(graphData))
	if flags {
		hash := sha3.Sum256(graphData)
		bk, err := s.database.GetBucket(db.BytesByHash)
		if err != nil {
			return errors.CriticalIOError.Wrap(err, "FailToGetBucket")
		}
		s.objGraph = &objectGraph{
			bk:        bk,
			nextHash:  nextHash,
			graphHash: hash[:],
			graphData: graphData,
		}
	} else {
		tmp := *s.objGraph
		tmp.nextHash = nextHash
		s.objGraph = &tmp
	}
	return nil
}

func (s *accountStateImpl) ContractOwner() module.Address {
	return s.contractOwner
}

func (s *accountStateImpl) Version() int {
	return s.version
}

func (s *accountStateImpl) ActiveContract() Contract {
	if s.state == ASActive &&
		s.curContract != nil && s.curContract.state == CSActive {
		return s.curContract
	}
	return nil
}

func (s *accountStateImpl) IsDisabled() bool {
	if s.state&ASDisabled == ASDisabled {
		return true
	}
	return false
}

func (s *accountStateImpl) IsBlocked() bool {
	if s.state&ASBlocked == ASBlocked {
		return true
	}
	return false
}

func (s *accountStateImpl) SetDisable(b bool) {
	if s.isContract == true {
		if b == true {
			s.state = s.state | ASDisabled
		} else {
			s.state = s.state & ^ASDisabled
		}
	}
}

func (s *accountStateImpl) SetBlock(b bool) {
	if s.isContract == true {
		if b == true {
			s.state = s.state | ASBlocked
		} else {
			s.state = s.state & ^ASBlocked
		}
	}
}

func (s *accountStateImpl) IsContractOwner(owner module.Address) bool {
	if s.isContract == false {
		return false
	}
	return s.contractOwner.Equal(owner)
}

func (s *accountStateImpl) InitContractAccount(address module.Address) bool {
	if s.isContract == true {
		log.Debug("already Contract account")
		return false
	}
	s.isContract = true
	s.contractOwner = address
	return true
}

func (s *accountStateImpl) DeployContract(code []byte, eeType EEType, contentType string, params []byte, txHash []byte) ([]byte, error) {
	if s.isContract == false {
		return nil, nil
	}
	state := CSPending
	codeHash := sha3.Sum256(code)
	bk, err := s.database.GetBucket(db.BytesByHash)
	if err != nil {
		err = errors.CriticalIOError.Wrap(err, "FailToGetBucket")
		return nil, err
	}
	var old []byte
	if s.nextContract != nil {
		old = s.nextContract.deployTxHash
	}
	s.nextContract = &contractImpl{contractSnapshotImpl{
		bk: bk, isNew: true, state: state, contentType: contentType,
		eeType: eeType, deployTxHash: txHash, codeHash: codeHash[:],
		params: params, code: code},
	}
	return old, nil
}

func (s *accountStateImpl) AcceptContract(
	txHash []byte, auditTxHash []byte) error {
	if s.isContract == false || s.nextContract == nil {
		return scoreresult.New(module.StatusContractNotFound, "NoAvailableContract")
	}
	if bytes.Equal(txHash, s.nextContract.deployTxHash) == false {
		return errors.NotFoundError.Errorf("NoMatchedDeployTxHash(%x)(%x)", txHash, s.nextContract.deployTxHash)
	}
	s.curContract = s.nextContract
	s.curContract.state = CSActive
	s.curContract.auditTxHash = auditTxHash
	s.nextContract = nil
	return nil
}

func (s *accountStateImpl) RejectContract(
	txHash []byte, auditTxHash []byte) error {
	if s.isContract == false || s.nextContract == nil {
		return scoreresult.New(module.StatusContractNotFound, "NoAvailableContract")
	}
	if bytes.Equal(txHash, s.nextContract.deployTxHash) == false {
		return errors.NotFoundError.Errorf("NoMatchedDeployTxHash(%x)(%x)", txHash, s.nextContract.deployTxHash)
	}
	s.nextContract.state = CSRejected
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
		contractOwner = common.NewAddress(s.contractOwner.Bytes())
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
		state:         s.state,
		contractOwner: contractOwner,
		apiInfo:       s.apiInfo,
		curContract:   curContract,
		nextContract:  nextContract,
		objGraph:      s.objGraph,
	}
}

// ensureCache set cache of the store if cacheID is specified.
// If it didn't enable cache of the accounts, cacheID would be nil.
func (s *accountStateImpl) attachCacheForStore() {
	if s.cacheID != nil && s.store != nil {
		if cache := cache.AccountNodeCacheOf(s.database, s.cacheID); cache != nil {
			ompt.SetCacheOfMutable(s.store, cache)
		}
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
	s.state = snapshot.state

	if snapshot.contractOwner != nil {
		s.contractOwner = common.NewAddress(snapshot.contractOwner.Bytes())
	}
	if snapshot.curContract != nil {
		s.curContract = new(contractImpl)
		s.curContract.reset(snapshot.curContract)
	}
	if snapshot.nextContract != nil {
		s.nextContract = new(contractImpl)
		s.nextContract.reset(snapshot.nextContract)
	}
	s.objGraph = snapshot.objGraph
	if snapshot.store == nil {
		s.store = nil
		return nil
	}
	if s.store == nil {
		s.store = trie_manager.NewMutableFromImmutable(snapshot.store)
		s.attachCacheForStore()
		return nil
	}
	if err := s.store.Reset(snapshot.store); err != nil {
		log.Panicf("Fail to make accountStateImpl err=%v", err)
	}
	return nil
}

func (s *accountStateImpl) Clear() {
	s.balance.SetInt64(0)
	s.isContract = false
	s.version = AccountVersion
	s.apiInfo = nil
	s.contractOwner = nil
	s.curContract = nil
	s.nextContract = nil
	s.store = nil
}

func (s *accountStateImpl) GetValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountStateImpl) SetValue(k, v []byte) ([]byte, error) {
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
		s.attachCacheForStore()
	}
	return s.store.Set(k, v)
}

func (s *accountStateImpl) DeleteValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
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

func (s *accountStateImpl) ClearCache() {
	if s.store != nil {
		s.store.ClearCache()
	}
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl, cacheID []byte) AccountState {
	s := &accountStateImpl{
		cacheID:  cacheID,
		database: database,
	}
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
	if a.IsBlocked() == true || a.IsDisabled() == true {
		return nil
	}

	if active := a.AccountSnapshot.ActiveContract(); active != nil {
		return newContractROState(active)
	}
	return nil
}

func (a *accountROState) NextContract() Contract {
	return a.nextContract
}

func (a *accountROState) SetDisable(b bool) {
	log.Panic("accountROState().SetDisable() is invoked")
}

func (a *accountROState) SetBlock(b bool) {
	log.Panic("accountROState().SetBlock() is invoked")
}

func (a *accountROState) SetBalance(v *big.Int) {
	log.Panic("accountROState().SetBalance() is invoked")
}

func (a *accountROState) SetValue(k, v []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) DeleteValue(k []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) GetSnapshot() AccountSnapshot {
	return a.AccountSnapshot
}

func (a *accountROState) Reset(snapshot AccountSnapshot) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) SetAPIInfo(*scoreapi.Info) {
	log.Panic("accountROState().SetApiInfo() is invoked")
}

func (a *accountROState) InitContractAccount(address module.Address) bool {
	log.Panic("accountROState().InitContractAccount() is invoked")
	return false
}

func (a *accountROState) DeployContract(code []byte, eeType EEType, contentType string, params []byte, txHash []byte) ([]byte, error) {
	log.Panic("accountROState().DeployContract() is invoked")
	return nil, nil
}

func (a *accountROState) AcceptContract(
	txHash []byte, auditTxHash []byte) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) RejectContract(
	txHash []byte, auditTxHash []byte) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) Clear() {
	// nothing to do
}

func (a *accountROState) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	return nil
}

func newAccountROState(snapshot AccountSnapshot) AccountState {
	if snapshot == nil {
		snapshot = new(accountSnapshotImpl)
	}
	return &accountROState{snapshot,
		newContractROState(snapshot.Contract()),
		newContractROState(snapshot.NextContract())}
}
