package contract

import (
	"container/list"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/icon-project/goloop/service/scoreresult"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/state"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type (
	cStatus int

	ContractManager interface {
		GetHandler(from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler
		GetCallHandler(from, to module.Address,
			value, stepLimit *big.Int, method string, paramObj *codec.TypedObj) ContractHandler
		PrepareContractStore(ws state.WorldState, contract state.Contract) (ContractStore, error)
	}

	ContractStore interface {
		WaitResult() (string, error)
		Dispose()
	}

	contractStoreImpl struct {
		sc   *storageCache
		elem *list.Element // slice
		ch   chan error
	}

	storageCache struct {
		lock    sync.Mutex
		clients *list.List // contractStoreImpl list
		status  cStatus
		path    string
		err     error
		timer   *time.Timer
	}

	contractManager struct {
		lock         sync.Mutex
		db           db.Database
		storageCache map[string]*storageCache
		storeRoot    string
		log          log.Logger
	}
)

const (
	csInProgress cStatus = iota
	csComplete
)

func (sc *storageCache) push(store *contractStoreImpl) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	store.elem = sc.clients.PushBack(store)
	store.sc = sc
}

func (sc *storageCache) remove(store *contractStoreImpl) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.clients.Remove(store.elem)
}

// if complete is already called, return false
func (sc *storageCache) complete(path string, err error) bool {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	if sc.status == csComplete {
		return false
	}
	sc.path = path
	sc.err = err
	for iter := sc.clients.Front(); iter != nil; iter = iter.Next() {
		if cs, ok := iter.Value.(*contractStoreImpl); ok {
			cs.notify(err)
		}
	}
	sc.status = csComplete
	return true
}

func (cs *contractStoreImpl) WaitResult() (string, error) {
	err := <-cs.ch
	if err == nil {
		return cs.sc.path, nil
	}
	return "", err
}

func (cs *contractStoreImpl) Dispose() {
	cs.ch <- nil
	cs.sc.remove(cs)
}

func (cs *contractStoreImpl) notify(err error) {
	cs.ch <- err
}

func (cm *contractManager) GetHandler(from, to module.Address, value,
	stepLimit *big.Int, ctype int, data []byte,
) ContractHandler {
	var handler ContractHandler
	ch := newCommonHandler(from, to, value, stepLimit, cm.log)
	switch ctype {
	case CTypeTransfer:
		handler = newTransferHandler(ch)
	case CTypeCall:
		handler = newCallHandler(ch, data, false)
	case CTypeDeploy:
		handler = newDeployHandler(ch, data)
	case CTypePatch:
		handler = newPatchHandler(ch, data)
	case CTypeTransferAndCall:
		handler = &TransferAndCallHandler{
			th:          newTransferHandler(ch),
			CallHandler: newCallHandler(ch, data, false),
		}
	}
	return handler
}

func (cm *contractManager) GetCallHandler(from, to module.Address,
	value, stepLimit *big.Int, method string, paramObj *codec.TypedObj,
) ContractHandler {
	if value != nil && value.Sign() == 1 { //value > 0
		ch := newCommonHandler(from, to, value, stepLimit, cm.log)
		th := newTransferHandler(ch)
		if to.IsContract() {
			return &TransferAndCallHandler{
				th:          th,
				CallHandler: newCallHandlerFromTypedObj(ch, method, paramObj, false),
			}
		} else {
			return th
		}
	} else {
		return newCallHandlerFromTypedObj(
			newCommonHandler(from, to, value, stepLimit, cm.log),
			method, paramObj, false)
	}
}

// if path does not exist, make the path
func (cm *contractManager) storeContract(eeType state.EEType,
	code []byte, codeHash []byte, sc *storageCache) (string, error) {
	// check directory with hash, if it exists return path, nil
	defer sc.timer.Stop()
	dir := "0x" + hex.EncodeToString(codeHash)
	path := filepath.Join(cm.storeRoot, dir)
	cm.log.Debugf("[contractmanager], storeContract dir(%s), path(%s)\n", dir, path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}

	err := eeType.Store(path, code, cm.log)
	if err != nil {
		return "", err
	}

	return path, nil
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
func (cm *contractManager) PrepareContractStore(
	ws state.WorldState, contract state.Contract) (ContractStore, error) {
	cm.lock.Lock()
	codeHash := contract.CodeHash()
	hashStr := string(codeHash)
	cs := &contractStoreImpl{ch: make(chan error, 1)}
	if cacheInfo, ok := cm.storageCache[hashStr]; ok {
		if cacheInfo.status != csComplete {
			cacheInfo.push(cs)
			cm.lock.Unlock()
			return cs, nil
		}
		if _, err := os.Stat(cacheInfo.path); !os.IsNotExist(err) {
			cacheInfo.push(cs)
			cs.ch <- nil
			cm.lock.Unlock()
			return cs, nil
		}
	}
	codeBuf, err := contract.Code()
	if err != nil {
		cm.lock.Unlock()
		return nil, err
	}
	sc := &storageCache{clients: list.New(),
		status: csInProgress, timer: nil}
	timer := time.AfterFunc(scoreDecompressTimeLimit,
		func() {
			if sc.complete("", scoreresult.New(module.StatusTimeout,
				"Timeout waiting for extracting score")) == true {
				cm.lock.Lock()
				delete(cm.storageCache, hashStr)
				cm.lock.Unlock()
			}
		})
	sc.timer = timer
	cm.storageCache[hashStr] = sc
	sc.push(cs)
	cm.lock.Unlock()

	go func() {
		path, err := cm.storeContract(state.EEType(contract.EEType()), codeBuf, codeHash, sc)
		if sc.complete(path, err) == false {
			os.RemoveAll(path)
		}
	}()
	return cs, nil
}

func NewContractManager(db db.Database, contractDir string, log log.Logger) (ContractManager, error) {
	/*
		contractManager has root path of each service manager's contract file
		So contractManager has to be initialized
		after configurable root path is passed to Service Manager
	*/
	// To manage separate contract store for each chain, add chain ID to
	// parameter here and add it to storeRoot.
	var storeRoot string
	if !filepath.IsAbs(contractDir) {
		var err error
		storeRoot, err = filepath.Abs(contractDir)
		if err != nil {
			return nil, errors.UnknownError.Wrapf(err, "FAIL to get abs(%s)", contractDir)
		}
	} else {
		storeRoot = contractDir
	}
	if _, err := os.Stat(storeRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			return nil, errors.UnknownError.Wrapf(err, "FAIL to make dir(%s)", contractDir)
		}
	}
	return &contractManager{db: db, storeRoot: storeRoot,
			storageCache: make(map[string]*storageCache), log: log},
		nil
}
