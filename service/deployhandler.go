package service

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"strings"

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
	content     string
	contentType string
	params      []byte
	txHash      []byte

	timestamp int
	nonce     int
}

func newDeployHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext, force bool,
) *DeployHandler {
	var dataJSON struct {
		ContentType string          `json:"contentType""`
		Content     string          `json:"content"`
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

		params: dataJSON.Params,
	}
}

// nonce, timestamp, from
// data = from(20 bytes) + timestamp (32 bytes) + if exists, nonce (32 bytes)
// digest = sha3_256(data)
// contract address = digest[len(digest) - 20:] // get last 20bytes
func genContractAddr(from, timestamp, nonce []byte) []byte {
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
	sysAs := wc.GetAccountState(SystemID)

	update := false
	var codeBuf []byte
	var contractID []byte
	if bytes.Equal(h.to.ID(), SystemID) { // install
		var tsBytes [4]byte
		_ = binary.Write(bytes.NewBuffer(tsBytes[:]), binary.BigEndian, h.timestamp)
		var nBytes [4]byte
		_ = binary.Write(bytes.NewBuffer(nBytes[:]), binary.BigEndian, h.nonce)
		contractID = genContractAddr(h.from.ID(), tsBytes[:], nBytes[:])
	} else { // deploy for update
		contractID = h.to.ID()
		update = true
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

	if stepUsed.Cmp(h.stepLimit) > 0 {
		return module.StatusNotPayable, h.stepLimit, nil, nil
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
	if update == false {
		as.InitContractAccount(h.from)
	} else {
		if as.IsContract() == false || as.IsContractOwner(h.from) == false {
			return module.StatusSystemError, stepUsed, nil, nil
		}
	}

	as.DeployContract(codeBuf, h.eeType, h.contentType, h.params, h.txHash)
	scoreAddr := scoredb.NewVarDB(sysAs, h.txHash)
	_ = scoreAddr.Set(common.NewContractAddress(contractID))

	//if audit == false || deployer {
	ah := newAcceptHandler(h.from, common.NewContractAddress(contractID),
		nil, nil, h.params, h.cc)
	ah.ExecuteSync(wc)
	//}
	return module.StatusSuccess, nil, nil, nil
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
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

const (
	deployInstall = "on_install"
	deployUpdate  = "on_update"
)

func (h *AcceptHandler) ExecuteSync(wc WorldContext,
) (module.Status, *big.Int, []byte, module.Address) {
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
	cgah := &callGetAPIHandler{newCallHandler(newCommonHandler(
		h.from, scoreAddr, nil, stepAvail), nil, h.cc, false)}
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
	as := wc.GetAccountState(scoreAddr.ID())
	if cur := as.Contract(); cur != nil {
		cur.SetStatus(csDisable)
	}
	handler := newCallHandlerFromTypedObj(
		newCommonHandler(h.from, scoreAddr, nil, stepAvail),
		methodStr, typedObj, h.cc, true)

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	status, stepUsed2, _, _ := h.cc.Call(handler)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}
	if err = as.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
	varDb.Delete()

	return status, stepUsed1.Add(stepUsed1, stepUsed2), nil, nil
}

type callGetAPIHandler struct {
	*CallHandler
}

// It's never called
func (h *callGetAPIHandler) Prepare(wc WorldContext) (WorldContext, error) {
	c := h.as.NextContract()
	if c == nil {
		return nil, errors.New("No pending contract")
	}
	wc.ContractManager().PrepareContractStore(wc, c)

	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(nil)), nil
}

func (h *callGetAPIHandler) ExecuteAsync(wc WorldContext) error {
	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.NextContract()
	if c == nil {
		return errors.New("No active contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		err := h.conn.GetAPI(h, r.path)
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				if r.err == nil {
					if err := h.conn.GetAPI(h, r.path); err == nil {
						return
					}
				}
				h.OnResult(module.StatusSystemError, h.stepLimit, nil)
			}
		}()
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
