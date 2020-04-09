package contract

import (
	"archive/zip"
	"bytes"
	"container/list"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
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
		GetHandler(from, to module.Address, value *big.Int, ctype int, data []byte) ContractHandler
		GetCallHandler(from, to module.Address, value *big.Int, method string, paramObj *codec.TypedObj) ContractHandler
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

func (cm *contractManager) GetHandler(from, to module.Address, value *big.Int, ctype int, data []byte) ContractHandler {
	var handler ContractHandler
	ch := newCommonHandler(from, to, value, cm.log)
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
	value *big.Int, method string, paramObj *codec.TypedObj,
) ContractHandler {
	if value != nil && value.Sign() == 1 { //value > 0
		ch := newCommonHandler(from, to, value, cm.log)
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
			newCommonHandler(from, to, value, cm.log),
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

	err := storeByEEType(eeType, path, code, cm.log)
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
		path, err := cm.storeContract(contract.EEType(), codeBuf, codeHash, sc)
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

const (
	javaCode               = "code.jar"
	tmpRoot                = "tmp"
	contractPythonRootFile = "package.json"
	tryTmpNum              = 10
)

func storePython(path string, code []byte, log log.Logger) error {
	basePath, _ := filepath.Split(path)
	var tmpPath string
	var i int
	for i = 0; i < tryTmpNum; i++ {
		tmpPath = filepath.Join(basePath, tmpRoot, path+strconv.Itoa(i))
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(tmpPath); err != nil {
				break
			}
		} else {
			break
		}
	}
	if i == tryTmpNum {
		return errors.CriticalIOError.Errorf("Fail to create temporary directory")
	}

	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}

	findRoot := false
	scoreRoot := ""
	for _, zipFile := range zipReader.File {
		if info := zipFile.FileInfo(); info.IsDir() {
			continue
		}
		if findRoot == false &&
			filepath.Base(zipFile.Name) == contractPythonRootFile {
			scoreRoot = filepath.Dir(zipFile.Name)
			findRoot = true
		}
		storePath := filepath.Join(tmpPath, zipFile.Name)
		storeDir := filepath.Dir(storePath)
		if _, err := os.Stat(storeDir); os.IsNotExist(err) {
			os.MkdirAll(storeDir, 0755)
		}
		reader, err := zipFile.Open()
		if err != nil {
			return scoreresult.IllegalFormatError.Wrap(err, "Fail to open zip file")
		}
		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			reader.Close()
			return scoreresult.IllegalFormatError.Wrap(err, "Fail to read zip file")
		}
		if err = ioutil.WriteFile(storePath, buf, 0755); err != nil {
			return errors.CriticalIOError.Wrapf(err, "FailToWriteFile(name=%s)", storePath)
		}
		err = reader.Close()
		if err != nil {
			return errors.CriticalIOError.Wrap(err, "Fail to close zip file")
		}
	}
	if findRoot == false {
		os.RemoveAll(tmpPath)
		return scoreresult.IllegalFormatError.Errorf(
			"Root file does not exist(required:%s)\n", contractPythonRootFile)
	}
	contractRoot := filepath.Join(tmpPath, scoreRoot)
	if err := os.Rename(contractRoot, path); err != nil {
		log.Warnf("tmpPath(%s), scoreRoot(%s), err(%s)\n", tmpPath, scoreRoot, err)
		return errors.CriticalIOError.Wrapf(err, "FailToRenameTo(from=%s to=%s)", contractRoot, path)
	}
	if err := os.RemoveAll(tmpPath); err != nil {
		log.Debugf("Failed to remove tmpPath(%s), err(%s)\n", tmpPath, err)
	}
	return nil
}

func storeJava(path string, code []byte, log log.Logger) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0755); err != nil {
			return errors.WithCode(err, errors.CriticalIOError)
		}
	}
	sPath := filepath.Join(path, javaCode)
	if err := ioutil.WriteFile(sPath, code, 0755); err != nil {
		_ = os.RemoveAll(sPath)
		return errors.WithCode(err, errors.CriticalIOError)
	}
	return nil
}

func storeByEEType(e state.EEType, path string, code []byte, log log.Logger) error {
	var err error
	switch e {
	case state.PythonEE:
		err = storePython(path, code, log)
	case state.JavaEE:
		err = storeJava(path, code, log)
	default:
		err = scoreresult.Errorf(module.StatusInvalidParameter,
			"UnexpectedEEType(%v)\n", e)
	}
	return err
}
