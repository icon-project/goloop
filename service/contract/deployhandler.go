package contract

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"sync"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type DeployHandler struct {
	*CommonHandler
	eeType         state.EEType
	content        *ContentBytes
	contentType    string
	params         []byte
	preDefinedAddr module.Address
}

type ContentBytes struct {
	Bytes []byte
	JSON  []byte
}

func (c *ContentBytes) UnmarshalJSON(b []byte) error {
	var os *string
	if err := json.Unmarshal(b, &os); err != nil {
		return err
	}
	if os == nil {
		c.JSON = nil
		c.Bytes = nil
		return nil
	}
	s := *os
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if bin, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		c.JSON = []byte(*os)
		c.Bytes = bin
		return nil
	}
}

func (c *ContentBytes) GetBytes() []byte {
	if c == nil {
		return nil
	} else {
		return c.Bytes
	}
}

func (c *ContentBytes) Count(revision module.Revision) int {
	if c == nil {
		return 0
	}
	if revision.LegacyContentCount() && len(c.JSON) > 0 {
		for _, b := range c.JSON[2:] {
			if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
				return len(c.JSON)
			}
		}
	}
	return len(c.Bytes)
}

type DeployData struct {
	ContentType string          `json:"contentType"`
	Content     *ContentBytes   `json:"content"`
	Params      json.RawMessage `json:"params"`
}

func newDeployHandler(
	ch *CommonHandler,
	data []byte,
) (*DeployHandler, error) {
	deploy, err := ParseDeployData(data)
	if err != nil {
		return nil, err
	}
	eeType, _ := state.EETypeFromContentType(deploy.ContentType)
	return &DeployHandler{
		CommonHandler: ch,
		content:       deploy.Content,
		contentType:   deploy.ContentType,
		eeType:        eeType,
		params:        deploy.Params,
	}, nil
}

func newDeployHandlerWithTypedObj(
	ch *CommonHandler,
	dataObj *codec.TypedObj,
) (*DeployHandler, error) {
	dataAny, err := common.DecodeAny(dataObj)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidData")
	}
	data, ok := dataAny.(map[string]interface{})
	if !ok {
		return nil, scoreresult.InvalidParameterError.Errorf("InvalidTypeForData(%T)", dataAny)
	}

	content, ok := data["content"].([]byte)
	if !ok {
		return nil, scoreresult.InvalidParameterError.New("InvalidDeployContent")
	}

	contentType, ok := data["contentType"].(string)
	if !ok {
		return nil, scoreresult.InvalidParameterError.New("InvalidDeployContentType")
	}

	eeType, ok := state.EETypeFromContentType(contentType)
	if !ok {
		return nil, scoreresult.InvalidParameterError.New("InvalidDeployContentType")
	}

	paramsAny := data["params"]
	var params []byte
	if paramsAny != nil {
		paramsJSO, err := common.AnyForJSON(paramsAny)
		if err != nil {
			return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidDeployParams")
		}
		params, err = json.Marshal(paramsJSO)
		if err != nil {
			return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidDeployParams")
		}
		params, err = common.CompactJSON(params)
		if err != nil {
			return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidDeployParams")
		}
	} else {
		params = nil
	}

	return &DeployHandler{
		CommonHandler: ch,
		content:       &ContentBytes{Bytes: content},
		contentType:   contentType,
		params:        params,
		eeType:        eeType,
	}, nil
}

func NewDeployHandlerForPreInstall(owner, scoreAddr module.Address, contentType string,
	content []byte, params *json.RawMessage, log log.Logger,
) *DeployHandler {
	var zero big.Int
	var p []byte
	if params == nil {
		p = nil
	} else {
		p = *params
	}
	return &DeployHandler{
		CommonHandler:  NewCommonHandler(owner, state.SystemAddress, &zero, false, log),
		content:        &ContentBytes{Bytes: content},
		contentType:    contentType,
		preDefinedAddr: scoreAddr,
		eeType:         state.MustEETypeFromContentType(contentType),
		params:         p,
	}
}

// genContractAddr generate new contract address
// nonce, timestamp, from
// data = from(20 bytes) + timestamp (32 bytes) + if exists, nonce (32 bytes)
// digest = sha3_256(data)
// contract address = digest[len(digest) - 20:] // get last 20bytes
// If there is salt, it would be added to nonce value.
func genContractAddr(from module.Address, timestamp int64, nonce, salt *big.Int) []byte {
	md := sha3.New256()

	// From ID(20 bytes)
	md.Write(from.ID())

	// Timestamp (32 bytes)
	md.Write(make([]byte, 24)) // add padding
	_ = binary.Write(md, binary.BigEndian, timestamp)

	// Nonce (32 bytes)
	if nonce != nil && nonce.Sign() != 0 {
		nb := intconv.BigIntToBytes(nonce)
		if len(nb) >= 32 {
			md.Write(nb[:32])
		} else {
			md.Write(make([]byte, 32-len(nb))) // add padding
			md.Write(nb)
		}
	}
	// Salt (16 bytes)
	if salt != nil && salt.Sign() != 0 {
		nb := intconv.BigIntToBytes(salt)
		if len(nb) >= 16 {
			md.Write(nb[len(nb)-16:])
		} else {
			md.Write(make([]byte, 16-len(nb))) // add padding
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

func getIDWithSalt(id []byte, salt *big.Int) []byte {
	if salt == nil {
		return id
	}
	var i big.Int
	i.SetBytes(id)
	i.Add(&i, salt)
	bs := i.Bytes()
	if len(bs) >= len(id) {
		return bs[len(bs)-len(id):]
	} else {
		bs2 := make([]byte, len(id))
		copy(bs2[len(id)-len(bs):], bs)
		return bs2
	}
}

func (h *DeployHandler) ExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, score module.Address) {
	h.Log.TSystemf("DEPLOY start to=%s", h.To)
	defer func() {
		if err != nil {
			h.Log.TSystemf("DEPLOY done status=%v", err)
		} else {
			h.Log.TSystemf("DEPLOY done status=%s score=%s", module.StatusSuccess, score)
		}
	}()

	return h.DoExecuteSync(cc)
}

func (h *DeployHandler) DoExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address) {
	sysAs := cc.GetAccountState(state.SystemID)

	update := false
	txInfo := cc.TransactionInfo()
	if txInfo == nil {
		return errors.CriticalUnknownError.New("InvalidTransactionInfo"), nil, nil
	}
	salt := cc.NextTransactionSalt()

	if cc.ReadOnlyMode() {
		return scoreresult.AccessDeniedError.New("DeployOnReadOnly"), nil, nil
	}

	var contractID []byte
	var as state.AccountState
	if h.To.Equal(state.SystemAddress) {
		// preDefinedAddr is not nil, it is pre-installed score.
		if h.preDefinedAddr != nil {
			if !h.preDefinedAddr.IsContract() {
				return scoreresult.InvalidParameterError.Errorf(
					"TargetMustBeContract(to=%s)", h.preDefinedAddr), nil, nil
			}
			contractID = h.preDefinedAddr.ID()
		} else {
			contractID = genContractAddr(h.From, txInfo.Timestamp, txInfo.Nonce, salt)
		}
		as = cc.GetAccountState(contractID)
	} else { // deploy for update
		if !h.To.IsContract() {
			return scoreresult.InvalidParameterError.Errorf(
				"TargetMustBeContract(to=%s)", h.To), nil, nil
		}
		contractID = h.To.ID()
		as = cc.GetAccountState(contractID)
		if h.To.Equal(cc.Governance()) && as.IsContract() == false {
			update = false
		} else {
			update = true
		}
	}

	// calculate stepUsed and apply it
	var st state.StepType
	if update {
		st = state.StepTypeContractUpdate
	} else {
		st = state.StepTypeContractCreate
	}
	codeLen := h.content.Count(cc.Revision())
	if !cc.ApplySteps(st, 1) ||
		!cc.ApplySteps(state.StepTypeContractSet, codeLen) {
		return scoreresult.ErrOutOfStep, nil, nil
	}

	if cc.DeployerWhiteListEnabled() == true && !cc.IsDeployer(h.From.String()) && h.preDefinedAddr == nil {
		h.Log.TSystemf("DEPLOY not in whitelist from=%s", h.From)
		return scoreresult.ErrAccessDenied, nil, nil
	}

	if !state.ValidateEEType(h.eeType) {
		return scoreresult.InvalidParameterError.Errorf("InvalidContentType(ct=%s)",
			h.contentType), nil, nil
	}

	if !cc.GetEnabledEETypes().Contains(h.eeType) {
		return scoreresult.InvalidParameterError.Errorf("UnsupportedContentType(ct=%s,enabled=%s)",
			h.contentType, cc.GetEnabledEETypes().String()), nil, nil
	}

	if update == false {
		if as.InitContractAccount(h.From) == false {
			return errors.ErrExecutionFail, nil, nil
		}
	} else {
		if as.IsContract() == false {
			return scoreresult.ErrContractNotFound, nil, nil
		}
		if as.IsContractOwner(h.From) == false {
			h.Log.TSystemf("DEPLOY different owner exp=%s from=%s",
				as.ContractOwner(), h.From)
			return scoreresult.ErrAccessDenied, nil, nil
		}
		if contract := as.Contract(); contract != nil && !h.eeType.AbleToUpdate(contract.EEType()) {
			return scoreresult.InvalidParameterError.Errorf("ProhibitToUpdate(old=%s,new=%s)",
				contract.EEType(), h.eeType), nil, nil
		}
	}
	scoreAddr := common.NewContractAddress(contractID)
	deployID := getIDWithSalt(txInfo.Hash, salt)
	h2a := scoredb.NewDictDB(sysAs, state.VarTxHashToAddress, 1)
	for h2a.Get(deployID) != nil {
		return scoreresult.AccessDeniedError.New("DuplicateDeployID"), nil, nil
	}

	oldTx, err := as.DeployContract(h.content.GetBytes(), h.eeType, h.contentType, h.params, deployID)
	if err != nil {
		return err, nil, nil
	}

	if err := h2a.Set(deployID, scoreAddr); err != nil {
		return err, nil, nil
	}
	if len(oldTx) > 0 {
		if err := h2a.Delete(oldTx); err != nil {
			return err, nil, nil
		}
	}

	if h.eeType.NeedAudit() == false || cc.AuditEnabled() == false ||
		cc.IsDeployer(h.From.String()) || h.preDefinedAddr != nil ||
		(cc.Revision().AutoAcceptGovernance() && cc.Governance().Equal(h.To)) {
		ah := NewAcceptHandler(NewCommonHandler(h.From, h.To, big.NewInt(0), false, h.Log), deployID, txInfo.Hash)
		status, acceptStepUsed, _, _ := cc.Call(ah, cc.StepAvailable())
		cc.DeductSteps(acceptStepUsed)
		if status != nil {
			return status, nil, nil
		}
	}

	return nil, common.MustEncodeAny(scoreAddr), scoreAddr
}

type AcceptHandler struct {
	*CommonHandler
	txHash      []byte
	auditTxHash []byte
}

func NewAcceptHandler(ch *CommonHandler, txHash []byte, auditTxHash []byte) *AcceptHandler {
	return &AcceptHandler{
		CommonHandler: ch,
		txHash:        txHash, auditTxHash: auditTxHash}
}

// It's never called
func (h *AcceptHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{{state.WorldIDStr, state.AccountWriteLock}}
	return ctx.GetFuture(lq), nil
}

func (h *AcceptHandler) ExecuteSync(cc CallContext) (err error, obj *codec.TypedObj, addr module.Address) {
	h.Log.TSystemf("ACCEPT start txhash=0x%x audit=0x%x", h.txHash, h.auditTxHash)
	defer func() {
		if err != nil {
			h.Log.TSystemf("ACCEPT done status=%v", err)
		} else {
			h.Log.TSystem("ACCEPT done status=SUCCESS")
		}
	}()

	// 1. call GetAPI
	sysAs := cc.GetAccountState(state.SystemID)
	h2a := scoredb.NewDictDB(sysAs, state.VarTxHashToAddress, 1)
	value := h2a.Get(h.txHash)
	if value == nil {
		err := scoreresult.ContractNotFoundError.New("NoSCOREForTx")
		return err, nil, nil
	}
	scoreAddr := value.Address()
	h2a.Delete(h.txHash)
	scoreAs := cc.GetAccountState(scoreAddr.ID())

	next := scoreAs.NextContract()
	if next.Status() != state.CSPending {
		return scoreresult.ContractNotFoundError.New("NoContractToAccept"), nil, nil
	}

	var methodStr string
	nextEEType := next.EEType()
	current := scoreAs.Contract()
	if current == nil {
		if method, ok := nextEEType.InstallMethod(); !ok {
			return scoreresult.MethodNotFoundError.New("NoInstallMethod"), nil, nil
		} else {
			methodStr = method
		}
	} else {
		if method, ok := nextEEType.UpdateMethod(current.EEType()); !ok {
			return scoreresult.MethodNotFoundError.New("NoUpdateMethod"), nil, nil
		} else {
			methodStr = method
		}
	}
	// GET API
	cgah := newCallGetAPIHandler(NewCommonHandler(h.From, scoreAddr, nil, false, h.Log))
	// It ignores stepUsed intentionally because it's not proper to charge step for GetAPI().
	status, _, _, _ := cc.Call(cgah, cc.StepAvailable())
	if status != nil {
		return status, nil, nil
	}
	apiInfo, err := scoreAs.APIInfo()
	if err != nil {
		return err, nil, nil
	}
	typedObj, err := apiInfo.ConvertParamsToTypedObj(
		methodStr, next.Params())
	if err != nil {
		return err, nil, nil
	}

	// 2. call on_install or on_update of the contract
	if err := scoreAs.ActivateNextContract(); err != nil {
		return err, nil, nil
	}
	handler := newCallHandlerWithParams(
		// NOTE : on_install or on_update should be invoked by score owner.
		// 	self.msg.sender should be deployer(score owner) when on_install or on_update is invoked in SCORE
		NewCommonHandler(scoreAs.ContractOwner(), scoreAddr, big.NewInt(0), false, h.Log),
		methodStr, typedObj, true)

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	status, stepUsed2, _, _ := cc.Call(handler, cc.StepAvailable())
	cc.DeductSteps(stepUsed2)
	if status != nil {
		return status, nil, nil
	}
	if err = scoreAs.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		return err, nil, nil
	}

	if cc.Revision().Has(module.ContractSetEvent) {
		cc.OnEvent(state.SystemAddress, [][]byte{
			[]byte("ContractSet(Address,Address,str,bytes)"),
			scoreAddr.Bytes(),
			h.From.Bytes(),
		}, [][]byte{
			[]byte(next.EEType()),
			next.CodeHash(),
		})
	}
	return nil, nil, nil
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
	h.Log.Panicf("SHOULD not reach here")
	return nil, nil
}

func (h *callGetAPIHandler) ExecuteAsync(cc CallContext) (ret error) {
	h.cc = cc
	h.as = cc.GetAccountState(h.To.ID())
	if !h.as.IsContract() {
		return scoreresult.Errorf(module.StatusContractNotFound, "Account(%s) is't contract", h.To)
	}

	conn := h.cc.GetProxy(h.EEType())
	if conn == nil {
		return NoAvailableProxy.Errorf(
			"FAIL to get connection of (%s)", h.EEType())
	}

	c := h.as.NextContract()
	if c == nil {
		return scoreresult.New(module.StatusContractNotFound,
			"No pending contract")
	}
	h.Log.TSystemf("GETAPI start code=<%x>", c.CodeHash())
	defer func() {
		if ret != nil {
			h.Log.TSystemf("GETAPI fail err=%+v", ret)
		}
	}()

	var err error
	h.lock.Lock()
	h.cs, err = cc.ContractManager().PrepareContractStore(cc, c)
	h.lock.Unlock()
	if err != nil {
		return err
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		h.Log.Warnf("FAIL to prepare contract. err=%+v", err)
		return PreparingContractError.New("FAIL to prepare contract")
	}

	h.lock.Lock()
	if !h.disposed {
		err = conn.GetAPI(h, path)
	}
	h.lock.Unlock()

	return err
}

func (h *callGetAPIHandler) SendResult(status error, steps *big.Int, result *codec.TypedObj) error {
	h.Log.Panicln("Unexpected SendResult() call")
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

func (h *callGetAPIHandler) EEType() state.EEType {
	c := h.as.NextContract()
	if c == nil {
		h.Log.Println("No associated contract exists")
		return ""
	}
	return c.EEType()
}

func (h *callGetAPIHandler) GetValue(key []byte) ([]byte, error) {
	h.Log.Panicln("Unexpected GetValue() call")
	return nil, nil
}

func (h *callGetAPIHandler) SetValue(key []byte, value []byte) ([]byte, error) {
	h.Log.Panicln("Unexpected SetValue() call")
	return nil, nil
}

func (h *callGetAPIHandler) DeleteValue(key []byte) ([]byte, error) {
	h.Log.Panicln("Unexpected DeleteValue() call")
	return nil, nil
}

func (h *callGetAPIHandler) ArrayDBContains(prefix, value []byte, limit int64) (bool, int, int, error) {
	h.Log.Panicln("Unexpected ArrayDBContains() call")
	return false, 0, 9, nil
}

func (h *callGetAPIHandler) GetInfo() *codec.TypedObj {
	h.Log.Panicln("Unexpected GetInfo() call")
	return nil
}

func (h *callGetAPIHandler) GetBalance(addr module.Address) *big.Int {
	h.Log.Panicln("Unexpected GetBalance() call")
	return nil
}

func (h *callGetAPIHandler) OnEvent(addr module.Address, indexed, data [][]byte) error {
	h.Log.Panicln("Unexpected OnEvent() call")
	return nil
}

func (h *callGetAPIHandler) OnResult(status error, flag int, steps *big.Int, result *codec.TypedObj) {
	if status == nil {
		h.Log.Panicln("Unexpected call OnResult() from GetAPI()")
	}
	h.OnAPI(status, nil)
}

func (h *callGetAPIHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	h.Log.Panicln("Unexpected call OnCall() from GetAPI()")
}

func (h *callGetAPIHandler) OnAPI(status error, info *scoreapi.Info) {
	if status == nil {
		h.Log.TSystemf("GETAPI done status=%s info=%v", module.StatusSuccess, info)
		if err := h.as.MigrateForRevision(h.cc.Revision()); err != nil {
			status = err
		} else {
			h.as.SetAPIInfo(info)
		}
	} else {
		s, _ := scoreresult.StatusOf(status)
		h.Log.TSystemf("GETAPI done status=%s msg=%s", s, status.Error())
	}
	h.cc.OnResult(status, 0, new(big.Int), nil, nil)
}

func (h *callGetAPIHandler) OnSetFeeProportion(portion int) {
	h.Log.Errorf("Unexpected call OnSetFeeProportion() from GetAPI()")
}

func (h *callGetAPIHandler) SetCode(code []byte) error {
	h.Log.Errorf("Unexpected call SetCode() from GetAPI()")
	return nil
}

func (h *callGetAPIHandler) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	h.Log.Errorf("Unexpected call GetObjGraph() from GetAPI()")
	return 0, nil, nil, nil
}

func (h *callGetAPIHandler) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	h.Log.Errorf("Unexpected call SetObjGraph() from GetAPI()")
	return nil
}

func ParseDeployData(data []byte) (*DeployData, error) {
	deploy := new(DeployData)
	if err := json.Unmarshal(data, deploy); err != nil {
		return nil, scoreresult.InvalidParameterError.Wrapf(err,
			"InvalidJSON(json=%s)", data)
	}
	return deploy, nil
}
