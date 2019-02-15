package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

type DeployHandler struct {
	*CommonHandler
	cc          CallContext
	eeType      string
	content     []byte
	contentType string
	params      []byte
	txHash      []byte
}

func newDeployHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext, force bool,
) *DeployHandler {
	var dataJSON struct {
		ContentType string          `json:"contentType""`
		Content     common.HexBytes `json:"content"`
		Params      json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &dataJSON); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &DeployHandler{
		CommonHandler: newCommonHandler(from, to, value, stepLimit),
		cc:            cc,
		content:       dataJSON.Content,
		contentType:   dataJSON.ContentType,
		// eeType is currently only python
		// but it should be checked later by json element
		eeType: "python",
		params: dataJSON.Params,
	}
}

// nonce, timestamp, from
// data = from(20 bytes) + timestamp (32 bytes) + if exists, nonce (32 bytes)
// digest = sha3_256(data)
// contract address = digest[len(digest) - 20:] // get last 20bytes
func genContractAddr(from module.Address, timestamp int64, nonce *big.Int) []byte {
	tsBytes := bytes.NewBuffer(nil)
	_ = binary.Write(tsBytes, binary.BigEndian, timestamp)
	data := make([]byte, 0, 84)
	data = append([]byte(nil), from.ID()...)
	alignLen := 32 // 32 bytes alignment
	tBytes := make([]byte, alignLen-tsBytes.Len(), alignLen)
	tBytes = append(tBytes, tsBytes.Bytes()...)
	data = append(data, tBytes...)
	if nonce != nil && nonce.Sign() != 0 {
		noBytes := bytes.NewBuffer(nil)
		_ = binary.Write(noBytes, binary.BigEndian, nonce.Bytes())
		nBytes := make([]byte, alignLen-noBytes.Len(), alignLen)
		nBytes = append(nBytes, noBytes.Bytes()...)
		data = append(data, nBytes...)
	}
	digest := sha3.Sum256(data)
	addr := make([]byte, 20)
	copy(addr, digest[len(digest)-20:])
	return addr
}

func (h *DeployHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{"", state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (h *DeployHandler) ExecuteSync(ctx Context) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	sysAs := ctx.GetAccountState(state.SystemID)

	update := false
	var contractID []byte
	if bytes.Equal(h.to.ID(), state.SystemID) { // install
		info := h.cc.GetInfo()
		if info == nil {
			msg, _ := common.EncodeAny("no GetInfo()")
			return module.StatusSystemError, h.stepLimit, msg, nil
		}
		contractID = genContractAddr(h.from, info[state.InfoTxTimestamp].(int64), info[state.InfoTxNonce].(*big.Int))
	} else { // deploy for update
		contractID = h.to.ID()
		update = true
	}

	// calculate stepUsed and apply it
	st := state.StepType(state.StepTypeContractCreate)
	if update {
		st = state.StepTypeContractUpdate
	}
	codeLen := len(h.content)
	if !h.ApplySteps(ctx, st, 1) ||
		!h.ApplySteps(ctx, state.StepTypeContractSet, codeLen) {
		msg, _ := common.EncodeAny("Not enough step limit")
		return module.StatusOutOfStep, h.stepLimit, msg, nil
	}

	// store ScoreDeployInfo and ScoreDeployTXParams
	as := ctx.GetAccountState(contractID)
	if update == false {
		if as.InitContractAccount(h.from) == false {
			msg, _ := common.EncodeAny("Already deployed contract")
			return module.StatusSystemError, h.stepUsed, msg, nil
		}
	} else {
		if as.IsContract() == false {
			msg, _ := common.EncodeAny("Not a contract account")
			return module.StatusContractNotFound, h.stepUsed, msg, nil
		}
		if as.IsContractOwner(h.from) == false {
			msg, _ := common.EncodeAny("Not a contract owner")
			return module.StatusAccessDenied, h.stepUsed, msg, nil
		}
	}
	scoreAddr := common.NewContractAddress(contractID)
	as.DeployContract(h.content, h.eeType, h.contentType, h.params, h.txHash)
	scoreDb := scoredb.NewVarDB(sysAs, h.txHash)
	_ = scoreDb.Set(scoreAddr)

	//if audit == false || deployer {
	ah := newAcceptHandler(h.from, h.to, //common.NewContractAddress(contractID),
		nil, h.StepAvail(), h.params, h.cc)
	status, acceptStepUsed, result, _ := ah.ExecuteSync(ctx)
	h.stepUsed.Add(h.stepUsed, acceptStepUsed)
	if status != module.StatusSuccess {
		return status, h.stepUsed, result, nil
	}
	//}

	return module.StatusSuccess, h.stepUsed, nil, scoreAddr
}

type AcceptHandler struct {
	*CommonHandler
	txHash      []byte
	auditTxHash []byte
	cc          CallContext
}

func newAcceptHandler(from, to module.Address, value, stepLimit *big.Int, data []byte, cc CallContext) *AcceptHandler {
	// TODO parse hash
	hash := make([]byte, 0)
	auditTxHash := make([]byte, 0)
	return &AcceptHandler{
		CommonHandler: newCommonHandler(from, to, value, stepLimit),
		txHash:        hash, auditTxHash: auditTxHash, cc: cc}
}

// It's never called
func (h *AcceptHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{{"", state.AccountWriteLock}}
	return ctx.GetFuture(lq), nil
}

const (
	deployInstall = "on_install"
	deployUpdate  = "on_update"
)

func (h *AcceptHandler) ExecuteSync(ctx Context) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	// 1. call GetAPI
	sysAs := ctx.GetAccountState(state.SystemID)
	varDb := scoredb.NewVarDB(sysAs, h.txHash)
	scoreAddr := varDb.Address()
	if scoreAddr == nil {
		log.Printf("Failed to get score address by txHash\n")
		msg, _ := common.EncodeAny("Score not found by tx hash")
		return module.StatusContractNotFound, h.stepLimit, msg, nil
	}
	scoreAs := ctx.GetAccountState(scoreAddr.ID())

	var methodStr string
	if bytes.Equal(h.to.ID(), state.SystemID) {
		methodStr = deployInstall
	} else {
		methodStr = deployUpdate
	}
	// GET API
	cgah := newCallGetAPIHandler(newCommonHandler(h.from, scoreAddr, nil, h.StepAvail()), h.cc)
	status, _, result, _ := h.cc.Call(cgah)
	if status != module.StatusSuccess {
		return status, h.stepLimit, result, nil
	}
	apiInfo := scoreAs.APIInfo()
	typedObj, err := apiInfo.ConvertParamsToTypedObj(
		methodStr, scoreAs.NextContract().Params())
	if err != nil {
		status, result := scoreresult.StatusAndMessageForError(module.StatusSystemError, err)
		msg, _ := common.EncodeAny(result)
		return status, h.stepLimit, msg, nil
	}

	// 2. call on_install or on_update of the contract
	if cur := scoreAs.Contract(); cur != nil {
		cur.SetStatus(state.CSDisable)
	}
	handler := newCallHandlerFromTypedObj(
		newCommonHandler(h.from, scoreAddr, big.NewInt(0), h.StepAvail()),
		methodStr, typedObj, h.cc, true)

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	status, stepUsed2, _, _ := h.cc.Call(handler)
	h.stepUsed.Add(h.stepUsed, stepUsed2)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}
	if err = scoreAs.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		status, result := scoreresult.StatusAndMessageForError(module.StatusSystemError, err)
		msg, _ := common.EncodeAny(result)
		return status, h.stepLimit, msg, nil
	}
	varDb.Delete()

	return status, h.stepUsed, nil, nil
}

type callGetAPIHandler struct {
	*CommonHandler

	cc       CallContext
	disposed bool
	lock     sync.Mutex
	cs       ContractStore

	// set in ExecuteAsync()
	as state.AccountState
}

func newCallGetAPIHandler(ch *CommonHandler, cc CallContext) *callGetAPIHandler {
	return &callGetAPIHandler{CommonHandler: ch, cc: cc, disposed: false}
}

// It's never called
func (h *callGetAPIHandler) Prepare(ctx Context) (state.WorldContext, error) {
	as := ctx.GetAccountState(h.to.ID())
	c := as.NextContract()
	if c == nil {
		return nil, errors.New("No pending contract")
	}

	var err error
	h.lock.Lock()
	if h.cs == nil {
		h.cs, err = ctx.ContractManager().PrepareContractStore(ctx, c)
	}
	h.lock.Unlock()
	if err != nil {
		return nil, err
	}

	return ctx.GetFuture(nil), nil
}

func (h *callGetAPIHandler) ExecuteAsync(ctx Context) error {
	h.as = ctx.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return errors.New("FAIL: not a contract account")
	}

	conn := h.cc.GetConnection(h.EEType())
	if conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.NextContract()
	if c == nil {
		return errors.New("No pending contract")
	}
	var err error
	h.lock.Lock()
	h.cs, err = ctx.ContractManager().PrepareContractStore(ctx, c)
	h.lock.Unlock()
	if err != nil {
		return err
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		return err
	}

	h.lock.Lock()
	if !h.disposed {
		err = conn.GetAPI(h, path)
	}
	h.lock.Unlock()

	return err
}

func (h *callGetAPIHandler) SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error {
	log.Panicln("Unexpected SendResult() call")
	return nil
}

func (h *callGetAPIHandler) Dispose() {
	h.lock.Lock()
	h.disposed = true
	if h.cs != nil {
		h.cs.Dispose()
	}
	h.lock.Unlock()
}

func (h *callGetAPIHandler) EEType() string {
	c := h.as.NextContract()
	if c == nil {
		log.Println("No associated contract exists")
		return ""
	}
	return c.EEType()
}

func (h *callGetAPIHandler) GetValue(key []byte) ([]byte, error) {
	log.Panicln("Unexpected GetValue() call")
	return nil, nil
}

func (h *callGetAPIHandler) SetValue(key, value []byte) error {
	log.Panicln("Unexpected SetValue() call")
	return nil
}

func (h *callGetAPIHandler) DeleteValue(key []byte) error {
	log.Panicln("Unexpected DeleteValue() call")
	return nil
}

func (h *callGetAPIHandler) GetInfo() *codec.TypedObj {
	log.Panicln("Unexpected GetInfo() call")
	return nil
}

func (h *callGetAPIHandler) GetBalance(addr module.Address) *big.Int {
	log.Panicln("Unexpected GetBalance() call")
	return nil
}

func (h *callGetAPIHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	log.Panicln("Unexpected OnEvent() call")
}

func (h *callGetAPIHandler) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	log.Panicln("Unexpected call OnResult() from GetAPI()")
}

func (h *callGetAPIHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	log.Panicln("Unexpected call OnCall() from GetAPI()")
}

func (h *callGetAPIHandler) OnAPI(status uint16, info *scoreapi.Info) {
	s := module.Status(status)
	if s == module.StatusSuccess {
		h.as.SetAPIInfo(info)
	}
	h.cc.OnResult(s, new(big.Int), nil, nil)
}
