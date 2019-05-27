package module

import (
	"io"
)

const (
	BlockVersion1 = iota + 1
	BlockVersion2
)

type Block interface {
	Version() int
	ID() []byte
	Height() int64
	PrevID() []byte
	NextValidators() ValidatorList
	// voters are subset of previous preivous block's next validators
	Votes() CommitVoteSet
	NormalTransactions() TransactionList
	PatchTransactions() TransactionList
	Timestamp() int64
	Proposer() Address // can be nil. e.g. in genesis block.
	LogBloom() LogBloom
	Result() []byte

	MarshalHeader(w io.Writer) error
	MarshalBody(w io.Writer) error
	Marshal(w io.Writer) error

	ToJSON(rcpVersion int) (interface{}, error)
}

type BlockManager interface {
	GetBlockByHeight(height int64) (Block, error)
	GetLastBlock() (Block, error)
	GetBlock(id []byte) (Block, error)

	// WaitForBlock returns a channel that receives the block with the given
	// height.
	WaitForBlock(height int64) (<-chan Block, error)

	//  NewBlockFromReader creates a Block from reader. The returned block
	//	shall be imported by ImportBlock before it is Committed or Finalized.
	NewBlockFromReader(r io.Reader) (Block, error)

	//	Propose proposes a Block following the parent Block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful. Proposed block can be Commited or
	// 	Finalized.
	Propose(parentID []byte, votes CommitVoteSet, cb func(Block, error)) (canceler func() bool, err error)

	//	Import creates a Block from blockBytes and verifies the block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful. Imported block can be Commited or
	//	Finalized.
	Import(r io.Reader, cb func(Block, error)) (canceler func() bool, err error)
	ImportBlock(blk Block, cb func(Block, error)) (canceler func() bool, err error)

	Commit(Block) error

	//	Finalize updates world state according to Block block and removes non-finalized committed blocks with the same height as block from persistent storage.
	Finalize(Block) error

	GetTransactionInfo(id []byte) (TransactionInfo, error)
	Term()
}

type TransactionInfo interface {
	Block() Block
	Index() int
	Group() TransactionGroup
	Transaction() Transaction
	GetReceipt() (Receipt, error)
}
