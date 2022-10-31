package contract

import (
	"archive/zip"
	"bytes"
	"container/list"
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/state"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	FileSizeLimit    = 1 * 1024 * 1024
	ContentSizeLimit = 2 * 1024 * 1024
)

type (
	cStatus int

	ContractManager interface {
		DefaultEnabledEETypes() state.EETypes
		GenesisTo() module.Address
		GetHandler(from, to module.Address, value *big.Int, ctype int, data []byte) (ContractHandler, error)
		GetCallHandler(from, to module.Address, value *big.Int, ctype int, paramObj *codec.TypedObj) (ContractHandler, error)
		PrepareContractStore(ws state.WorldState, contract state.ContractState) (ContractStore, error)
		GetSystemScore(contentID string, cc CallContext, from module.Address, value *big.Int) (SystemScore, error)
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
	DataTypeCall    = "call"
	DataTypeMessage = "message"
	DataTypeDeploy  = "deploy"
	DataTypeDeposit = "deposit"
	DataTypePatch   = "patch"
)

func IsCallableDataType(dt *string) bool {
	return dt == nil ||
		*dt == DataTypeCall ||
		*dt == DataTypeMessage
}

func DeployAndInstallSystemSCORE(cc CallContext, contentID string, owner, to module.Address, param []byte, tid []byte) error {
	cm := cc.ContractManager()
	sas := cc.GetAccountState(to.ID())
	sas.InitContractAccount(owner)
	sas.DeployContract([]byte(contentID), state.SystemEE, state.CTAppSystem, nil, tid)
	if err := sas.AcceptContract(tid, tid); err != nil {
		return err
	}
	score, err := cm.GetSystemScore(contentID, cc, owner, new(big.Int))
	if err != nil {
		return err
	}
	if err := score.Install(param); err != nil {
		return err
	}
	apiInfo := score.GetAPI()
	if err := CheckMethod(score, apiInfo); err != nil {
		return err
	}
	sas.MigrateForRevision(cc.Revision())
	sas.SetAPIInfo(apiInfo)
	return nil
}

func (cm *contractManager) ToRevision(value int) module.Revision {
	panic("implement me")
}

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

func (cm *contractManager) GetHandler(from, to module.Address, value *big.Int, ctype int, data []byte) (ContractHandler, error) {
	var handler ContractHandler
	ch := NewCommonHandler(from, to, value, false, cm.log)
	switch ctype {
	case CTypeTransfer:
		if to.IsContract() {
			call := newCallHandlerWithParams(ch, scoreapi.FallbackMethodName, nil, false)
			return newTransferAndCallHandler(ch, call), nil
		} else {
			return newTransferHandler(ch), nil
		}
	case CTypeCall:
		call, err := newCallHandlerWithData(ch, data)
		if err != nil {
			return nil, err
		}
		if value != nil && value.Sign() == 1 {
			return newTransferAndCallHandler(ch, call), nil
		}
		return call, nil
	case CTypeDeploy:
		return newDeployHandler(ch, data)
	case CTypePatch:
		return newPatchHandler(ch, data)
	case CTypeDeposit:
		return newDepositHandler(ch, data)
	}
	return handler, nil
}

func (cm *contractManager) GetCallHandler(
	from, to module.Address,
	value *big.Int,
	ctype int,
	data *codec.TypedObj,
) (ContractHandler, error) {
	ch := NewCommonHandler(from, to, value, true, cm.log)
	switch ctype {
	case CTypeTransfer:
		if to.IsContract() {
			call := newCallHandlerWithParams(ch, scoreapi.FallbackMethodName, nil, false)
			return newTransferAndCallHandler(ch, call), nil
		} else {
			return newTransferHandler(ch), nil
		}
	case CTypeCall:
		call, err := newCallHandlerWithTypedObj(ch, data)
		if err != nil {
			return nil, err
		}
		if value != nil && value.Sign() == 1 {
			return newTransferAndCallHandler(ch, call), nil
		}
		return call, nil
	case CTypeDeploy:
		return newDeployHandlerWithTypedObj(ch, data)
	}
	return nil, errors.NotFoundError.New("UnknownCType")
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
	ws state.WorldState, contract state.ContractState) (ContractStore, error) {
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

func (cm *contractManager) GetSystemScore(contentID string, cc CallContext, from module.Address, value *big.Int) (SystemScore, error) {
	return getSystemScore(contentID, cc, from, value)
}

func (cm *contractManager) DefaultEnabledEETypes() state.EETypes {
	return state.AllEETypes
}

func (cm *contractManager) GenesisTo() module.Address {
	return state.SystemAddress
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
	tmpPattern             = "tmp-*"
	contractPythonRootFile = "package.json"
	tryTmpNum              = 10
)

func storePython(dst string, code []byte, log log.Logger) (ret error) {
	basePath := filepath.Dir(dst)
	tmpPath, err := ioutil.TempDir(basePath, tmpPattern)
	if err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}
	defer func() {
		if ret != nil {
			os.RemoveAll(tmpPath)
		}
	}()

	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}

	var pkg string
	for _, zFile := range zipReader.File {
		info := zFile.FileInfo()
		if info.Name() == contractPythonRootFile && info.IsDir() == false {
			pkg = zFile.Name
			break
		}
	}

	if len(pkg) == 0 {
		return scoreresult.IllegalFormatError.New("NoPackageFile")
	}
	pkgBase, _ := path.Split(pkg)
	var totalSize int64
	for _, zFile := range zipReader.File {
		if zFile.FileInfo().IsDir() {
			continue
		}
		dir, base := path.Split(zFile.Name)
		if !strings.HasPrefix(dir, pkgBase) {
			continue
		}
		dir = strings.TrimPrefix(dir, pkgBase)
		if strings.Contains(dir, "__MACOSX") ||
			strings.Contains(dir, "__pycache__") {
			continue
		}
		sz := zFile.FileInfo().Size()
		if sz > FileSizeLimit {
			return scoreresult.IllegalFormatError.Errorf("OversizeFile(file=%s,size=%d,limit=%d)",
				zFile.Name, sz, FileSizeLimit)
		}
		totalSize += sz
		if totalSize > ContentSizeLimit {
			return scoreresult.IllegalFormatError.Errorf("OversizeContent(size=%d,limit=%d)",
				totalSize, ContentSizeLimit)
		}
		in, err := zFile.Open()
		if err != nil {
			return scoreresult.IllegalFormatError.Errorf("FailToOpen(f=%s)", zFile.Name)
		}
		tmpDir := filepath.Join(tmpPath, filepath.FromSlash(dir))
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return errors.WithCode(err, errors.CriticalIOError)
		}
		tmpFile := filepath.Join(tmpDir, base)
		out, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
		if err != nil {
			return errors.WithCode(err, errors.CriticalIOError)
		}
		if _, err := io.CopyN(out, in, sz); err != nil {
			return errors.WithCode(err, errors.CriticalIOError)
		}
	}

	return os.Rename(tmpPath, dst)
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
