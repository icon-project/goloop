package consensus

import "fmt"

type step int

const (
	stepNewHeight step = iota
	stepTransactionWait
	stepNewRound
	stepPropose
	stepPrevote
	stepPrevoteWait
	stepPrecommit
	stepPrecommitWait
	stepCommit
)

func (step step) String() string {
	switch step {
	case stepNewHeight:
		return "stepNewHeight"
	case stepTransactionWait:
		return "stepTransactionWait"
	case stepNewRound:
		return "stepNewRound"
	case stepPropose:
		return "stepPropose"
	case stepPrevote:
		return "stepPrevote"
	case stepPrevoteWait:
		return "stepPrevoteWait"
	case stepPrecommit:
		return "stepPrecommit"
	case stepPrecommitWait:
		return "stepPrecommitWait"
	case stepCommit:
		return "stepCommit"
	default:
		return fmt.Sprintf("step %d", step)
	}
}
