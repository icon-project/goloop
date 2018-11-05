package module

import (
	"io"
	"time"
)

type Block interface {
	Version() int
	ID() []byte
	Height() int64
	PrevID() []byte
	NextValidators() ValidatorList
	//	TODO remove
	Verify() error
	Votes() VoteList
	NormalTransactions() TransactionList
	PatchTransactions() TransactionList
	Timestamp() time.Time
	Proposer() Address
	LogBloom() []byte
	Result() []byte
	PatchReceipts() ReceiptList
	NormalReceipts() ReceiptList

	MarshalHeader(w io.Writer)
	MarshalBody(w io.Writer)
}

type BlockManager interface {
	GetBlock(id []byte) Block

	//	Propose proposes a Block following the parent Block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Propose(parentID []byte, votes VoteList, cb func(Block, error)) (canceler func() bool, err error)

	//	Import creates a Block from blockBytes.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Import(r io.Reader, cb func(Block, error)) (canceler func() bool, err error)

	Commit(Block) error

	//	Finalize updates world state according to Block block and removes non-finalized committed blocks with the same height as block from persistent storage.
	Finalize(Block) error
}
