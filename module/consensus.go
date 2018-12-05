package module

type ConsensusStatus struct {
	Height   int64
	Round    int32
	Proposer bool
}

type Consensus interface {
	Start()
	GetStatus() *ConsensusStatus
}
