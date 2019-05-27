package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var v2Codec = codec.MP

const blockV2String = "2.0"

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
	timestamp          int64
	proposer           module.Address
	prevID             []byte
	logBloom           module.LogBloom
	result             []byte
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
	nextValidators     module.ValidatorList
	votes              module.CommitVoteSet
	_id                []byte
}

func (b *blockV2) Version() int {
	return module.BlockVersion2
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

func (b *blockV2) Votes() module.CommitVoteSet {
	return b.votes
}

func (b *blockV2) NextValidators() module.ValidatorList {
	return b.nextValidators
}

func (b *blockV2) NormalTransactions() module.TransactionList {
	return b.normalTransactions
}

func (b *blockV2) PatchTransactions() module.TransactionList {
	return b.patchTransactions
}

func (b *blockV2) Timestamp() int64 {
	return b.timestamp
}

func (b *blockV2) Proposer() module.Address {
	return b.proposer
}

func (b *blockV2) LogBloom() module.LogBloom {
	return b.logBloom
}

func (b *blockV2) Result() []byte {
	return b.result
}

func (b *blockV2) MarshalHeader(w io.Writer) error {
	return v2Codec.Marshal(w, b._headerFormat())
}

func (b *blockV2) MarshalBody(w io.Writer) error {
	bf, err := b._bodyFormat()
	if err != nil {
		return err
	}
	return v2Codec.Marshal(w, bf)
}

func (b *blockV2) Marshal(w io.Writer) error {
	if err := b.MarshalHeader(w); err != nil {
		return err
	}
	return b.MarshalBody(w)
}

func (b *blockV2) _headerFormat() *blockV2HeaderFormat {
	var proposerBS []byte
	if b.proposer != nil {
		proposerBS = b.proposer.Bytes()
	}
	return &blockV2HeaderFormat{
		Version:                b.Version(),
		Height:                 b.height,
		Timestamp:              b.timestamp,
		Proposer:               proposerBS,
		PrevID:                 b.prevID,
		VotesHash:              b.votes.Hash(),
		NextValidatorsHash:     b.nextValidators.Hash(),
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogBloom:               b.logBloom.CompressedBytes(),
		Result:                 b.result,
	}
}

func (b *blockV2) ToJSON(rpcVersion int) (interface{}, error) {
	res := make(map[string]interface{})
	res["version"] = blockV2String
	res["prev_block_hash"] = hex.EncodeToString(b.PrevID())
	// TODO calc merkle_tree_root_hash
	res["merkle_tree_root_hash"] = hex.EncodeToString(b.NormalTransactions().Hash())
	res["time_stamp"] = b.Timestamp()
	res["confirmed_transaction_list"] = b.NormalTransactions()
	res["block_hash"] = hex.EncodeToString(b.ID())
	res["height"] = b.Height()
	if b.Proposer() != nil {
		res["peer_id"] = fmt.Sprintf("hx%x", b.Proposer().ID())
	} else {
		res["peer_id"] = ""
	}
	// TODO add signautre?
	res["signature"] = ""
	return res, nil
}

func bssFromTransactionList(l module.TransactionList) ([][]byte, error) {
	var res [][]byte
	for it := l.Iterator(); it.Has(); it.Next() {
		tr, _, err := it.Get()
		if err != nil {
			return nil, err
		}
		bs := tr.Bytes()
		res = append(res, bs)
	}
	return res, nil
}

func (b *blockV2) _bodyFormat() (*blockV2BodyFormat, error) {
	ptbss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntbss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	return &blockV2BodyFormat{
		PatchTransactions:  ptbss,
		NormalTransactions: ntbss,
		Votes:              b.votes.Bytes(),
	}, nil
}
