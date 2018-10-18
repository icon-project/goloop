package module

type Address interface {
	String() string
	Bytes() []byte
}

type Vote interface {
	Voter() Address
	Bytes() []byte
}

type Block interface {
	ID() []byte
	Height() int64
	PrevRound() int
	PrevID() []byte
	Votes() []Vote
	NextValidators() []Address
	Verify() error
}

type BlockManager interface {
	//	Propose proposes a Block following the parent Block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Propose(parent Block, votes []Vote, cb func(Block, error)) (canceler func() bool, err error)

	//	Import creates a Block from blockBytes.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Import(blockBytes []byte, cb func(Block, error)) (canceler func() bool, err error)
	Commit(Block) error
	Finalize(Block) error
}
