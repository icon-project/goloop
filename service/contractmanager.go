package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type (
	tsStatus int

	ContractManager interface {
		GetHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler
		GetCallHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, method string, paramObj *codec.TypedObj) ContractHandler
		PrepareContractStore(ws WorldState,
			contract Contract) <-chan *storageResult
	}

	storageResult struct {
		path string
		err  error
	}

	storageCache struct {
		status tsStatus
		result []chan *storageResult
	}

	contractManager struct {
		lock         sync.Mutex
		db           db.Database
		storageCache map[string]*storageCache
		storeRoot    string
	}
)

const (
	contractStoreRoot          = "./contract"
	tsInProgress      tsStatus = iota
	tsComplete
)

func (cm *contractManager) GetHandler(cc CallContext,
	from, to module.Address, value, stepLimit *big.Int, ctype int, data []byte,
) ContractHandler {
	var handler ContractHandler
	switch ctype {
	case ctypeTransfer:
		handler = newTransferHandler(from, to, value, stepLimit)
	case ctypeCall:
		handler = newCallHandler(newCommonHandler(from, to, value, stepLimit), data, cc, false)
	case ctypeTransferAndMessage:
		handler = &TransferAndMessageHandler{
			TransferHandler: newTransferHandler(from, to, value, stepLimit),
			data:            data,
		}
	case ctypeTransferAndCall:
		th := newTransferHandler(from, to, value, stepLimit)
		handler = &TransferAndCallHandler{
			th:          th,
			CallHandler: newCallHandler(th.CommonHandler, data, cc, false),
		}
	case ctypeTransferAndDeploy:
		handler = newDeployHandler(from, to, value, stepLimit, data, cc, false)
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
			&CommonHandler{from: from, to: to, value: value, stepLimit: stepLimit},
			method, paramObj, cc, false)
	}
}

// if path does not exist, make the path
func (cm *contractManager) storeContract(eeType string, code []byte, codeHash []byte) (string, error) {
	tmpPath := fmt.Sprintf("%s/tmp/%016x", cm.storeRoot, codeHash)
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		if err := os.RemoveAll(tmpPath); err != nil {
			return "", err
		}
	}
	os.MkdirAll(tmpPath, 0755)
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return "", err
	}

	var path string
	switch eeType {
	case "python":
		noRoot := false
		rootDir := ""
		for _, zipFile := range zipReader.File {
			if info := zipFile.FileInfo(); info.IsDir() {
				continue
			}
			if strings.Contains(zipFile.Name, "/") {
				if len(rootDir) == 0 {
					rootDir = strings.Split(zipFile.Name, "/")[0]
				}
			} else if noRoot == false {
				noRoot = true
			}
			log.Printf("zipFile.Name : %s\n", zipFile.Name)
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
		if noRoot == false {
			tmpPath = tmpPath + "/" + rootDir
		}
		path = cm.getContractPath(codeHash)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := os.RemoveAll(path); err != nil {
				return "", err
			}
		}
		if err := os.Rename(tmpPath, path); err != nil {
			return "", err
		}
	default:
	}

	return path, nil
}

func (cm *contractManager) getContractPath(codeHash []byte) string {
	return fmt.Sprintf("%s/%016x", cm.storeRoot, codeHash)
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
// Do not call PrepareContractStore on onEndCallback
func (cm *contractManager) PrepareContractStore(
	ws WorldState, contract Contract) <-chan *storageResult {
	cm.lock.Lock()
	codeHash := contract.CodeHash()
	hashStr := string(codeHash)
	var path string
	sr := make(chan *storageResult, 1)
	if cacheInfo, ok := cm.storageCache[hashStr]; ok {
		if cacheInfo.status != tsComplete {
			cacheInfo.result = append(cacheInfo.result, sr)
			cm.lock.Unlock()
			return sr
		}
		path = cm.getContractPath(codeHash)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			sr <- &storageResult{path, nil}
			cm.lock.Unlock()
			return sr
		}
	}

	cm.storageCache[hashStr] =
		&storageCache{tsInProgress,
			[]chan *storageResult{sr}}
	cm.lock.Unlock()

	go func() {
		callEndCb := func(path string, err error) {
			cm.lock.Lock()
			storage := cm.storageCache[hashStr]
			for _, f := range storage.result {
				f <- &storageResult{path, err}
			}
			storage.result = nil
			storage.status = tsComplete
			cm.lock.Unlock()
		}

		codeBuf, err := contract.Code()
		if err != nil {
			callEndCb("", err)
			return
		}
		path, err = cm.storeContract(contract.EEType(), codeBuf, codeHash)
		if err != nil {
			callEndCb("", err)
			return
		}
		callEndCb(path, nil)
	}()
	return sr
}

func NewContractManager(db db.Database) ContractManager {
	/*
		contractManager has root path of each service manager's contract file
		So contractManager has to be initialized
		after configurable root path is passed to Service Manager
	*/
	// To manage separate contract store for each chain, add chain ID to
	// parameter here and add it to storeRoot.

	// remove tmp to prepare contract
	storeRoot, _ := filepath.Abs(contractStoreRoot)
	tmp := fmt.Sprintf("%s/tmp", storeRoot)
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		if err := os.RemoveAll(tmp); err != nil {
			log.Panicf("Failed to remove %s\n", tmp)
		}
	}
	return &contractManager{db: db, storeRoot: storeRoot,
		storageCache: make(map[string]*storageCache)}
}
