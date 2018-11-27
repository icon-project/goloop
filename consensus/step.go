package consensus

type step int

const (
	stepPropose step = iota
	stepPrevote
	stepPrevoteWait
	stepPrecommit
	stepPrecommitWait
	stepCommit
	stepNewHeight
)
