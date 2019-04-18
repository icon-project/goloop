package contract

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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"golang.org/x/crypto/sha3"
)

type DeployHandler struct {
	*CommonHandler
	eeType         string
	content        []byte
	contentType    string
	params         []byte
	txHash         []byte
	preDefinedAddr module.Address
}

func newDeployHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, force bool,
) *DeployHandler {
	var dataJSON struct {
		ContentType string          `json:"contentType"`
		Content     common.HexBytes `json:"content"`
		Params      json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &dataJSON); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &DeployHandler{
		CommonHandler: newCommonHandler(from, to, value, stepLimit),
		content:       dataJSON.Content,
		contentType:   dataJSON.ContentType,
		// eeType is currently only python
		// but it should be checked later by json element
		eeType: "python",
		params: dataJSON.Params,
	}
}

func NewDeployHandlerForPreInstall(owner, scoreAddr module.Address, contentType string,
	content []byte, params *json.RawMessage,
) *DeployHandler {
	var zero big.Int
	var p []byte
	if params == nil {
		p = nil
	} else {
		p = *params
	}
	return &DeployHandler{
		CommonHandler: newCommonHandler(owner,
			common.NewContractAddress(state.SystemID),
			&zero, &zero),
		content:        content,
		contentType:    contentType,
		preDefinedAddr: scoreAddr,
		// eeType is currently only for python
		// but it should be checked later by json element
		eeType: "python",
		params: p,
	}
}

// nonce, timestamp, from
// data = from(20 bytes) + timestamp (32 bytes) + if exists, nonce (32 bytes)
// digest = sha3_256(data)
// contract address = digest[len(digest) - 20:] // get last 20bytes
func genContractAddr(from module.Address, timestamp int64, nonce *big.Int) []byte {
	md := sha3.New256()

	// From ID(20 bytes)
	md.Write(from.ID())

	// Timestamp (32 bytes)
	md.Write(make([]byte, 24)) // add padding
	_ = binary.Write(md, binary.BigEndian, timestamp)

	// Nonce (32 bytes)
	if nonce != nil && nonce.Sign() != 0 {
		var n common.HexInt
		n.Set(nonce)
		nb := n.Bytes()
		if len(nb) >= 32 {
			md.Write(nb[:32])
		} else {
			md.Write(make([]byte, 32-len(nb))) // add padding
			md.Write(nb)
		}
	}

	digest := md.Sum([]byte{})
	addr := make([]byte, 20)
	copy(addr, digest[len(digest)-20:])
	return addr
}

func (h *DeployHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (h *DeployHandler) ExecuteSync(cc CallContext) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	sysAs := cc.GetAccountState(state.SystemID)

	update := false
	var contractID []byte
	info := cc.GetInfo()
	if info == nil {
		msg, _ := common.EncodeAny("no GetInfo()")
		return module.StatusSystemError, h.StepUsed(), msg, nil
	} else {
		h.txHash = info[state.InfoTxHash].([]byte)
	}
	if bytes.Equal(h.to.ID(), state.SystemID) { // install
		// preDefinedAddr is not nil, it is pre-installed score.
		if h.preDefinedAddr != nil {
			contractID = h.preDefinedAddr.ID()
		} else {
			contractID = genContractAddr(h.from, info[state.InfoTxTimestamp].(int64), info[state.InfoTxNonce].(*big.Int))
		}
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
	if !h.ApplySteps(cc, st, 1) ||
		!h.ApplySteps(cc, state.StepTypeContractSet, codeLen) {
		msg, _ := common.EncodeAny("Not enough step limit")
		return module.StatusOutOfStep, h.StepUsed(), msg, nil
	}

	// store ScoreDeployInfo and ScoreDeployTXParams
	as := cc.GetAccountState(contractID)
	if update == false {
		if as.InitContractAccount(h.from) == false {
			msg, _ := common.EncodeAny("Already deployed contract")
			return module.StatusSystemError, h.StepUsed(), msg, nil
		}
	} else {
		if as.IsContract() == false {
			msg, _ := common.EncodeAny("Not a contract account")
			return module.StatusContractNotFound, h.StepUsed(), msg, nil
		}
		if as.IsContractOwner(h.from) == false {
			msg, _ := common.EncodeAny("Not a contract owner")
			return module.StatusAccessDenied, h.StepUsed(), msg, nil
		}
	}
	scoreAddr := common.NewContractAddress(contractID)
	as.DeployContract(h.content, h.eeType, h.contentType, h.params, h.txHash)
	scoreDB := scoredb.NewVarDB(sysAs, h.txHash)
	_ = scoreDB.Set(scoreAddr)

	if cc.AuditEnabled() == false ||
		cc.IsDeployer(h.from.String()) || h.preDefinedAddr != nil {
		ah := newAcceptHandler(h.from, h.to,
			nil, h.StepAvail(), h.txHash, h.txHash)
		status, acceptStepUsed, result, _ := ah.ExecuteSync(cc)
		h.DeductSteps(acceptStepUsed)
		if status != module.StatusSuccess {
			return status, h.StepUsed(), result, nil
		}
	}

	return module.StatusSuccess, h.StepUsed(), nil, scoreAddr
}

type AcceptHandler struct {
	*CommonHandler
	txHash      []byte
	auditTxHash []byte
}

func newAcceptHandler(from, to module.Address, value, stepLimit *big.Int, txHash []byte, auditTxHash []byte) *AcceptHandler {
	return &AcceptHandler{
		CommonHandler: newCommonHandler(from, to, value, stepLimit),
		txHash:        txHash, auditTxHash: auditTxHash}
}

// It's never called
func (h *AcceptHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{{state.WorldIDStr, state.AccountWriteLock}}
	return ctx.GetFuture(lq), nil
}

const (
	deployInstall = "on_install"
	deployUpdate  = "on_update"
)

func (h *AcceptHandler) ExecuteSync(cc CallContext) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	// 1. call GetAPI
	sysAs := cc.GetAccountState(state.SystemID)
	varDb := scoredb.NewVarDB(sysAs, h.txHash)
	scoreAddr := varDb.Address()
	if scoreAddr == nil {
		log.Printf("Failed to get score address by txHash\n")
		msg, _ := common.EncodeAny("Score not found by tx hash")
		return module.StatusContractNotFound, h.stepLimit, msg, nil
	}
	scoreAs := cc.GetAccountState(scoreAddr.ID())

	var methodStr string
	if scoreAs.Contract() == nil {
		methodStr = deployInstall
	} else {
		methodStr = deployUpdate
	}
	// GET API
	cgah := newCallGetAPIHandler(newCommonHandler(h.from, scoreAddr, nil, h.StepAvail()))
	// It ignores stepUsed intentionally because it's not proper to charge step for GetAPI().
	status, _, result, _ := cc.Call(cgah)
	if status != module.StatusSuccess {
		return status, h.StepUsed(), result, nil
	}
	apiInfo := scoreAs.APIInfo()
	typedObj, err := apiInfo.ConvertParamsToTypedObj(
		methodStr, scoreAs.NextContract().Params())
	if err != nil {
		status, _ := scoreresult.StatusOf(err)
		msg, _ := common.EncodeAny(err.Error())
		return status, h.StepUsed(), msg, nil
	}

	// 2. call on_install or on_update of the contract
	if cur := scoreAs.Contract(); cur != nil {
		cur.SetStatus(state.CSInactive)
	}
	handler := newCallHandlerFromTypedObj(
		newCommonHandler(h.from, scoreAddr, big.NewInt(0), h.StepAvail()),
		methodStr, typedObj, true)

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	status, stepUsed2, _, _ := cc.Call(handler)
	h.DeductSteps(stepUsed2)
	if status != module.StatusSuccess {
		return status, h.StepUsed(), nil, nil
	}
	if err = scoreAs.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		status, _ := scoreresult.StatusOf(err)
		msg, _ := common.EncodeAny(err.Error())
		return status, h.StepUsed(), msg, nil
	}
	varDb.Delete()

	return status, h.StepUsed(), nil, nil
}

type callGetAPIHandler struct {
	*CommonHandler

	disposed bool
	lock     sync.Mutex
	cs       ContractStore

	// set in ExecuteAsync()
	cc CallContext
	as state.AccountState
}

func newCallGetAPIHandler(ch *CommonHandler) *callGetAPIHandler {
	return &callGetAPIHandler{CommonHandler: ch, disposed: false}
}

// It's never called
func (h *callGetAPIHandler) Prepare(ctx Context) (state.WorldContext, error) {
	log.Panicf("SHOULD not reach here")
	return nil, nil
}

func (h *callGetAPIHandler) ExecuteAsync(cc CallContext) error {
	h.cc = cc

	h.as = cc.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return InvalidContractError.New("NotAContractAccount")
	}

	conn := h.cc.GetProxy(h.EEType())
	if conn == nil {
		return errors.InvalidStateError.Errorf(
			"FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.NextContract()
	if c == nil {
		return errors.InvalidStateError.Errorf("No pending contract")
	}
	var err error
	h.lock.Lock()
	h.cs, err = cc.ContractManager().PrepareContractStore(cc, c)
	h.lock.Unlock()
	if err != nil {
		return err
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		return errors.Wrapc(err, PreparingContractError, "FAIL to prepare contract")
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
