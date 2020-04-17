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
	StepTypeDefaultGet       = "defaultGet"
	StepTypeDefaultSet       = "defaultSet"
	StepTypeReplaceBase      = "replaceBase"
	StepTypeDefaultDelete    = "defaultDelete"
	StepTypeEventLogBase     = "eventLogBase"
)

const (
	StepLimitTypeInvoke = "invoke"
	StepLimitTypeQuery  = "query"
)

var AllStepLimitTypes = []string{
	StepLimitTypeInvoke,
	StepLimitTypeQuery,
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
	StepTypeDefaultGet,
	StepTypeDefaultSet,
	StepTypeReplaceBase,
	StepTypeDefaultDelete,
	StepTypeEventLogBase,
}
