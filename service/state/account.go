package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/cache"
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

const (
	ASDisabled = 1 << iota
	ASBlocked
	ASUseSystemDeposit
)

var AccountType = reflect.TypeOf((*accountSnapshotImpl)(nil))

type AccountData interface {
	Version() int
	GetBalance() *big.Int
	IsContract() bool
	IsEmpty() bool
	IsDisabled() bool
	IsBlocked() bool
	UseSystemDeposit() bool
	GetValue(k []byte) ([]byte, error)
	IsContractOwner(owner module.Address) bool
	ContractOwner() module.Address
	APIInfo() (*scoreapi.Info, error)
	CanAcceptTx(pc PayContext) bool
	CheckDeposit(pc PayContext) bool
	GetObjGraph(hash []byte, flags bool) (int, []byte, []byte, error)
	GetDepositInfo(dc DepositContext, v module.JSONVersion) (map[string]interface{}, error)
}

// AccountSnapshot represents immutable account state
// It can be get from AccountState or WorldSnapshot.
type AccountSnapshot interface {
	AccountData
	trie.Object
	StorageChangedAfter(snapshot AccountSnapshot) bool
	Contract() ContractSnapshot
	ActiveContract() ContractSnapshot
	NextContract() ContractSnapshot
}

// AccountState represents mutable account state.
// You may change account state with this object. It can be get from
// WorldState. Changes in this object will be retrieved by WorldState.
// Of course, it also can be changed by WorldState.
type AccountState interface {
	AccountData
	MigrateForRevision(rev module.Revision) error
	SetBalance(v *big.Int)
	SetValue(k, v []byte) ([]byte, error)
	DeleteValue(k []byte) ([]byte, error)
	GetSnapshot() AccountSnapshot
	Reset(snapshot AccountSnapshot) error
	Clear()

	SetContractOwner(owner module.Address) error
	InitContractAccount(address module.Address) bool
	DeployContract(code []byte, eeType EEType, contentType string, params []byte, txHash []byte) ([]byte, error)
	SetAPIInfo(*scoreapi.Info)
	ActivateNextContract() error
	AcceptContract(txHash []byte, auditTxHash []byte) error
	RejectContract(txHash []byte, auditTxHash []byte) error
	Contract() ContractState
	ActiveContract() ContractState
	NextContract() ContractState
	SetDisable(b bool)
	SetBlock(b bool)
	SetUseSystemDeposit(yn bool) error
	SetObjGraph(id []byte, flags bool, nextHash int, objGraph []byte) error

	AddDeposit(dc DepositContext, value *big.Int) error
	WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error)
	PaySteps(pc PayContext, steps *big.Int) (*big.Int, *big.Int, error)
}

const (
	ExObjectGraph int = 1 << iota
	ExDepositInfo
)

var zeroBalance big.Int

type accountStore interface {
	Get(k []byte) ([]byte, error)
}

type accountData struct {
	database      db.Database
	version       int
	balance       *big.Int
	isContract    bool
	state         int
	contractOwner *common.Address
	apiInfo       apiInfoStore
	curContract   *contract
	nextContract  *contract
	store         accountStore
	deposits      depositList
	objCache      objectGraphCache
}

func (s *accountData) ContractOwner() module.Address {
	if s.contractOwner == nil {
		return nil
	}
	return s.contractOwner
}

func (s *accountData) Version() int {
	return s.version
}

func (s *accountData) IsDisabled() bool {
	return s.state&ASDisabled != 0
}

func (s *accountData) IsBlocked() bool {
	return s.state&ASBlocked != 0
}

func (s *accountData) UseSystemDeposit() bool {
	return s.state&ASUseSystemDeposit != 0
}

func (s *accountData) IsActive() bool {
	return s.state&(ASDisabled|ASBlocked) == 0
}

func (s *accountData) GetBalance() *big.Int {
	return s.balance
}

func (s *accountData) IsContract() bool {
	return s.isContract
}

func (s *accountData) GetValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountData) IsEmpty() bool {
	return s.balance.Sign() == 0 && s.store == nil && (!s.isContract) && s.state == 0
}

func (s *accountData) IsContractOwner(owner module.Address) bool {
	if !s.isContract || owner == nil || s.contractOwner == nil {
		return false
	}
	return s.contractOwner.Equal(owner)
}

func (s *accountData) APIInfo() (*scoreapi.Info, error) {
	return s.apiInfo.Get()
}

func (s *accountData) GetObjGraph(hash []byte, flags bool) (int, []byte, []byte, error) {
	og := s.objCache.Get(hash)
	return og.Get(flags)
}

func (s *accountData) CanAcceptTx(pc PayContext) bool {
	if s.IsContract() && (s.IsDisabled() || s.IsBlocked()) {
		return false
	}
	return s.CheckDeposit(pc)
}

func (s *accountData) CheckDeposit(pc PayContext) bool {
	if pc.FeeSharingEnabled() {
		if s.deposits.Has() {
			return s.deposits.CanPay(pc)
		}
	}
	return true
}

func (s *accountData) GetDepositInfo(dc DepositContext, v module.JSONVersion) (
	map[string]interface{}, error,
) {
	return s.deposits.ToJSON(dc, v)
}

type accountSnapshotImpl struct {
	accountData
	objGraph *objectGraph
}

func (s *accountSnapshotImpl) String() string {
	if s.IsContract() {
		return fmt.Sprintf("Account{balance=%d state=%d cur=%v next=%v store=%v obj=%v}",
			s.balance, s.state, s.curContract, s.nextContract, s.store, s.objGraph)
	} else {
		return fmt.Sprintf("Account{balance=%d state=%d}", s.balance, s.state)
	}
}

func (s *accountSnapshotImpl) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(s)
}

func (s *accountSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	var storeHash []byte
	if s.store != nil {
		storeHash = s.store.(trie.Immutable).Hash()
	}

	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		s.version,
		s.balance,
		s.isContract,
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

func (s *accountSnapshotImpl) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.BC.UnmarshalFromBytes(data, s)
	return err
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
		&s.isContract,
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
		s.store.(trie.Immutable).ClearCache()
	}
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
		if s.isContract != s2.isContract ||
			s.balance.Cmp(s2.balance) != 0 || s.state != s2.state {
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
		return s.store.(trie.Immutable).Equal(s2.store.(trie.Immutable), false)
	} else {
		log.Panicf("Replacing accountSnapshot with other object(%T)", object)
	}
	return false
}

func (s *accountSnapshotImpl) Resolve(bd merkle.Builder) error {
	if err := s.apiInfo.Resolve(bd); err != nil {
		return err
	}
	if s.store != nil {
		s.store.(trie.Immutable).Resolve(bd)
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
	if s.objGraph != nil {
		if err := s.objGraph.Resolve(bd); err != nil {
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
		if s2.store.(trie.Immutable).Equal(s.store.(trie.Immutable), false) {
			return false
		}
	}
	return true
}

func (s *accountSnapshotImpl) Contract() ContractSnapshot {
	if s.curContract == nil {
		return nil
	}
	return s.curContract
}

func (s *accountSnapshotImpl) ActiveContract() ContractSnapshot {
	if s.IsActive() &&
		s.curContract != nil && s.curContract.state == CSActive {
		return s.curContract
	}
	return nil
}

func (s *accountSnapshotImpl) NextContract() ContractSnapshot {
	if s.nextContract == nil {
		return nil
	}
	return s.nextContract
}

func (s *accountSnapshotImpl) Store() trie.Immutable {
	store, _ := s.store.(trie.Immutable)
	return store
}

func newAccountSnapshot(dbase db.Database) *accountSnapshotImpl {
	return &accountSnapshotImpl{
		accountData: accountData{
			version:  AccountVersion,
			balance:  &zeroBalance,
			database: dbase,
		},
	}
}

type accountStateImpl struct {
	accountData
	store    trie.Mutable
	last     *accountSnapshotImpl
	key      []byte
	useCache bool
}

func (s *accountStateImpl) markDirty() {
	s.last = nil
}

func (s *accountStateImpl) SetObjGraph(id []byte, flags bool, nextHash int, objGraph []byte) error {
	obj := s.objCache.Get(id)
	if no, err := obj.Changed(s.database, flags, nextHash, objGraph); err != nil {
		return err
	} else {
		s.objCache.Set(id, no)
		s.markDirty()
		return nil
	}
}

func (s *accountStateImpl) ActiveContract() ContractState {
	if s.IsActive() &&
		s.curContract != nil && s.curContract.state == CSActive {
		return s.curContract
	}
	return nil
}

func (s *accountStateImpl) SetDisable(b bool) {
	if s.isContract == true {
		if ((s.state & ASDisabled) != 0) != b {
			s.state = s.state ^ ASDisabled
			s.markDirty()
		}
	}
}

func (s *accountStateImpl) SetBlock(b bool) {
	if ((s.state & ASBlocked) != 0) != b {
		s.state = s.state ^ ASBlocked
		s.markDirty()
	}
}

func (s *accountStateImpl) SetUseSystemDeposit(yn bool) error {
	if !s.isContract {
		return scoreresult.ContractNotFoundError.New("NotContract")
	}
	if ((s.state & ASUseSystemDeposit) != 0) != yn {
		s.state = s.state ^ ASUseSystemDeposit
		s.markDirty()
	}
	return nil
}

func (s *accountStateImpl) SetContractOwner(owner module.Address) error {
	if !s.isContract {
		return scoreresult.ContractNotFoundError.New("NotContract")
	}
	if !s.contractOwner.Equal(owner) {
		s.markDirty()
		s.contractOwner = common.AddressToPtr(owner)
	}
	return nil
}

func (s *accountStateImpl) InitContractAccount(address module.Address) bool {
	if s.isContract == true {
		log.Debug("already Contract account")
		return false
	}
	s.markDirty()
	s.isContract = true
	s.contractOwner = common.AddressToPtr(address)
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
		if s.nextContract.Status() == CSActive {
			return nil, scoreresult.AccessDeniedError.New("AlreadyDeploying")
		}
		old = s.nextContract.deployTxHash
	}
	s.nextContract = &contract{
		bk: bk, needFlush: true, state: state, contentType: contentType,
		eeType: eeType, deployTxHash: txHash, codeHash: codeHash[:],
		params: params, code: code,
		markDirty: s.markDirty,
	}
	s.markDirty()
	return old, nil
}

func (s *accountStateImpl) ActivateNextContract() error {
	if s.nextContract == nil {
		return scoreresult.InvalidParameterError.New("NoNextContract")
	}
	if s.nextContract.state == CSActive {
		return scoreresult.InvalidParameterError.New("InvalidNextContract")
	}
	if s.curContract != nil {
		s.curContract.state = CSInactive
	}
	s.nextContract.state = CSActive
	s.markDirty()
	return nil
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
	s.markDirty()
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
	s.markDirty()
	return nil
}

func (s *accountStateImpl) MigrateForRevision(rev module.Revision) error {
	v := accountVersionForRevision(rev)
	if v > s.version {
		if s.version < AccountVersion2 && v >= AccountVersion2 {
			if err := s.apiInfo.ResetDB(s.database); err != nil {
				return err
			}
			s.apiInfo.dirty = true
		}
		s.version = v
		s.markDirty()
	}
	return nil
}

func (s *accountStateImpl) SetAPIInfo(apiInfo *scoreapi.Info) {
	s.apiInfo.Set(apiInfo)
	s.markDirty()
}

func (s *accountStateImpl) SetBalance(v *big.Int) {
	if s.balance.Cmp(v) != 0 {
		s.balance = v
		s.markDirty()
	}
}

func (s *accountStateImpl) GetSnapshot() AccountSnapshot {
	if s.last != nil {
		return s.last
	}

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
	s.last = &accountSnapshotImpl{
		accountData: accountData{
			database:      s.database,
			version:       s.version,
			balance:       s.balance,
			isContract:    s.isContract,
			store:         store,
			state:         s.state,
			contractOwner: s.contractOwner,
			apiInfo:       s.apiInfo,
			curContract:   s.curContract.getSnapshot(),
			nextContract:  s.nextContract.getSnapshot(),
			objCache:      s.objCache.Clone(),
			deposits:      s.deposits.Clone(),
		},
		objGraph: objGraph,
	}
	return s.last
}

// attachCacheForStore enable cache of the store if useCache is true
func (s *accountStateImpl) attachCacheForStore() {
	if s.useCache && s.store != nil {
		if cache := cache.AccountNodeCacheOf(s.database, s.key); cache != nil {
			trie_manager.SetCacheOfMutable(s.store, cache)
		}
	}
}

func (s *accountStateImpl) Reset(isnapshot AccountSnapshot) error {
	snapshot, ok := isnapshot.(*accountSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	if s.last == snapshot {
		return nil
	}
	s.last = snapshot

	s.balance = snapshot.balance
	s.isContract = snapshot.isContract
	s.version = snapshot.version
	s.apiInfo = snapshot.apiInfo
	s.state = snapshot.state
	s.contractOwner = snapshot.contractOwner
	s.curContract = newContractState(snapshot.curContract, s.markDirty)
	s.nextContract = newContractState(snapshot.nextContract, s.markDirty)
	s.objCache = snapshot.objCache.Clone()
	s.deposits = snapshot.deposits.Clone()
	if snapshot.store == nil {
		s.store = nil
		s.accountData.store = nil
		return nil
	}
	store := snapshot.store.(trie.Immutable)
	if s.store == nil {
		s.store = trie_manager.NewMutableFromImmutable(store)
		s.accountData.store = s.store
		s.attachCacheForStore()
		return nil
	}
	if err := s.store.Reset(store); err != nil {
		log.Panicf("Fail to make accountStateImpl err=%v", err)
	}
	return nil
}

func (s *accountStateImpl) Clear() {
	*s = accountStateImpl{
		key:      s.key,
		useCache: s.useCache,
		accountData: accountData{
			database: s.database,
			version:  AccountVersion,
			balance:  &zeroBalance,
		},
	}
}

func (s *accountStateImpl) SetValue(k, v []byte) ([]byte, error) {
	if len(v) == 0 {
		return s.DeleteValue(k)
	}
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
		s.accountData.store = s.store
		s.attachCacheForStore()
	}
	if old, err := s.store.Set(k, v); err == nil {
		s.markDirty()
		return old, nil
	} else {
		return nil, err
	}
}

func (s *accountStateImpl) DeleteValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	if old, err := s.store.Delete(k); err == nil && len(old) > 0 {
		s.markDirty()
		return old, nil
	} else {
		return nil, err
	}
}

func (s *accountStateImpl) Contract() ContractState {
	if s.curContract == nil {
		return nil
	}
	return s.curContract
}

func (s *accountStateImpl) NextContract() ContractState {
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
	if err := s.deposits.AddDeposit(dc, value); err == nil {
		s.markDirty()
		return nil
	} else {
		return err
	}
}

func (s *accountStateImpl) WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error) {
	amount, fee, err := s.deposits.WithdrawDeposit(dc, id, value)
	if err != nil {
		return nil, nil, err
	}
	s.markDirty()
	return amount, fee, nil
}

func (s *accountStateImpl) PaySteps(pc PayContext, steps *big.Int) (*big.Int, *big.Int, error) {
	if pc.FeeSharingEnabled() && s.deposits.Has() {
		s.markDirty()
		paidSteps, stepsByDeposit := s.deposits.PaySteps(pc, steps)
		return paidSteps, stepsByDeposit, nil
	}
	return nil, nil, nil
}

func newAccountState(database db.Database, snapshot AccountSnapshot, key []byte, useCache bool) AccountState {
	s := &accountStateImpl{
		accountData: accountData{
			database: database,
		},
		key:      key,
		useCache: useCache,
	}
	if snapshot != nil {
		if err := s.Reset(snapshot); err != nil {
			return nil
		}
	} else {
		s.version = AccountVersion
		s.balance = &zeroBalance
	}
	return s
}

type accountROState struct {
	AccountSnapshot
	curContract  ContractState
	nextContract ContractState
}

func (a *accountROState) Contract() ContractState {
	return a.curContract
}

func (a *accountROState) ActiveContract() ContractState {
	if a.IsBlocked() == true || a.IsDisabled() == true {
		return nil
	}

	if active := a.AccountSnapshot.ActiveContract(); active != nil {
		return newContractROState(active)
	}
	return nil
}

func (a *accountROState) NextContract() ContractState {
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

func (a *accountROState) SetUseSystemDeposit(b bool) error {
	log.Panic("accountROState().SetUseSystemDeposit() is invoked")
	return errors.InvalidStateError.New("ReadOnlyState")
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

func (a *accountROState) ActivateNextContract() error {
	return errors.InvalidStateError.New("ReadOnlyState")
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

func (a *accountROState) PaySteps(pc PayContext, steps *big.Int) (*big.Int, *big.Int, error) {
	return nil, nil, errors.InvalidStateError.New("ReadOnlyState")
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
