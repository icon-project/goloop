package module

import (
	"bytes"
	"io"

	"github.com/icon-project/goloop/common/db"
)

const (
	BlockVersion0 = iota
	BlockVersion1
	BlockVersion2
)

type BlockData interface {
	Version() int
	ID() []byte
	Height() int64
	PrevID() []byte
	NextValidatorsHash() []byte
	// voters are subset of previous previous block's next validators
	Votes() CommitVoteSet
	NormalTransactions() TransactionList
	PatchTransactions() TransactionList
	Timestamp() int64
	Proposer() Address // can be nil. e.g. in genesis block.
	LogsBloom() LogsBloom
	Result() []byte

	MarshalHeader(w io.Writer) error
	MarshalBody(w io.Writer) error
	Marshal(w io.Writer) error

	ToJSON(version JSONVersion) (interface{}, error)
	NewBlock(vl ValidatorList) Block
	Hash() []byte
}

func BlockDataToBytes(blk BlockData) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := blk.Marshal(buf)
	return buf.Bytes(), err
}

type Block interface {
	BlockData
	NextValidators() ValidatorList
}

type BlockCandidate interface {
	Block
	Dup() BlockCandidate
	Dispose()
}

// ImportXXX is used as flag value of BlockManager.Import and
// BlockManager.ImportBlock.
const (
	ImportByForce = 0x1
)

type BlockDataFactory interface {
	NewBlockDataFromReader(r io.Reader) (BlockData, error)
}

type BlockManager interface {
	GetBlockByHeight(height int64) (Block, error)
	GetLastBlock() (Block, error)
	GetBlock(id []byte) (Block, error)

	// WaitForBlock returns a channel that receives the block with the given
	// height.
	WaitForBlock(height int64) (<-chan Block, error)

	//  NewBlockDataFromReader creates a BlockData from reader. The returned block
	//	shall be imported by ImportBlock before it is Committed or Finalized.
	NewBlockDataFromReader(r io.Reader) (BlockData, error)

	//	Propose proposes a Block following the parent Block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful. Proposed block can be Commited or
	// 	Finalized.
	Propose(parentID []byte, votes CommitVoteSet, cb func(BlockCandidate, error)) (canceler Canceler, err error)

	//	Import creates a Block from blockBytes and verifies the block.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful. Imported block can be Commited or
	//	Finalized.
	Import(r io.Reader, flags int, cb func(BlockCandidate, error)) (canceler Canceler, err error)
	ImportBlock(blk BlockData, flags int, cb func(BlockCandidate, error)) (canceler Canceler, err error)

	Commit(BlockCandidate) error

	//	Finalize updates world state according to BlockCandidate and removes non-finalized committed blocks with the same height as block from persistent storage.
	Finalize(BlockCandidate) error

	GetTransactionInfo(id []byte) (TransactionInfo, error)
	Term()

	// WaitTransaction waits for a transaction with timestamp between
	// bi.Timestamp() - TimestampThreshold and current time +
	// TimestampThreshold. If such a transaction is available now, the function
	// returns false and callback cb is not called.
	WaitForTransaction(parentID []byte, cb func()) bool

	// SendAndWaitTransaction sends a transaction, and get a channel to
	// to wait for the result of it.
	SendTransactionAndWait(result []byte, height int64, txi interface{}) (tid []byte, rc <-chan interface{}, err error)

	// WaitTransactionResult check whether it knows about the transaction
	// and wait for the result.
	WaitTransactionResult(id []byte) (rc <-chan interface{}, err error)

	// ExportBlock exports blocks assuring specified block ranges.
	ExportBlocks(from, to int64, dst db.Database, on func(height int64) error) error

	// ExportGenesis exports genesis to the writer based on the block.
	ExportGenesis(blk Block, votes CommitVoteSet, writer GenesisStorageWriter) error

	// GetGenesisVotes returns available votes from genesis storage.
	// They are available only when it starts from genesis.
	GetGenesisData() (Block, CommitVoteSet, error)

	// NewConsensusInfo returns a ConsensusInfo with blk's proposer and
	// votes in blk.
	NewConsensusInfo(blk Block) (ConsensusInfo, error)
}

type TransactionInfo interface {
	Block() Block
	Index() int
	Group() TransactionGroup
	Transaction() (Transaction, error)
	GetReceipt() (Receipt, error)
}
