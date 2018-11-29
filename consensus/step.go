package consensus

import "fmt"

type step int

const (
	stepPrepropose step = iota
	stepPropose
	stepPrevote
	stepPrevoteWait
	stepPrecommit
	stepPrecommitWait
	stepCommit
	stepNewHeight
)

func (step step) String() string {
	switch step {
	case stepPrepropose:
		return "stepPrepropose"
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
	case stepNewHeight:
		return "stepNewHeight"
	default:
		return fmt.Sprintf("step %d", step)
	}
}
