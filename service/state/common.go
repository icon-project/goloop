package state

// TODO Check if everything here is adequate for state package.
const (
	GIGA = 1000 * 1000 * 1000
	TERA = 1000 * GIGA
	PETA = 1000 * TERA
	EXA  = 1000 * PETA
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
	StepTypeSchema           = "schema"
	StepTypeGetBase          = "getBase"
	StepTypeSetBase          = "setBase"
	StepTypeDeleteBase       = "deleteBase"
	StepTypeLogBase          = "logBase"
	StepTypeLog              = "log"
)

const (
	StepLimitTypeInvoke = "invoke"
	StepLimitTypeQuery  = "query"
)

var AllStepLimitTypes = []string{
	StepLimitTypeInvoke,
	StepLimitTypeQuery,
}

var InitialStepTypes = []string{
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
	StepTypeSchema,
	StepTypeGetBase,
	StepTypeSetBase,
	StepTypeDeleteBase,
	StepTypeLogBase,
	StepTypeLog,
}

func IsValidStepType(s string) bool {
	switch s {
	case StepTypeDefault,
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
		StepTypeApiCall:
		return true
	case StepTypeSchema,
		StepTypeGetBase,
		StepTypeSetBase,
		StepTypeDeleteBase,
		StepTypeLogBase,
		StepTypeLog:
		return true
	default:
		return false
	}
}

func IsValidStepLimitType(s string) bool {
	switch s {
	case StepLimitTypeInvoke, StepLimitTypeQuery:
		return true
	default:
		return false
	}
}
