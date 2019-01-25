package service

import (
	"archive/zip"
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type (
	cStatus int

	ContractManager interface {
		GetHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler
		GetCallHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, method string, paramObj *codec.TypedObj) ContractHandler
		PrepareContractStore(ws WorldState, contract Contract) (ContractStore, error)
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
	}
)

const (
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

func (cm *contractManager) GetHandler(cc CallContext,
	from, to module.Address, value, stepLimit *big.Int, ctype int, data []byte,
) ContractHandler {
	var handler ContractHandler
	switch ctype {
	case ctypeTransfer:
		handler = newTransferHandler(from, to, value, stepLimit)
	case ctypeCall:
		handler = newCallHandler(newCommonHandler(from, to, value, stepLimit), data, cc, false)
	case ctypeDeploy:
		handler = newDeployHandler(from, to, value, stepLimit, data, cc, false)
	case ctypeTransferAndCall:
		th := newTransferHandler(from, to, value, stepLimit)
		handler = &TransferAndCallHandler{
			th:          th,
			CallHandler: newCallHandler(th.CommonHandler, data, cc, false),
		}
	}
	return handler
}

func (cm *contractManager) GetCallHandler(cc CallContext, from, to module.Address,
	value, stepLimit *big.Int, method string, paramObj *codec.TypedObj,
) ContractHandler {
	if value != nil && value.Sign() == 1 { //value > 0
		th := newTransferHandler(from, to, value, stepLimit)
		return &TransferAndCallHandler{
			th:          th,
			CallHandler: newCallHandlerFromTypedObj(th.CommonHandler, method, paramObj, cc, false),
		}
	} else {
		return newCallHandlerFromTypedObj(
			newCommonHandler(from, to, value, stepLimit),
			method, paramObj, cc, false)
	}
}

// if path does not exist, make the path
func (cm *contractManager) storeContract(eeType string, code []byte, codeHash []byte, sc *storageCache) (string, error) {
	var path string
	path = fmt.Sprintf("%s/%016x", cm.storeRoot, codeHash)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}

	var tmpPath string
	var i int
	for i = 0; i < 10; i++ {
		tmpPath = fmt.Sprintf("%s/tmp/%016x%d", cm.storeRoot, codeHash, i)
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(tmpPath); err != nil {
				continue
			}
		} else {
			break
		}
	}
	if i == 10 {
		return "", errors.New("Failed to create temporary directory\n")
	}
	os.MkdirAll(tmpPath, 0755)
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return "", err
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
				strings.HasSuffix(zipFile.Name, contractPythonRootFile) {
				rootDir = strings.TrimSuffix(zipFile.Name, contractPythonRootFile)
				findRoot = true
			}
			storePath := tmpPath + "/" + zipFile.Name
			storeDir := filepath.Dir(storePath)
			if _, err := os.Stat(storeDir); os.IsNotExist(err) {
				os.MkdirAll(storeDir, 0755)
			}
			reader, err := zipFile.Open()
			if err != nil {
				return "", errors.New("Failed to open zip file\n")
			}
			buf, err := ioutil.ReadAll(reader)
			if err != nil {
				return "", errors.New("Failed to read zip file\n")
			}
			if err = ioutil.WriteFile(storePath, buf, os.ModePerm); err != nil {
				log.Printf("Failed to write file. err = %s\n", err)
			}
		}
		contractRoot := tmpPath + "/" + rootDir

		sc.timer.Stop()
		sc.lock.Lock()
		if sc.status == csComplete {
			sc.lock.Unlock()
			os.Remove(contractRoot)
			return "", nil
		}
		sc.lock.Unlock()
		if err := os.Rename(contractRoot, path); err != nil {
			return "", err
		}
		os.RemoveAll(tmpPath)
	default:
	}

	return path, nil
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
func (cm *contractManager) PrepareContractStore(
	ws WorldState, contract Contract) (ContractStore, error) {
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
			if sc.complete("", errors.New("Expired decompress time\n")) == true {
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

func NewContractManager(db db.Database, contractDir string) ContractManager {
	/*
		contractManager has root path of each service manager's contract file
		So contractManager has to be initialized
		after configurable root path is passed to Service Manager
	*/
	// To manage separate contract store for each chain, add chain ID to
	// parameter here and add it to storeRoot.

	// remove tmp to prepare contract
	storeRoot, _ := filepath.Abs(contractDir)
	tmp := fmt.Sprintf("%s/tmp", storeRoot)
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		if err := os.RemoveAll(tmp); err != nil {
			log.Panicf("Failed to remove %s\n", tmp)
		}
	}
	return &contractManager{db: db, storeRoot: storeRoot,
		storageCache: make(map[string]*storageCache)}
}
