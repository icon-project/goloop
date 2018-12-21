package service

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/db"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/pkg/errors"
)

func NewContractManager(db db.Database) ContractManager {
	/*
		contractManager has root path of each service manager's contract file
		So contractManager has to be initialized
		after configurable root path is passed to Service Manager
	*/
	return &contractManager{db: db}
}

const (
	transactionTimeLimit = time.Duration(2 * time.Second)

	ctypeTransfer = 0x100
	ctypeNone     = iota
	ctypeMessage
	ctypeCall
	ctypeDeploy
	ctypeTransferAndMessage = ctypeTransfer | ctypeMessage
	ctypeTransferAndCall    = ctypeTransfer | ctypeCall
	ctypeTransferAndDeploy  = ctypeTransfer | ctypeDeploy
)

type (
	ContractManager interface {
		GetHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler
		PrepareContractStore(ws WorldState, addr module.Address,
			onEndCallback func(path string, err error))
	}

	ContractHandler interface {
		StepLimit() *big.Int
		Prepare(wc WorldContext) (WorldContext, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (module.Status, *big.Int, []byte, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) error
		SendResult(status module.Status, steps *big.Int, result []byte) error
		Cancel()

		EEType() string
		eeproxy.CallContext
	}
)

type typeStore int

const (
	storeProgress typeStore = iota
	storeComplete
)

type storageCache struct {
	status   typeStore
	callback []func(string, error)
}

type contractManager struct {
	lock         sync.Mutex
	db           db.Database
	storageCache map[string]*storageCache
	contractRoot string
}

func (cm *contractManager) GetHandler(cc CallContext,
	from, to module.Address, value, stepLimit *big.Int, ctype int, data []byte,
) ContractHandler {
	var handler ContractHandler
	switch ctype {
	case ctypeTransfer:
		handler = &TransferHandler{
			from:      from,
			to:        to,
			value:     value,
			stepLimit: stepLimit,
		}
	case ctypeTransferAndMessage:
		handler = &TransferAndMessageHandler{
			TransferHandler: &TransferHandler{
				from:      from,
				to:        to,
				value:     value,
				stepLimit: stepLimit,
			},
			data: data,
		}
	case ctypeTransferAndDeploy:
		handler = newDeployHandler(from, to, value, stepLimit, data, cc, false)
	case ctypeTransferAndCall:
		handler = &TransferAndCallHandler{
			newCallHandler(from, to, value, stepLimit, data, cc),
		}
	case ctypeCall:
		handler = newCallHandler(from, to, value, stepLimit, data, cc)
	}
	return handler
}

// storeContract don't check if path exists or not
// path existence has to be checked before storeContract is called
func storeContract(path string, contractCode []byte) error {
	zipReader, err :=
		zip.NewReader(bytes.NewReader(contractCode), int64(len(contractCode)))
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

func prepareContract(compressedCode []byte, path string, removeIfExist bool) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if removeIfExist == false {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	os.MkdirAll(path, os.ModePerm)
	err := storeContract(path, compressedCode)

	return err
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
// Do not call PrepareContractStore on onEndCallback
func (cm *contractManager) PrepareContractStore(
	ws WorldState, addr module.Address, onEndCallback func(string, error)) {
	go func() {
		contractId := addr.ID()
		cm.lock.Lock()
		if cacheInfo, ok := cm.storageCache[string(contractId)]; ok {
			if cacheInfo.status != storeComplete {
				cacheInfo.callback = append(cacheInfo.callback, onEndCallback)
				cm.lock.Unlock()
				return
			}
			path := contractPath(
				fmt.Sprintf("%s/%x", cm.contractRoot, contractId))
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				onEndCallback(path, nil)
				cm.lock.Unlock()
				return
			}
		}

		cm.storageCache[string(contractId)] =
			&storageCache{storeProgress,
				[]func(string, error){onEndCallback}}
		cm.lock.Unlock()

		callEndCb := func(path string, err error) {
			cm.lock.Lock()
			storage := cm.storageCache[string(contractId)]
			for _, f := range storage.callback {
				f(path, err)
			}
			storage.callback = nil
			storage.status = storeComplete
			cm.lock.Unlock()
		}

		as := ws.GetAccountState(contractId)
		contract := as.GetCurContract()
		if contract == nil {
			callEndCb("",
				errors.New("Failed to get current contract info"))
			return
		}
		codeHash := contract.GetCodeHash()
		bk, err := cm.db.GetBucket(db.BytesByHash)
		if err != nil {
			callEndCb("", err)
			return
		}
		code, err := bk.Get(codeHash)
		if err != nil {
			callEndCb("", err)
			return
		}
		path := contractPath(fmt.Sprintf("%s/%x", cm.contractRoot, addr.ID()))
		err = prepareContract(code, path, false)
		if err != nil {
			callEndCb("", err)
			return
		}
		callEndCb(path, nil)
	}()
}

// TODO Where is the root directory of contract
// TODO How to generate contract path from codeHash
func contractPath(codeHash string) string {
	path := "./contract/" + codeHash
	return path
}

func executeTransfer(wc WorldContext, from, to module.Address,
	value, limit *big.Int,
) (module.Status, *big.Int) {
	stepUsed := big.NewInt(wc.StepsFor(StepTypeDefault, 1))

	if stepUsed.Cmp(limit) > 0 {
		return module.StatusNotPayable, limit
	}

	as1 := wc.GetAccountState(from.ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(value) < 0 {
		return module.StatusOutOfBalance, limit
	}
	bal1.Sub(bal1, value)
	as1.SetBalance(bal1)

	as2 := wc.GetAccountState(to.ID())
	bal2 := as2.GetBalance()
	bal2.Add(bal2, value)
	as2.SetBalance(bal2)

	return module.StatusSuccess, stepUsed
}

type TransferHandler struct {
	from, to         module.Address
	value, stepLimit *big.Int
}

func (h *TransferHandler) StepLimit() *big.Int {
	return h.stepLimit
}

func (h *TransferHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{string(h.from.ID()), AccountWriteLock},
		{string(h.to.ID()), AccountWriteLock},
	}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *TransferHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, []byte, module.Address) {
	stepPrice := wc.StepPrice()
	var (
		fee                 big.Int
		status              module.Status
		step, bal1          *big.Int
		stepUsed, stepAvail big.Int
	)
	wcs := wc.GetSnapshot()
	as1 := wc.GetAccountState(h.from.ID())
	stepAvail.Set(h.stepLimit)

	// it tries to execute
	status, step = executeTransfer(wc, h.from, h.to, h.value, &stepAvail)
	stepUsed.Set(step)
	stepAvail.Sub(&stepAvail, step)

	// try to charge fee
	fee.Mul(&stepUsed, stepPrice)
	bal1 = as1.GetBalance()
	for bal1.Cmp(&fee) < 0 {
		if status == 0 {
			// rollback all changes
			status = module.StatusNotPayable
			wc.Reset(wcs)
			bal1 = as1.GetBalance()

			stepUsed.Set(h.stepLimit)
			fee.Mul(&stepUsed, stepPrice)
		} else {
			//stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal1.Sub(bal1, &fee)
	as1.SetBalance(bal1)

	return status, &stepUsed, nil, nil
}

type TransferAndMessageHandler struct {
	*TransferHandler
	data []byte
}

func (h *TransferAndMessageHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, []byte, module.Address) {
	stepPrice := wc.StepPrice()
	var (
		fee                 big.Int
		status              module.Status
		step, bal1          *big.Int
		stepUsed, stepAvail big.Int
	)
	wcs := wc.GetSnapshot()
	as1 := wc.GetAccountState(h.from.ID())
	stepAvail.Set(h.stepLimit)

	// it tries to execute
	status, step = executeTransfer(wc, h.from, h.to, h.value, &stepAvail)
	stepUsed.Set(step)
	stepAvail.Sub(&stepAvail, step)

	if status == 0 {
		var data interface{}
		if err := json.Unmarshal(h.data, &data); err != nil {
			status = module.StatusSystemError
			step = &stepAvail
		} else {
			var stepsForMessage big.Int
			stepsForMessage.SetInt64(wc.StepsFor(StepTypeInput, h.countBytesOfData(data)))
			if stepAvail.Cmp(&stepsForMessage) < 0 {
				status = module.StatusNotPayable
				step = &stepAvail
			} else {
				step = &stepsForMessage
			}
		}
		stepUsed.Add(&stepUsed, step)
		stepAvail.Sub(&stepAvail, step)
	}

	// try to charge fee
	fee.Mul(&stepUsed, stepPrice)
	bal1 = as1.GetBalance()
	for bal1.Cmp(&fee) < 0 {
		if status == 0 {
			// rollback all changes
			status = module.StatusNotPayable
			wc.Reset(wcs)
			bal1 = as1.GetBalance()

			stepUsed.Set(h.stepLimit)
			fee.Mul(&stepUsed, stepPrice)
		} else {
			//stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal1.Sub(bal1, &fee)
	as1.SetBalance(bal1)

	return status, &stepUsed, nil, nil
}

func (h *TransferAndMessageHandler) countBytesOfData(data interface{}) int {
	switch o := data.(type) {
	case string:
		if len(o) > 2 && o[:2] == "0x" {
			o = o[2:]
		}
		bs := []byte(o)
		for _, b := range bs {
			if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
				return len(bs)
			}
		}
		return (len(bs) + 1) / 2
	case []interface{}:
		var count int
		for _, i := range o {
			count += h.countBytesOfData(i)
		}
		return count
	case map[string]interface{}:
		var count int
		for _, i := range o {
			count += h.countBytesOfData(i)
		}
		return count
	case bool:
		return 1
	case float64:
		return len(common.Int64ToBytes(int64(o)))
	default:
		return 0
	}
}

type contractStoreProxy struct {
	started bool
	path    string
	err     error
	cv      *sync.Cond
}

func newContractStoreProxy() *contractStoreProxy {
	return &contractStoreProxy{cv: sync.NewCond(new(sync.Mutex))}
}

func (p *contractStoreProxy) prepare(wc WorldContext, addr module.Address) {
	p.cv.L.Lock()
	if p.started {
		// avoid to call PrepareContractStore() more than once
		return
	}
	p.started = true
	p.cv.L.Unlock()
	wc.ContractManager().PrepareContractStore(wc, addr, p.onContractStoreCompleted)
}

func (p *contractStoreProxy) check(wc WorldContext, addr module.Address) (string, error) {
	p.cv.L.Lock()
	defer p.cv.L.Unlock()

	if p.err != nil || p.path != "" {
		return p.path, p.err
	}

	p.prepare(wc, addr)
	p.cv.Wait()
	return p.path, p.err
}

func (p *contractStoreProxy) onContractStoreCompleted(path string, err error) {
	p.cv.L.Lock()
	p.path = path
	p.err = err
	p.cv.Broadcast()
	p.cv.L.Unlock()
}

type CallHandler struct {
	// Don't embed TransferHandler because it should not be an instance of
	// SyncContractHandler.
	th *TransferHandler

	method string
	params []byte

	cc  CallContext
	csp *contractStoreProxy

	// set in ExecuteAsync()
	as   AccountState
	cm   ContractManager
	conn eeproxy.Proxy
}

func newCallHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext,
) *CallHandler {
	var dataJSON struct {
		method string          `json:"method"`
		params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &dataJSON); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &CallHandler{
		th:     &TransferHandler{from: from, to: to, value: value, stepLimit: stepLimit},
		method: dataJSON.method,
		params: dataJSON.params,
		cc:     cc,
		csp:    newContractStoreProxy(),
	}
}

func (h *CallHandler) StepLimit() *big.Int {
	return h.th.stepLimit
}

func (h *CallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	h.csp.prepare(wc, h.th.to)

	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.th.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.th.to)
	if err != nil {
		return err
	}

	err = h.conn.Invoke(h, path, false, h.th.from, h.th.to, h.th.value,
		h.th.stepLimit, h.method, h.params)
	if err != nil {
		return err
	}

	return nil
}

func (h *CallHandler) SendResult(status module.Status, steps *big.Int, result []byte) error {
	if h.conn == nil {
		return errors.New("Don't have a connection of (" + h.EEType() + ")")
	}
	return h.conn.SendResult(h, uint16(status), steps, result)
}

func (h *CallHandler) Cancel() {
	// Do nothing
}

func (h *CallHandler) EEType() string {
	// TODO resolve it at run time
	return "python"
}

func (h *CallHandler) GetValue(key []byte) ([]byte, error) {
	if h.as != nil {
		return h.as.GetValue(key)
	} else {
		return nil, errors.New("GetValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) SetValue(key, value []byte) error {
	if h.as != nil {
		return h.as.SetValue(key, value)
	} else {
		return errors.New("SetValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) DeleteValue(key []byte) error {
	if h.as != nil {
		return h.as.DeleteValue(key)
	} else {
		return errors.New("DeleteValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) GetInfo() map[string]interface{} {
	return h.cc.GetInfo()
}

func (h *CallHandler) GetBalance(addr module.Address) *big.Int {
	return h.cc.GetBalance(addr)
}

func (h *CallHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	h.cc.OnEvent(indexed, data)
}

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result []byte) {
	h.cc.OnResult(module.Status(status), steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params []byte,
) {
	ctype := ctypeNone
	if method != "" {
		ctype |= ctypeCall
	}
	if value.Sign() == 1 { // value >= 0
		ctype |= ctypeTransfer
	}
	if ctype == ctypeNone {
		log.Println("Invalid call:", from, to, value, method)

		if conn := h.cc.GetConnection(h.EEType()); conn != nil {
			conn.SendResult(h, uint16(module.StatusSystemError), h.th.stepLimit, nil)
		} else {
			// It can't be happened
			log.Println("FAIL to get connection of (", h.EEType(), ")")
		}
		return
	}

	// TODO make data from method and params
	var data []byte
	handler := h.cm.GetHandler(h.cc, from, to, value, limit, ctype, data)
	h.cc.OnCall(handler)
}

func (h *CallHandler) OnAPI(obj interface{}) {
	// TODO
	panic("implement me")
}

type TransferAndCallHandler struct {
	*CallHandler
}

func (h *TransferAndCallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	if wc, err := h.th.Prepare(wc); err == nil {
		return h.CallHandler.Prepare(wc)
	} else {
		return wc, err
	}
}

func (h *TransferAndCallHandler) ExecuteAsync(wc WorldContext) error {
	if status, stepUsed, result, addr := h.th.ExecuteSync(wc); status == 0 {
		return h.CallHandler.ExecuteAsync(wc)
	} else {
		go func() {
			h.cc.OnResult(module.Status(status), stepUsed, result, addr)
		}()

		return nil
	}
}

func newDeployHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext, force bool,
) *DeployHandler {
	var dataJSON struct {
		contentType string          `json:"contentType""`
		content     string          `json:"content"`
		params      json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &dataJSON); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	// TODO set db
	return &DeployHandler{
		TransferHandler: &TransferHandler{from: from,
			to: to, value: value, stepLimit: stepLimit},
		cc:          cc,
		csp:         newContractStoreProxy(),
		content:     dataJSON.content,
		contentType: dataJSON.contentType,

		params: dataJSON.params,
	}
}

type deployCmdType int

const (
	deployCmdDeploy deployCmdType = iota
	deployCmdAccept
	deployCmdReject
)

type DeployHandler struct {
	*TransferHandler
	cc          CallContext
	csp         *contractStoreProxy
	db          db.Database
	eeType      string
	content     string
	contentType string
	params      json.RawMessage
	data        []byte
	cmdType     deployCmdType
	txHash      []byte

	timestamp int
	nonce     int
}

// nonce, timestamp, from
// data = from(20 bytes) + timestamp (32 bytes) + if exists, nonce (32 bytes)
// digest = sha3_256(data)
// contract address = digest[len(digest) - 20:] // get last 20bytes
func GenContractAddr(from, timestamp, nonce []byte) []byte {
	data := make([]byte, 0, 84)
	data = append([]byte(nil), from...)
	alignLen := 32 // 32 bytes alignment
	tBytes := make([]byte, alignLen-len(timestamp), alignLen)
	tBytes = append(tBytes, timestamp...)
	data = append(data, tBytes...)
	if len(nonce) != 0 {
		nBytes := make([]byte, alignLen-len(nonce), alignLen)
		nBytes = append(nBytes, nonce...)
		data = append(data, nBytes...)
	}
	digest := sha3.Sum256(data)
	addr := make([]byte, 20)
	copy(addr, digest[len(digest)-20:])
	return addr
}

func (h *DeployHandler) ExecuteSync(wc WorldContext) (
	module.Status, *big.Int, []byte, module.Address) {
	const (
		deployInstall = iota
		deployUpdate

		scoreSystemAddr     = "cx0000000000000000000000000000000000000000"
		governanceScoreAddr = "cx0000000000000000000000000000000000000001"
	)
	var sysAddr common.Address
	var governanceAddr common.Address
	// TODO Address for system and governance have to be declared as global variable
	sysAddr.SetString(scoreSystemAddr)
	governanceAddr.SetString(governanceScoreAddr)

	var codeBuf []byte
	var contractAddr *common.Address
	deployType := deployInstall
	if strings.Compare(h.to.String(), scoreSystemAddr) != 0 {
		deployType = deployUpdate
		contractAddr = common.NewAccountAddress(h.from.ID())
	}

	if h.cmdType == deployCmdDeploy {
		// check if audit or from is deployer
		force := false

		// calculate fee
		hexContent := strings.TrimPrefix(h.content, "0x")
		if len(hexContent)%2 != 0 {
			hexContent = "0" + hexContent
		}
		var err error
		codeBuf, err = hex.DecodeString(hexContent)
		if err != nil {
			log.Printf("Failed to")
			return module.StatusSystemError, nil, nil, nil
		}
		// store codeHash
		bk, err := h.db.GetBucket(db.BytesByHash)
		codeHash := sha3.Sum256(codeBuf)
		v, err := bk.Get(codeHash[:])
		if err != nil || v != nil {
			log.Printf("err : %s, v = %x\n", err, v)
			return module.StatusSystemError, nil, nil, nil
		}
		if err = bk.Set(codeHash[:], codeBuf); err != nil {
			log.Printf("failed to set code. err : %s\n", err)
			return module.StatusSystemError, nil, nil, nil
		}

		// calculate stepUsed and apply it
		codeLen := int64(len(codeBuf))
		stepUsed := new(big.Int)
		stepUsed.Mul(wc.StepPrice(), big.NewInt(codeLen))

		ownerAs := wc.GetAccountState(h.from.ID())
		bal := ownerAs.GetBalance()

		if bal.Cmp(stepUsed) < 0 {
			stepUsed.Set(bal)
			ownerAs.SetBalance(big.NewInt(0))
			return module.StatusOutOfBalance, stepUsed, nil, nil
		}
		bal.Sub(bal, stepUsed)
		ownerAs.SetBalance(bal)

		if deployType == deployInstall {
			var bTimestamp []byte
			var bNonce []byte
			contractAddr = common.NewAccountAddress(
				GenContractAddr(h.from.ID(), bTimestamp, bNonce))
		}

		// store ScoreDeployInfo and ScoreDeployTXParams
		as := wc.GetAccountState(contractAddr.ID())
		contract := NewContract()
		defer as.SetNextContract(contract)

		codeHash = sha3.Sum256(codeBuf)
		contract.SetCodeHash(codeHash[:])
		contract.SetDeployTx(h.txHash)
		// TODO check when apiInfo is invoked // ???
		contract.SetParams(h.params)

		cType := cTypeAppZip
		switch h.contentType {
		case "application/zip":
			cType = cTypeAppZip
		default:
			log.Printf("WrongType. %s\n", h.contentType)
		}
		contract.SetContentType(cType)

		if force == false {
			if deployType == deployUpdate {
				contract.SetStatus(csPending)
			} else {
				contract.SetStatus(csInactive)
			}
			systemAs := wc.GetAccountState(sysAddr.ID())
			systemAs.SetValue(h.txHash, contractAddr.ID())

			return module.StatusSuccess, stepUsed, nil, nil
		}
	} else if h.cmdType == deployCmdAccept {
		as := wc.GetAccountState(sysAddr.ID())
		cAddr, err := as.GetValue(h.txHash) // get contract address by txHash
		if err != nil {
			log.Printf("Failed to get value. err : %s\n", err)
		}
		contractAddr = common.NewAccountAddress(cAddr)
		contractAs := wc.GetAccountState(contractAddr.ID())
		nc := contractAs.GetNextContract()
		codeHash := nc.GetCodeHash()
		// store codeHash
		bk, err := h.db.GetBucket(db.BytesByHash)
		codeBuf, err = bk.Get(codeHash[:])
		if err != nil || len(codeBuf) == 0 {
			log.Printf("failed to get code. err : %s\n", err)
			return module.StatusSystemError, nil, nil, nil
		}
	} else if h.cmdType == deployCmdReject {
	}

	path := contractPath(fmt.Sprintf("%x", contractAddr.ID()))
	if err := prepareContract(codeBuf,
		path, deployType == deployUpdate); err != nil {
		log.Printf("failed to prepare contract. err : %s\n", err)
	}

	// statue -> active if failed to on_install, set inactive
	// on_install or on_update
	as := wc.GetAccountState(contractAddr.ID())
	contract := as.GetNextContract()
	contract.SetStatus(csActive)
	handler := wc.ContractManager().GetHandler(h.cc, h.from,
		contractAddr, nil, nil, ctypeCall, h.data)
	h.cc.Call(handler)
	// TODO receive result

	// GET API
	handler = wc.ContractManager().GetHandler(h.cc, h.from,
		contractAddr, nil, nil, ctypeCall, h.data)
	h.cc.Call(handler)
	// TODO receive result

	return module.StatusSuccess, nil, nil, nil
}
