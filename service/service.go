package service

import (
	"errors"
	"math/big"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	GIGA = 1000 * 1000 * 1000
	TERA = 1000 * GIGA
	PETA = 1000 * TERA
	EXA  = 1000 * PETA
)

var (
	ErrNotEnoughBalance   = errors.New("NotEnoughBalance")
	ErrTimeOut            = errors.New("TimeOut")
	ErrInvalidValueValue  = errors.New("InvalidValueValue")
	ErrInvalidFeeValue    = errors.New("InvalidFeeValue")
	ErrInvalidDataValue   = errors.New("InvalidDataValue")
	ErrNotEnoughStep      = errors.New("NotEnoughStep")
	ErrContractIsRequired = errors.New("ContractIsRequired")
	ErrInvalidHashValue   = errors.New("InvalidHashValue")
	ErrNotContractAccount = errors.New("NotContractAccount")
	ErrNotEOA             = errors.New("NotEOA")
	ErrNoActiveContract   = errors.New("NoActiveContract")
	ErrNotContractOwner   = errors.New("NotContractOwner")
	ErrBlacklisted        = errors.New("Blacklisted")
	ErrInvalidMethod      = errors.New("InvalidMethod")
)

type StepType string

const (
	StepTypeDefault          = "default"
	StepTypeContractCall     = "contractCall"
	StepTypeContractCreate   = "contractCreate"
	StepTypeContractUpdate   = "contractUpdate"
	StepTypeContractDestruct = "contractDestruct"
	StepTypeContractSet      = "contractSet"
	StepTypeGet              = "get"
	StepTypeSet              = "set"
	StepTypeReplace          = "replace"
	StepTypeDelete           = "delete"
	StepTypeInput            = "input"
	StepTypeEventLog         = "eventLog"
	StepTypeApiCall          = "apiCall"
)

type BlockInfo struct {
	Timestamp int64
	Height    int64
}

type TransactionInfo struct {
	Index     int32
	Hash      []byte
	Timestamp int64
	Nonce     *big.Int
}

type ContractInfo struct {
	Owner module.Address
}

type WorldContext interface {
	WorldState
	StepsFor(t StepType, n int) int64
	StepPrice() *big.Int
	BlockTimeStamp() int64
	BlockHeight() int64
	Treasury() module.Address
	Governance() module.Address
	GetInfo() map[string]interface{}
	ContractManager() ContractManager
	EEManager() eeproxy.Manager
	WorldStateChanged(ws WorldState) WorldContext
	WorldVirtualState() WorldVirtualState
	GetFuture(lq []LockRequest) WorldContext
	SetTransactionInfo(ti *TransactionInfo)
	GetTransactionInfo(ti *TransactionInfo)
	SetContractInfo(si *ContractInfo)
}

type Transaction interface {
	module.Transaction
	PreValidate(wc WorldContext, update bool) error
	GetHandler(cm ContractManager) (TransactionHandler, error)
	Timestamp() int64
	Nonce() *big.Int
}

type Receipt interface {
	module.Receipt
	AddLog(addr module.Address, indexed, data [][]byte)
	SetCumulativeStepUsed(cumulativeUsed *big.Int)
	SetResult(status module.Status, used, price *big.Int, addr module.Address)
}
