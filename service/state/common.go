package state

import (
	"errors"
)

// TODO Check if everything here is adequate for state package.
const (
	GIGA = 1000 * 1000 * 1000
	TERA = 1000 * GIGA
	PETA = 1000 * TERA
	EXA  = 1000 * PETA
)

var (
	ErrNotEnoughBalance   = errors.New("NotEnoughBalance")
	ErrTimeOut            = errors.New("TimeOut")
	ErrFutureTransaction  = errors.New("FutureTransaction")
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

var AllStepTypes = []string{
	StepTypeDefault,
	StepTypeContractCall,
	StepTypeContractCreate,
	StepTypeContractUpdate,
	StepTypeContractDestruct,
	StepTypeContractSet,
	StepTypeGet,
	StepTypeSet,
	StepTypeReplace,
	StepTypeDelete,
	StepTypeInput,
	StepTypeEventLog,
	StepTypeApiCall,
}
