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
	AccountVersion2
	AccountVersion = AccountVersion1
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
	APIInfo() (*scoreapi.Info, error)
	Contract() ContractSnapshot
	ActiveContract() ContractSnapshot
	NextContract() ContractSnapshot
	IsDisabled() bool
	IsBlocked() bool
	ContractOwner() module.Address

	GetObjGraph(hash []byte, flags bool) (int, []byte, []byte, error)

	CanAcceptTx(pc PayContext) bool
	GetDepositInfo(dc DepositContext, v module.JSONVersion) (map[string]interface{}, error)
}

// AccountState represents mutable account state.
// You may change account state with this object. It can be get from
// WorldState. Changes in this object will be retrieved by WorldState.
// Of course, it also can be changed by WorldState.
type AccountState interface {
	Version() int
	MigrateForRevision(rev module.Revision) error
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
	SetContractOwner(owner module.Address) error
	InitContractAccount(address module.Address) bool
	DeployContract(code []byte, eeType EEType, contentType string, params []byte, txHash []byte) ([]byte, error)
	APIInfo() (*scoreapi.Info, error)
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

	GetObjGraph(id []byte, flags bool) (int, []byte, []byte, error)
	SetObjGraph(id []byte, flags bool, nextHash int, objGraph []byte) error

	AddDeposit(dc DepositContext, value *big.Int) error
	WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error)
	PaySteps(dc DepositContext, steps *big.Int) (*big.Int, error)
	CanAcceptTx(pc PayContext) bool
	GetDepositInfo(dc DepositContext, v module.JSONVersion) (map[string]interface{}, error)
}

const (
	ExObjectGraph int = 1 << iota
	ExDepositInfo
)

type accountSnapshotImpl struct {
	version     int
	balance     *common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database

	state         int
	contractOwner *common.Address
	apiInfo       apiInfoStore
	curContract   *contractSnapshotImpl
	nextContract  *contractSnapshotImpl

	objCache objectGraphCache
	objGraph *objectGraph
	deposits depositList
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
	return &s.balance.Int
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
	if err := s.apiInfo.Flush(); err != nil {
		return err
	}
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
		if s.version != s2.version {
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
		if s.apiInfo.Equal(&s2.apiInfo) == false {
			return false
		}
		if s.objGraph.Equal(s2.objGraph) == false {
			return false
		}
		if s.deposits.Equal(s2.deposits) == false {
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
	if err := s.apiInfo.Resolve(bd); err != nil {
		return err
	}
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
			return false
		}
		if s.store == nil || s2.store == nil {
			return true
		}
		if s2.store.Equal(s.store, false) {
			return false
		}
	}
	return true
}

func (s *accountSnapshotImpl) IsContractOwner(owner module.Address) bool {
	if s.fIsContract == false {
		return false
	}
	return s.contractOwner.Equal(owner)
}

func (s *accountSnapshotImpl) APIInfo() (*scoreapi.Info, error) {
	return s.apiInfo.Get()
}

func (s *accountSnapshotImpl) GetObjGraph(hash []byte, flags bool) (int, []byte, []byte, error) {
	og := s.objCache.Get(hash)
	return og.Get(flags)
}

func (s *accountSnapshotImpl) CanAcceptTx(pc PayContext) bool {
	if pc.FeeSharingEnabled() {
		if s.deposits.Has() {
			return s.deposits.CanPay(pc)
		}
	}
	return true
}

func (s *accountSnapshotImpl) GetDepositInfo(dc DepositContext, v module.JSONVersion) (
	map[string]interface{}, error,
) {
	return s.deposits.ToJSON(dc, v)
}

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
		s.balance,
		s.fIsContract,
		storeHash,
		s.state,
		s.contractOwner,
		&s.apiInfo,
		s.curContract,
		s.nextContract,
	); err != nil {
		return err
	}

	flag := s.extensionFlag()
	if flag != 0 {
		if err := e2.Encode(flag); err != nil {
			return err
		}
		if (flag & ExObjectGraph) != 0 {
			if err := e2.Encode(s.objGraph); err != nil {
				return err
			}
		}
		if (flag & ExDepositInfo) != 0 {
			if err := e2.Encode(s.deposits); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *accountSnapshotImpl) extensionFlag() int {
	var flag int
	if s.objGraph != nil {
		flag |= ExObjectGraph
	}
	if s.deposits.Has() {
		flag |= ExDepositInfo
	}
	return flag
}

func (s *accountSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	var storeHash []byte
	if err := d2.Decode(&s.version); err != nil {
		return err
	}
	if s.version >= AccountVersion2 {
		if err := s.apiInfo.ResetDB(s.database); err != nil {
			return err
		}
	}
	if _, err := d2.DecodeMulti(
		&s.balance,
		&s.fIsContract,
		&storeHash,
		&s.state,
		&s.contractOwner,
		&s.apiInfo,
		&s.curContract,
		&s.nextContract,
	); err != nil {
		return errors.Wrap(err, "Fail to decode accountSnapshot")
	}

	if len(storeHash) > 0 {
		s.store = trie_manager.NewImmutable(s.database, storeHash)
	}
	if err := s.curContract.ResetDB(s.database); err != nil {
		return err
	}
	if err := s.nextContract.ResetDB(s.database); err != nil {
		return err
	}

	var extension int
	if err := d2.Decode(&extension); err != nil && err != io.EOF {
		return errors.Wrap(codec.ErrInvalidFormat, "Fail to decode extension")
	} else if err == nil {
		if (extension & ExObjectGraph) != 0 {
			if err := d2.Decode(&s.objGraph); err != nil {
				return errors.Wrap(codec.ErrInvalidFormat, "Fail to decode objectGraph")
			}
			if err := s.objGraph.ResetDB(s.database); err != nil {
				return err
			}
			s.objCache.Set(s.curContract.CodeID(), s.objGraph)
		}

		if (extension & ExDepositInfo) != 0 {
			if err := d2.Decode(&s.deposits); err != nil {
				return errors.Wrap(codec.ErrInvalidFormat, "Fail to decode deposits")
			}
		}
	}
	return nil
}

func (s *accountSnapshotImpl) ClearCache() {
	if s.store != nil {
		s.store.ClearCache()
	}
}

func newAccountSnapshot(dbase db.Database) *accountSnapshotImpl {
	return &accountSnapshotImpl{
		version:  AccountVersion,
		balance:  common.HexIntZero,
		database: dbase,
	}
}

type accountStateImpl struct {
	key      []byte
	useCache bool

	version    int
	database   db.Database
	balance    *common.HexInt
	isContract bool

	state         int
	contractOwner module.Address
	apiInfo       apiInfoStore
	curContract   *contractImpl
	nextContract  *contractImpl
	store         trie.Mutable

	objCache objectGraphCache
	deposits depositList
}

func (s *accountStateImpl) GetObjGraph(id []byte, flags bool) (int, []byte, []byte, error) {
	obj := s.objCache.Get(id)
	return obj.Get(flags)
}

func (s *accountStateImpl) SetObjGraph(id []byte, flags bool, nextHash int, objGraph []byte) error {
	obj := s.objCache.Get(id)
	if no, err := obj.Changed(s.database, flags, nextHash, objGraph); err != nil {
		return err
	} else {
		s.objCache.Set(id, no)
		return nil
	}
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

func (s *accountStateImpl) SetContractOwner(owner module.Address) error {
	if !s.isContract {
		return scoreresult.ContractNotFoundError.New("NotContract")
	}
	s.contractOwner = owner
	return nil
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

func (s *accountStateImpl) APIInfo() (*scoreapi.Info, error) {
	return s.apiInfo.Get()
}

func (s *accountStateImpl) MigrateForRevision(rev module.Revision) error {
	v := accountVersionForRevision(rev)
	return s.migrate(v)
}

func (s *accountStateImpl) migrate(v int) error {
	if v > s.version {
		if s.version < AccountVersion2 && v >= AccountVersion2 {
			if err := s.apiInfo.ResetDB(s.database); err != nil {
				return err
			}
			s.apiInfo.dirty = true
		}
		s.version = v
	}
	return nil
}

func (s *accountStateImpl) SetAPIInfo(apiInfo *scoreapi.Info) {
	s.apiInfo.Set(apiInfo)
}

func (s *accountStateImpl) GetBalance() *big.Int {
	return &s.balance.Int
}

func (s *accountStateImpl) SetBalance(v *big.Int) {
	nv := new(common.HexInt)
	nv.Set(v)
	s.balance = nv
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
	var objGraph *objectGraph
	if s.curContract != nil {
		objGraph = s.objCache.Get(s.curContract.CodeID())
	}
	return &accountSnapshotImpl{
		database:      s.database,
		version:       s.version,
		balance:       s.balance,
		fIsContract:   s.isContract,
		store:         store,
		state:         s.state,
		contractOwner: common.AddressToPtr(s.contractOwner),
		apiInfo:       s.apiInfo,
		curContract:   s.curContract.getSnapshot(),
		nextContract:  s.nextContract.getSnapshot(),
		objGraph:      objGraph,
		objCache:      s.objCache.Clone(),
		deposits:      s.deposits.Clone(),
	}
}

// attachCacheForStore enable cache of the store if useCache is true
func (s *accountStateImpl) attachCacheForStore() {
	if s.useCache && s.store != nil {
		if cache := cache.AccountNodeCacheOf(s.database, s.key); cache != nil {
			ompt.SetCacheOfMutable(s.store, cache)
		}
	}
}

func (s *accountStateImpl) Reset(isnapshot AccountSnapshot) error {
	snapshot, ok := isnapshot.(*accountSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	s.balance = snapshot.balance
	s.isContract = snapshot.fIsContract
	s.version = snapshot.version
	s.apiInfo = snapshot.apiInfo
	s.state = snapshot.state
	s.contractOwner = snapshot.contractOwner

	if snapshot.curContract != nil {
		s.curContract = new(contractImpl)
		s.curContract.reset(snapshot.curContract)
	} else {
		s.curContract = nil
	}
	if snapshot.nextContract != nil {
		s.nextContract = new(contractImpl)
		s.nextContract.reset(snapshot.nextContract)
	} else {
		s.nextContract = nil
	}
	s.objCache = snapshot.objCache.Clone()
	s.deposits = snapshot.deposits.Clone()
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
	s.balance = common.HexIntZero
	s.isContract = false
	s.version = AccountVersion
	s.apiInfo = apiInfoStore{}
	s.contractOwner = nil
	s.curContract = nil
	s.nextContract = nil
	s.store = nil
	s.deposits = nil
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

func (s *accountStateImpl) AddDeposit(dc DepositContext, value *big.Int) error {
	return s.deposits.AddDeposit(dc, value)
}

func (s *accountStateImpl) WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error) {
	amount, fee, err := s.deposits.WithdrawDeposit(dc, id, value)
	if err != nil {
		return nil, nil, err
	}
	balance := new(common.HexInt)
	balance.Add(&s.balance.Int, amount)
	s.balance = balance
	return amount, fee, nil
}

func (s *accountStateImpl) PaySteps(dc DepositContext, steps *big.Int) (*big.Int, error) {
	if s.deposits.Has() {
		return s.deposits.PaySteps(dc, steps), nil
	}
	return nil, nil
}

func (s *accountStateImpl) CanAcceptTx(pc PayContext) bool {
	if pc.FeeSharingEnabled() {
		if s.deposits.Has() {
			return s.deposits.CanPay(pc)
		}
	}
	return true
}

func (s *accountStateImpl) GetDepositInfo(dc DepositContext, v module.JSONVersion) (
	map[string]interface{}, error,
) {
	return s.deposits.ToJSON(dc, v)
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl, key []byte, useCache bool) AccountState {
	s := &accountStateImpl{
		key:      key,
		useCache: useCache,
		database: database,
	}
	if snapshot != nil {
		if err := s.Reset(snapshot); err != nil {
			return nil
		}
	} else {
		s.version = AccountVersion
		s.balance = common.HexIntZero
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

func (a *accountROState) SetContractOwner(owner module.Address) error {
	log.Panic("accountROState().SetOwner() is invoked")
	return errors.InvalidStateError.New("ReadOnlyState")
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

func (a *accountROState) MigrateForRevision(rev module.Revision) error {
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

func (a *accountROState) SetObjGraph(hash []byte, flags bool, nextHash int, objGraph []byte) error {
	return nil
}

func (a *accountROState) AddDeposit(dc DepositContext, value *big.Int) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error) {
	return nil, nil, errors.InvalidStateError.New("ReadOnlyState")
}

func (a *accountROState) PaySteps(dc DepositContext, steps *big.Int) (*big.Int, error) {
	return nil, errors.InvalidStateError.New("ReadOnlyState")
}

func newAccountROState(dbase db.Database, snapshot AccountSnapshot) AccountState {
	if snapshot == nil {
		snapshot = newAccountSnapshot(dbase)
	}
	return &accountROState{snapshot,
		newContractROState(snapshot.Contract()),
		newContractROState(snapshot.NextContract())}
}

func accountVersionForRevision(rev module.Revision) int {
	if rev.UseCompactAPIInfo() {
		return AccountVersion2
	} else {
		return AccountVersion1
	}
}
