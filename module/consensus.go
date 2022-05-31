package module

type ConsensusStatus struct {
	Height   int64
	Round    int32
	Proposer bool
}

const (
	FlagNextProofContext = 0x1
	FlagBTPBlockHeader   = 0x2
	FlagBTPBlockProof    = 0x4
)

type Consensus interface {
	Start() error
	Term()
	GetStatus() *ConsensusStatus
	GetVotesByHeight(height int64) (CommitVoteSet, error)

	// GetBTPBlockHeaderAndProof returns header and proof according to the given
	// flag.
	GetBTPBlockHeaderAndProof(
		blk Block, nid int64, flag uint,
	) (btpBlk BTPBlockHeader, proof []byte, err error)
}
