package contract

import (
	"archive/zip"
	"bytes"
	"container/list"
	"encoding/hex"
	"github.com/icon-project/goloop/service/scoreresult"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

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
	tmpRoot                        = "tmp"
	contractRoot                   = "contract"
	contractPythonRootFile         = "package.json"
	csInProgress           cStatus = iota
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

const tryTmpNum = 10

// if path does not exist, make the path
func (cm *contractManager) storeContract(eeType string, code []byte, codeHash []byte, sc *storageCache) (string, error) {
	var path string
	defer sc.timer.Stop()
	contractDir := "hx" + hex.EncodeToString(codeHash)
	path = filepath.Join(cm.storeRoot, contractDir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}

	var tmpPath string
	var i int
	for i = 0; i < tryTmpNum; i++ {
		tmpPath = filepath.Join(cm.storeRoot, tmpRoot, contractDir+strconv.Itoa(i))
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(tmpPath); err != nil {
				break
			}
		} else {
			break
		}
	}
	if i == tryTmpNum {
		return "", scoreresult.Errorf(module.StatusSystemError, "Fail to create temporary directory")
	}
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return "", scoreresult.WithStatus(err, module.StatusSystemError)
	}
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return "", scoreresult.WithStatus(err, module.StatusSystemError)
	}

	switch eeType {
	case "python":
		findRoot := false
		rootDir := ""
		for _, zipFile := range zipReader.File {
			if info := zipFile.FileInfo(); info.IsDir() {
				continue
			}
			if findRoot == false &&
				filepath.Base(zipFile.Name) == contractPythonRootFile {
				rootDir = filepath.Dir(zipFile.Name)
				findRoot = true
			}
			storePath := filepath.Join(tmpPath, zipFile.Name)
			storeDir := filepath.Dir(storePath)
			if _, err := os.Stat(storeDir); os.IsNotExist(err) {
				os.MkdirAll(storeDir, 0755)
			}
			reader, err := zipFile.Open()
			if err != nil {
				return "", errors.Wrap(err, "Fail to open zip file")
			}
			buf, err := ioutil.ReadAll(reader)
			if err != nil {
				err = reader.Close()
				return "", errors.Wrap(err, "Fail to read zip file")
			}
			if err = ioutil.WriteFile(storePath, buf, 0755); err != nil {
				return "", errors.Wrapf(err, "Fail to write file. path(%s)\n", storePath)
			}
			err = reader.Close()
			if err != nil {
				return "", errors.Wrap(err, "Fail to close zip file")
			}
		}
		if findRoot == false {
			os.RemoveAll(tmpPath)
			return "", scoreresult.Errorf(module.StatusIllegalFormat,
				"Root file does not exist(required:%s)\n", contractPythonRootFile)
		}
		contractRoot := filepath.Join(tmpPath, rootDir)
		sc.lock.Lock()
		if sc.status == csComplete {
			sc.lock.Unlock()
			os.Remove(contractRoot)
			return "", nil
		}
		sc.lock.Unlock()
		if err := os.Rename(contractRoot, path); err != nil {
			return "", errors.Wrap(err, "Fail to rename")

		}
		os.RemoveAll(tmpPath)
	default:
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
		path, err := cm.storeContract(contract.EEType(), codeBuf, codeHash, sc)
		if sc.complete(path, err) == false {
			os.RemoveAll(path)
		}
	}()
	return cs, nil
}

func NewContractManager(db db.Database, chainRoot string, log log.Logger) (ContractManager, error) {
	/*
		contractManager has root path of each service manager's contract file
		So contractManager has to be initialized
		after configurable root path is passed to Service Manager
	*/
	// To manage separate contract store for each chain, add chain ID to
	// parameter here and add it to storeRoot.
	contractDir := path.Join(chainRoot, contractRoot)
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
	tmp := filepath.Join(storeRoot, tmpRoot)
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		os.RemoveAll(tmp)
	}
	return &contractManager{db: db, storeRoot: storeRoot,
			storageCache: make(map[string]*storageCache), log: log},
		nil
}
