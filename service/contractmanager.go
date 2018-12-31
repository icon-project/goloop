package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"sync"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

// type for store status
type tsStatus int

const (
	tsInProgress tsStatus = iota
	tsComplete
)

type storageResult struct {
	path string
	err  error
}
type storageCache struct {
	status tsStatus
	//callback []func(string, error)
	callback []chan *storageResult
}

type contractManager struct {
	lock         sync.Mutex
	db           db.Database
	storageCache map[string]*storageCache
	storeRoot    string
}

const (
	contractStoreRoot = "./contract/"
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
	case ctypeGovCall:
		handler = &GovCallHandler{
			newCallHandler(newCommonHandler(from, to, value, stepLimit), data, cc, false),
		}
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
func storeContract(code []byte, path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	os.MkdirAll(path, os.ModePerm)
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return err
	}

	for _, zipFile := range zipReader.File {
		storePath := path + "/" + zipFile.Name
		if info := zipFile.FileInfo(); info.IsDir() {
			os.MkdirAll(path+"/"+info.Name(), os.ModePerm)
			continue
		}
		reader, err := zipFile.Open()
		if err != nil {
			return errors.New("Failed to open zip file\n")
		}
		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			return errors.New("Failed to read zip file\n")
		}
		if err = ioutil.WriteFile(storePath, buf, os.ModePerm); err != nil {
			log.Printf("Failed to write file. err = %s\n", err)
		}
	}
	return nil
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
			cacheInfo.callback = append(cacheInfo.callback, sr)
			cm.lock.Unlock()
			return sr
		}
		path = fmt.Sprintf("%s/%x", cm.storeRoot, codeHash)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			sr <- &storageResult{path, nil}
			cm.lock.Unlock()
			return sr
		}
	}

	cm.storageCache[hashStr] =
		&storageCache{tsInProgress,
			[]chan *storageResult{}}
	cm.lock.Unlock()

	go func() {
		callEndCb := func(path string, err error) {
			cm.lock.Lock()
			storage := cm.storageCache[hashStr]
			for _, f := range storage.callback {
				f <- &storageResult{path, err}
			}
			storage.callback = nil
			storage.status = tsComplete
			cm.lock.Unlock()
		}

		if len(path) == 0 {
			path = fmt.Sprintf("%s/%x", cm.storeRoot, codeHash)
		}
		codeBuf, err := contract.Code()
		if err != nil {
			callEndCb("", err)
			return
		}
		err = storeContract(codeBuf, path)
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
	return &contractManager{db: db, storeRoot: contractStoreRoot}
}
