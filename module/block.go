package module

type Address []byte

type Vote struct {
	Hight int64
	Round int
	Type byte
	BlockID []byte
	Signature []byte
}

type Block interface {
	Height() int
	ID() []byte
	Votes() []Vote
	NextValidators() []Address
}

type BlockManager interface {
	//	Propose let this BlockManager proposes a Block following the parent Block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Propose(parent Block, votes []Vote, cb func(Block, error)) (canceler func()bool, err error)

	//	Import creates a Block from blockBytes.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Import(blockBytes []byte, cb func(Block, error)) (canceler func()bool, err error)
	Commit(Block) error
	Finalize(Block) error
}
