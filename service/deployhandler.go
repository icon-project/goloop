package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/big"
	"sync"

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

func (h *DeployHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{"", AccountWriteLock},
	}
	return wc.GetFuture(lq), nil
}

func (h *DeployHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int,
	*codec.TypedObj, module.Address,
) {
	sysAs := wc.GetAccountState(SystemID)

	update := false
	var contractID []byte
	if bytes.Equal(h.to.ID(), SystemID) { // install
		info := h.cc.GetInfo()
		if info == nil {
			return module.StatusSystemError, h.stepLimit, nil, nil
		}
		contractID = genContractAddr(h.from, info[InfoTxTimestamp].(int64), info[InfoTxNonce].(*big.Int))
	} else { // deploy for update
		contractID = h.to.ID()
		update = true
	}

	var stepUsed *big.Int

	// calculate stepUsed and apply it
	codeLen := int64(len(h.content))
	stepUsed = new(big.Int)
	stepUsed.SetInt64(codeLen)
	stepCost := big.NewInt(wc.StepsFor(StepTypeContractCreate, 1))
	stepUsed.Mul(stepUsed, stepCost)

	if stepUsed.Cmp(h.stepLimit) > 0 {
		return module.StatusNotPayable, h.stepLimit, nil, nil
	}

	// store ScoreDeployInfo and ScoreDeployTXParams
	as := wc.GetAccountState(contractID)
	if update == false {
		as.InitContractAccount(h.from)
	} else {
		if as.IsContract() == false || as.IsContractOwner(h.from) == false {
			return module.StatusSystemError, stepUsed, nil, nil
		}
	}
	scoreAddr := common.NewContractAddress(contractID)
	as.DeployContract(h.content, h.eeType, h.contentType, h.params, h.txHash)
	scoreDb := scoredb.NewVarDB(sysAs, h.txHash)
	_ = scoreDb.Set(scoreAddr)

	//if audit == false || deployer {
	ah := newAcceptHandler(h.from, h.to, //common.NewContractAddress(contractID),
		nil, h.stepLimit, h.params, h.cc)
	status, acceptStepUsed, _, _ := ah.ExecuteSync(wc)
	if acceptStepUsed != nil {
		stepUsed = stepUsed.Add(stepUsed, acceptStepUsed)
	}
	if status != module.StatusSuccess {
		return status, stepUsed, nil, nil
	}
	//}
	return module.StatusSuccess, stepUsed, nil, scoreAddr
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

func (h *AcceptHandler) StepLimit() *big.Int {
	return h.stepLimit
}

// It's never called
func (h *AcceptHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.GetFuture(lq), nil
}

const (
	deployInstall = "on_install"
	deployUpdate  = "on_update"
)

func (h *AcceptHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int,
	*codec.TypedObj, module.Address,
) {
	// 1. call GetAPI
	stepAvail := h.stepLimit
	sysAs := wc.GetAccountState(SystemID)
	varDb := scoredb.NewVarDB(sysAs, h.txHash)
	scoreAddr := varDb.Address()
	if scoreAddr == nil {
		log.Printf("Failed to get score address by txHash\n")
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
	scoreAs := wc.GetAccountState(scoreAddr.ID())

	var methodStr string
	if bytes.Equal(h.to.ID(), SystemID) {
		methodStr = deployInstall
	} else {
		methodStr = deployUpdate
	}
	// GET API
	cgah := newCallGetAPIHandler(newCommonHandler(h.from, scoreAddr, nil, stepAvail), h.cc)
	status, stepUsed1, _, _ := h.cc.Call(cgah)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}
	apiInfo := scoreAs.APIInfo()
	typedObj, err := apiInfo.ConvertParamsToTypedObj(
		methodStr, scoreAs.NextContract().Params())
	if err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}

	// 2. call on_install or on_update of the contract
	stepAvail = stepAvail.Sub(stepAvail, stepUsed1)
	if cur := scoreAs.Contract(); cur != nil {
		cur.SetStatus(csDisable)
	}
	handler := newCallHandlerFromTypedObj(
		newCommonHandler(h.from, scoreAddr, big.NewInt(0), stepAvail),
		methodStr, typedObj, h.cc, true)

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	status, stepUsed2, _, _ := h.cc.Call(handler)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}
	if err = scoreAs.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
	varDb.Delete()

	return status, stepUsed1.Add(stepUsed1, stepUsed2), nil, nil
}

type callGetAPIHandler struct {
	*CommonHandler

	cc       CallContext
	canceled bool
	lock     sync.Mutex

	// set in ExecuteAsync()
	as AccountState
}

func newCallGetAPIHandler(ch *CommonHandler, cc CallContext) *callGetAPIHandler {
	return &callGetAPIHandler{CommonHandler: ch, cc: cc, canceled: false}
}

// It's never called
func (h *callGetAPIHandler) Prepare(wc WorldContext) (WorldContext, error) {
	as := wc.GetAccountState(h.to.ID())
	c := as.NextContract()
	if c == nil {
		return nil, errors.New("No pending contract")
	}
	wc.ContractManager().PrepareContractStore(wc, c)

	return wc.GetFuture(nil), nil
}

func (h *callGetAPIHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.to.ID())
	conn := h.cc.GetConnection(h.EEType())
	if conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.NextContract()
	if c == nil {
		return errors.New("No pending contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)

	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		err := conn.GetAPI(h, r.path)
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				h.lock.Lock()
				if !h.canceled {
					if r.err == nil {
						if err := conn.GetAPI(h, r.path); err == nil {
							return
						}
					}
					h.cc.OnResult(module.StatusSystemError, h.stepLimit, nil, nil)
				}
				h.lock.Unlock()
			}
		}()
	}
	return nil
}

func (h *callGetAPIHandler) SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error {
	log.Panicln("Unexpected SendResult() call")
	return nil
}

func (h *callGetAPIHandler) Cancel() {
	h.lock.Lock()
	h.canceled = true
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

func (h *callGetAPIHandler) OnAPI(info *scoreapi.Info) {
	h.as.SetAPIInfo(info)
	h.cc.OnResult(module.StatusSuccess, new(big.Int), nil, nil)
}
