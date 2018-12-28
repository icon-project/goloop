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
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

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

	// GET API
	cgah := &callGetAPIHandler{newCallHandler(h.from, h.to, nil, stepAvail, nil, h.cc)}
	status, stepUsed1, _, _ := h.cc.Call(cgah)
	if status != module.StatusSuccess {
		return status, h.stepLimit, nil, nil
	}

	// 2. call on_install or on_update of the contract
	stepAvail = stepAvail.Sub(stepAvail, stepUsed1)
	as := wc.GetAccountState(addr)
	// TODO Set current contract to disable
	if cur := as.Contract(); cur != nil {
		cur.SetStatus(csDisable)
	}

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

	// state -> active if failed to on_install, set inactive
	// on_install or on_update
	handler := wc.ContractManager().GetHandler(h.cc, h.from,
		common.NewContractAddress(addr), nil, stepAvail,
		ctypeCall, data)
	status, stepUsed2, _, _ := h.cc.Call(handler)
	if err = as.AcceptContract(h.txHash, h.auditTxHash); err != nil {
		return module.StatusSystemError, h.stepLimit, nil, nil
	}
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

func (h *callGetAPIHandler) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	log.Panicln("Unexpected call OnResult() from GetAPI()")
}

func (h *callGetAPIHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	log.Panicln("Unexpected call OnCall() from GetAPI()")
}

func (h *callGetAPIHandler) OnAPI(info *scoreapi.Info) {
	// TODO implement after deciding how to store
	panic("implement me")
}
