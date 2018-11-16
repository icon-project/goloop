package block

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var v2Codec = codec.MP

type blockV2HeaderFormat struct {
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte
	PrevID                 []byte
	VotesHash              []byte
	NextValidatorsHash     []byte
	PatchTransactionsHash  []byte
	NormalTransactionsHash []byte
	LogBloom               []byte
	Result                 []byte
}

type blockV2BodyFormat struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	Votes              []byte
}

type blockV2Format struct {
	blockV2HeaderFormat
	blockV2BodyFormat
}

type blockV2 struct {
	height             int64
	timestamp          time.Time
	proposer           module.Address
	prevID             []byte
	logBloom           []byte
	result             []byte
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
	nextValidators     module.ValidatorList
	votes              module.VoteList
	_id                []byte
}

func unixMicroFromTime(t time.Time) int64 {
	return t.UnixNano() / 1000
}

func timeFromUnixMicro(usec int64) time.Time {
	return time.Unix(0, usec*1000)
}

func (b *blockV2) Version() int {
	return common.BlockVersion2
}

func (b *blockV2) ID() []byte {
	if b._id == nil {
		buf := bytes.NewBuffer(nil)
		v2Codec.Marshal(buf, b._headerFormat())
		b._id = crypto.SHA3Sum256(buf.Bytes())
	}
	return b._id
}

func (b *blockV2) Height() int64 {
	return b.height
}

func (b *blockV2) PrevID() []byte {
	return b.prevID
}

func (b *blockV2) Votes() module.VoteList {
	return b.votes
}

func (b *blockV2) NextValidators() module.ValidatorList {
	return b.nextValidators
}

func (b *blockV2) Verify() error {
	return nil
}

func (b *blockV2) NormalTransactions() module.TransactionList {
	return b.normalTransactions
}

func (b *blockV2) PatchTransactions() module.TransactionList {
	return b.patchTransactions
}

func (b *blockV2) Timestamp() time.Time {
	return b.timestamp
}

func (b *blockV2) Proposer() module.Address {
	return b.proposer
}

func (b *blockV2) LogBloom() []byte {
	return b.logBloom
}

func (b *blockV2) Result() []byte {
	return b.result
}

func (b *blockV2) MarshalHeader(w io.Writer) error {
	return v2Codec.Marshal(w, b._headerFormat())
}

func (b *blockV2) MarshalBody(w io.Writer) error {
	return v2Codec.Marshal(w, b._bodyFormat())
}

func (b *blockV2) _headerFormat() *blockV2HeaderFormat {
	return &blockV2HeaderFormat{
		Version:                b.Version(),
		Height:                 b.height,
		Timestamp:              unixMicroFromTime(b.timestamp),
		Proposer:               b.proposer.Bytes(),
		PrevID:                 b.prevID,
		VotesHash:              b.votes.Hash(),
		NextValidatorsHash:     b.nextValidators.Hash(),
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogBloom:               b.logBloom,
		Result:                 b.result,
	}
}

func bssFromTransactionList(l module.TransactionList) [][]byte {
	var res [][]byte
	for it := l.Iterator(); it.Has(); it.Next() {
		tr, _, _ := it.Get()
		bs := tr.Bytes()
		res = append(res, bs)
	}
	return res
}

func (b *blockV2) _bodyFormat() *blockV2BodyFormat {
	return &blockV2BodyFormat{
		PatchTransactions:  bssFromTransactionList(b.patchTransactions),
		NormalTransactions: bssFromTransactionList(b.normalTransactions),
		Votes:              b.votes.Bytes(),
	}
}
