package block

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var v2codec = codec.MP

type blockV2HeaderForCodec struct {
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte
	PrevID                 []byte
	VotesHash              []byte
	NextValidatorsHash     []byte
	PatchTransactionsRoot  []byte
	NormalTransactionsRoot []byte
	LogBloom               []byte
	Result                 []byte
}

type blockV2BodyForCodec struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	Votes              []byte
}

type blockV2ForCodec struct {
	blockV2HeaderForCodec
	blockV2BodyForCodec
}

type blockV2Header struct {
	Height                 int64
	Timestamp              time.Time
	Proposer               module.Address
	PrevID                 []byte
	VotesHash              []byte
	NextValidatorsHash     []byte
	PatchTransactionsRoot  []byte
	NormalTransactionsRoot []byte
	LogBloom               []byte
	Result                 []byte
	id                     []byte
	nextValidators         []module.Validator
}

type blockV2Impl struct {
	blockV2Header
	PatchTransactions  module.TransactionList
	NormalTransactions module.TransactionList
	Votes              module.VoteList
}

type blockV2 struct {
	blockV2Impl
}

func (b *blockV2) Version() int {
	return 2
}

func (b *blockV2) ID() []byte {
	if b.id == nil {
		buf := bytes.NewBuffer(nil)
		v2codec.Marshal(buf, b.blockV2Header)
		b.id = crypto.SHA3Sum256(buf.Bytes())
	}
	return b.id
}

func (b *blockV2) Height() int64 {
	return b.blockV2Header.Height
}

func (b *blockV2) PrevID() []byte {
	return b.blockV2Header.PrevID
}

func (b *blockV2) Votes() module.VoteList {
	return nil
}

func (b *blockV2) NextValidators() module.ValidatorList {
	return nil
}

func (b *blockV2) Verify() error {
	return nil
}

func (b *blockV2) NormalTransactions() module.TransactionList {
	return b.blockV2Impl.NormalTransactions
}

func (b *blockV2) PatchTransactions() module.TransactionList {
	return b.blockV2Impl.PatchTransactions
}

func (b *blockV2) Timestamp() time.Time {
	return b.blockV2Impl.Timestamp
}

func (b *blockV2) Proposer() module.Address {
	return b.blockV2Impl.Proposer
}

func (b *blockV2) LogBloom() []byte {
	return b.blockV2Impl.LogBloom
}

func (b *blockV2) Result() []byte {
	return b.blockV2Impl.Result
}

func (b *blockV2) MarshalHeader(w io.Writer) {
}

func (b *blockV2) MarshalBody(w io.Writer) {
}

type blockV2Param struct {
	parent             module.Block
	timestamp          time.Time
	proposer           module.Address
	logBloom           []byte
	result             []byte
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
	nextValidators     module.ValidatorList
	votes              module.VoteList
}

// TODO rename
func newBlockV2(blockv2 *blockV2ForCodec) module.Block {
	return nil
}

func newBlockV2FromHeaderForCodec(*blockV2HeaderForCodec) module.Block {
	return nil
}

func newBlockV2FromParam(param *blockV2Param) module.Block {
	block := blockV2{
		blockV2Impl: blockV2Impl{
			blockV2Header: blockV2Header{
				Height:                 param.parent.Height() + 1,
				Timestamp:              param.timestamp,
				Proposer:               param.proposer,
				PrevID:                 param.parent.ID(),
				VotesHash:              param.votes.Hash(),
				NextValidatorsHash:     param.nextValidators.Hash(),
				PatchTransactionsRoot:  param.patchTransactions.Hash(),
				NormalTransactionsRoot: param.normalTransactions.Hash(),
				LogBloom:               param.logBloom,
				Result:                 param.result,
			},
			PatchTransactions:  param.patchTransactions,
			NormalTransactions: param.normalTransactions,
			Votes:              param.votes,
		},
	}
	return &block
}
