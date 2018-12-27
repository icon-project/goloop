package service

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
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
	ctypeDeploy
	ctypeAccept
	ctypeCall
	ctypeGovCall
	ctypeTransferAndMessage = ctypeTransfer | ctypeMessage
	ctypeTransferAndCall    = ctypeTransfer | ctypeCall
	ctypeTransferAndDeploy  = ctypeTransfer | ctypeDeploy
)

type (
	ContractManager interface {
		GetHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler

		PrepareContractStore(ws WorldState, contract Contract,
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
		SendResult(status module.Status, steps *big.Int, result interface{}) error
		Cancel()

		EEType() string
		eeproxy.CallContext
	}
)

// type for store status
type tsStatus int

const (
	tsInProgress tsStatus = iota
	tsComplete
)

type storageCache struct {
	status   tsStatus
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
	case ctypeCall:
		handler = newCallHandler(from, to, value, stepLimit, data, cc)
	case ctypeGovCall:
		handler = &GovCallHandler{
			newCallHandler(from, to, value, stepLimit, data, cc),
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
	case ctypeTransferAndCall:
		handler = &TransferAndCallHandler{
			newCallHandler(from, to, value, stepLimit, data, cc),
		}
	case ctypeTransferAndDeploy:
		handler = newDeployHandler(from, to, value, stepLimit, data, cc, false)
	}
	return handler
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
	ws WorldState, contract Contract, onEndCallback func(string, error)) {
	go func() {
		cm.lock.Lock()
		codeHash := contract.CodeHash()
		hashStr := string(codeHash)
		var path string
		if cacheInfo, ok := cm.storageCache[hashStr]; ok {
			if cacheInfo.status != tsComplete {
				cacheInfo.callback = append(cacheInfo.callback, onEndCallback)
				cm.lock.Unlock()
				return
			}
			path = contractPath(
				fmt.Sprintf("%s/%x", cm.contractRoot, codeHash))
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				onEndCallback(path, nil)
				cm.lock.Unlock()
				return
			}
		}

		cm.storageCache[hashStr] =
			&storageCache{tsInProgress,
				[]func(string, error){onEndCallback}}
		cm.lock.Unlock()

		callEndCb := func(path string, err error) {
			cm.lock.Lock()
			storage := cm.storageCache[hashStr]
			for _, f := range storage.callback {
				f(path, err)
			}
			storage.callback = nil
			storage.status = tsComplete
			cm.lock.Unlock()
		}

		if len(path) == 0 {
			path = contractPath(fmt.Sprintf("%s/%x", cm.contractRoot, codeHash))
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
}

var contractRoot = "./contract/"

func contractPath(codeHash string) string {
	path := contractRoot + codeHash
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
	contract Contract
	started  bool
	path     string
	err      error
	cv       *sync.Cond
}

func newContractStoreProxy() *contractStoreProxy {
	return &contractStoreProxy{cv: sync.NewCond(new(sync.Mutex))}
}

func (p *contractStoreProxy) prepare(wc WorldContext, contract Contract) {
	p.cv.L.Lock()
	if contract != nil && contract.Equal(p.contract) {
		// avoid to call PrepareContractStore() more than once for same contract
		return
	}
	p.cv.L.Unlock()

	if contract == nil {
		p.cv.L.Lock()
		p.err = errors.New("No contract exists")
		p.cv.L.Unlock()
		return
	}

	p.contract = contract
	wc.ContractManager().PrepareContractStore(wc, contract, p.onStoreCompleted)
}

func (p *contractStoreProxy) check(wc WorldContext, contract Contract) (string, error) {
	p.cv.L.Lock()
	defer p.cv.L.Unlock()

	if contract != nil && contract.Equal(p.contract) && (p.err != nil || p.path != "") {
		return p.path, p.err
	}

	p.prepare(wc, contract)
	p.cv.Wait()
	return p.path, p.err
}

func (p *contractStoreProxy) onStoreCompleted(path string, err error) {
	p.cv.L.Lock()
	p.path = path
	p.err = err
	p.cv.Broadcast()
	p.cv.L.Unlock()
}

type dataCallJSON struct {
	method string          `json:"method"`
	params json.RawMessage `json:"params"`
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

// TODO data is not always JSON string, so consider it
func newCallHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext,
) *CallHandler {
	var jso dataCallJSON
	if err := json.Unmarshal(data, &jso); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &CallHandler{
		th:     &TransferHandler{from: from, to: to, value: value, stepLimit: stepLimit},
		method: jso.method,
		params: jso.params,
		cc:     cc,
		csp:    newContractStoreProxy(),
	}
}

func (h *CallHandler) StepLimit() *big.Int {
	return h.th.stepLimit
}

func (h *CallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	h.csp.prepare(wc, h.as.ActiveContract())

	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	// TODO check if contract is active
	h.as = wc.GetAccountState(h.th.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.as.ActiveContract())
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

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result interface{}) {
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

	jso := dataCallJSON{method: method, params: params}
	data, err := json.Marshal(jso)
	if err != nil {
		log.Panicln("Wrong params: FAIL to create data JSON string")
	}
	handler := h.cm.GetHandler(h.cc, from, to, value, limit, ctype, data)
	h.cc.OnCall(handler)
}

func (h *CallHandler) OnAPI(obj interface{}) {
	log.Panicln("Unexpected OnAPI() call from Invoke()")
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
		content:     dataJSON.content,
		contentType: dataJSON.contentType,

		params: dataJSON.params,
	}
}

type DeployHandler struct {
	*TransferHandler
	cc          CallContext
	eeType      string
	content     string
	contentType string
	params      json.RawMessage
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

func (h *DeployHandler) ExecuteSync(wc WorldContext, limit *big.Int) (
	module.Status, *big.Int, []byte, module.Address) {
	sysAs := wc.GetAccountState(SystemID)

	var codeBuf []byte
	var contractID []byte
	if bytes.Equal(h.to.ID(), SystemID) {
		var tsBytes [4]byte
		_ = binary.Write(bytes.NewBuffer(tsBytes[:]), binary.BigEndian, h.timestamp)
		var nBytes [4]byte
		_ = binary.Write(bytes.NewBuffer(nBytes[:]), binary.BigEndian, h.timestamp)
		contractID = GenContractAddr(h.from.ID(), tsBytes[:], nBytes[:])
	} else {
		contractID = h.to.ID()
	}

	var stepUsed *big.Int

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

	// calculate stepUsed and apply it
	codeLen := int64(len(codeBuf))
	stepUsed = new(big.Int)
	stepUsed.SetInt64(codeLen)
	step := big.NewInt(wc.StepsFor(StepTypeContractCreate, 1))
	stepUsed.Mul(stepUsed, step)

	if stepUsed.Cmp(limit) > 0 {
		return module.StatusNotPayable, limit, nil, nil
	}

	ownerAs := wc.GetAccountState(h.from.ID())
	bal := ownerAs.GetBalance()

	if bal.Cmp(stepUsed) < 0 {
		stepUsed.Set(bal)
		ownerAs.SetBalance(big.NewInt(0))
		return module.StatusOutOfBalance, stepUsed, nil, nil
	}
	bal.Sub(bal, stepUsed)
	ownerAs.SetBalance(bal)

	// store ScoreDeployInfo and ScoreDeployTXParams
	as := wc.GetAccountState(contractID)

	as.InitContractAccount(h.from)
	as.DeployContract(codeBuf, h.eeType, h.contentType, h.params, h.txHash)
	sysAs.SetValue(h.txHash, contractID)

	// TODO create AcceptHandler and execute
	return module.StatusSuccess, nil, nil, nil
}

type GovCallHandler struct {
	*CallHandler
}

func (h *GovCallHandler) ExecuteAsync(wc WorldContext) error {
	// skip to check if governance is active
	h.as = wc.GetAccountState(h.th.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.as.NextContract())
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

type AcceptHandler struct {
	from        module.Address
	to          module.Address
	stepLimit   *big.Int
	txHash      []byte
	auditTxHash []byte
	cc          CallContext
}

func newAcceptHandler(from, to module.Address, value, stepLimit *big.Int, data []byte, cc CallContext) *AcceptHandler {
	// TODO parse hash
	hash := make([]byte, 0)
	auditTxHash := make([]byte, 0)
	return &AcceptHandler{from: from, to: to, stepLimit: stepLimit, txHash: hash, auditTxHash: auditTxHash, cc: cc}
}

func (h *AcceptHandler) StepLimit() *big.Int {
	return h.stepLimit
}

// It's never called
func (h *AcceptHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *AcceptHandler) ExecuteSync(wc WorldContext,
) (module.Status, *big.Int, []byte, module.Address) {
	// 1. call GetAPI
	stepAvail := h.stepLimit
	sysAs := wc.GetAccountState(SystemID)
	addr, err := sysAs.GetValue(h.txHash)
	if err != nil || len(addr) == 0 {
		log.Printf("Failed to get score address by txHash\n")
		return module.StatusSystemError, h.stepLimit, nil, nil
	}

	cgah := &callGetAPIHandler{newCallHandler(h.from, h.to, nil, stepAvail, nil, h.cc)}
	status, stepUsed1, _, _ := h.cc.Call(cgah)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}

	// 2. call on_install or on_update of the contract
	stepAvail = stepAvail.Sub(stepAvail, stepUsed1)
	as := wc.GetAccountState(addr)
	var method string
	if bytes.Equal(h.to.ID(), SystemID) {
		method = "on_install"
	} else {
		method = "on_update"
	}
	// TODO check the type of params
	dataJson := map[string]interface{}{
		"method": method, //on_install, on_update
		"params": as.NextContract().Params(),
	}
	data, err := json.Marshal(dataJson)
	if err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
	if err = as.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	handler := wc.ContractManager().GetHandler(h.cc, h.from,
		common.NewContractAddress(addr), nil, stepAvail,
		ctypeCall, data)
	status, stepUsed2, _, _ := h.cc.Call(handler)
	_ = sysAs.DeleteValue(h.txHash)

	return status, stepUsed1.Add(stepUsed1, stepUsed2), nil, nil
}

type callGetAPIHandler struct {
	*CallHandler
}

// It's never called
func (h *callGetAPIHandler) Prepare(wc WorldContext) (WorldContext, error) {
	h.csp.prepare(wc, h.as.NextContract())
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(nil)), nil
}

func (h *callGetAPIHandler) ExecuteAsync(wc WorldContext) error {
	// TODO check which contract it should use, current or next?
	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.as.NextContract())
	if err != nil {
		return err
	}

	err = h.conn.GetAPI(h, path)
	if err != nil {
		return err
	}

	return nil
}

func (h *callGetAPIHandler) GetValue(key []byte) ([]byte, error) {
	return nil, errors.New("Invalid GetValue() call")
}

func (h *callGetAPIHandler) SetValue(key, value []byte) error {
	return errors.New("Invalid SetValue() call")
}

func (h *callGetAPIHandler) DeleteValue(key []byte) error {
	return errors.New("Invalid DeleteValue() call")
}

func (h *callGetAPIHandler) OnResult(status uint16, steps *big.Int, result interface{}) {
	log.Panicln("Unexpected call OnResult() from GetAPI()")
}

func (h *callGetAPIHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params []byte) {
	log.Panicln("Unexpected call OnCall() from GetAPI()")
}

func (h *callGetAPIHandler) OnAPI(obj interface{}) {
	// TODO implement after deciding how to store
	panic("implement me")
}
