package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

var v2Codec = codec.BC

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
	LogsBloom              []byte
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
	logsBloom          module.LogsBloom
	result             []byte
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
	nextValidatorsHash []byte
	_nextValidators    module.ValidatorList
	votes              module.CommitVoteSet

	// mutable data
	mu  sync.Mutex
	_id []byte
}

func (b *blockV2) Version() int {
	return module.BlockVersion2
}

func (b *blockV2) ID() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b._id == nil {
		bs := v2Codec.MustMarshalToBytes(b._headerFormat())
		b._id = crypto.SHA3Sum256(bs)
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

func (b *blockV2) NextValidatorsHash() []byte {
	return b.nextValidatorsHash
}

func (b *blockV2) NextValidators() module.ValidatorList {
	return b._nextValidators
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

func (b *blockV2) LogsBloom() module.LogsBloom {
	return b.logsBloom
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
		NextValidatorsHash:     b.nextValidatorsHash,
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogsBloom:              b.logsBloom.CompressedBytes(),
		Result:                 b.result,
	}
}

func (b *blockV2) ToJSON(version module.JSONVersion) (interface{}, error) {
	res := make(map[string]interface{})
	res["version"] = blockV2String
	res["prev_block_hash"] = hex.EncodeToString(b.PrevID())
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
	res["signature"] = ""
	return res, nil
}

func bssFromTransactionList(l module.TransactionList) ([][]byte, error) {
	var res [][]byte
	for it := l.Iterator(); it.Has(); log.Must(it.Next()) {
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
	ptBss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntBss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	return &blockV2BodyFormat{
		PatchTransactions:  ptBss,
		NormalTransactions: ntBss,
		Votes:              b.votes.Bytes(),
	}, nil
}

func (b *blockV2) NewBlock(vl module.ValidatorList) module.Block {
	if !bytes.Equal(b.nextValidatorsHash, vl.Hash()) {
		return nil
	}
	blk := *b
	blk._nextValidators = vl
	return &blk
}

func (b *blockV2) Hash() []byte {
	return b.ID()
}

func (b *blockV2) FinalizeHeader(dbase db.Database) error {
	hb, err := db.NewCodedBucket(dbase, db.BytesByHash, nil)
	if err != nil {
		return err
	}
	if err = hb.Put(b._headerFormat()); err != nil {
		return err
	}
	if err = hb.Set(db.Raw(b.Votes().Hash()), db.Raw(b.Votes().Bytes())); err != nil {
		return err
	}
	hh, err := db.NewCodedBucket(dbase, db.BlockHeaderHashByHeight, nil)
	if err != nil {
		return err
	}
	if err = hh.Set(b.Height(), db.Raw(b.ID())); err != nil {
		return err
	}
	return nil
}

func (b *blockV2) GetVoters(ctx base.BlockHandlerContext) (module.ValidatorList, error) {
	if b.Height() == 0 {
		return nil, nil
	}
	prevBlk, err := ctx.GetBlockByHeight(b.Height() - 1)
	if err != nil {
		return nil, err
	}
	return prevBlk.NextValidators(), nil
}

func (b *blockV2) VerifyTimestamp(
	prev module.BlockData, prevVoters module.ValidatorList,
) error {
	if b.Height() > 1 && b.Timestamp() != b.Votes().Timestamp() {
		return errors.New("bad timestamp")
	}
	if b.Height() > 1 && prev.Timestamp() >= b.Timestamp() {
		return errors.New("non-increasing timestamp")
	}
	return nil
}

func (b *blockV2) Copy() module.Block {
	// blockV2 is safe to be used in multiple goroutine
	return b
}

type blockBuilder struct {
	vld   module.CommitVoteSetDecoder
	block *blockV2
}

func (b *blockBuilder) OnData(value []byte, builder merkle.Builder) error {
	header := new(blockV2HeaderFormat)
	err := v2Codec.Unmarshal(bytes.NewReader(value), header)
	if err != nil {
		return err
	}
	b.block.height = header.Height
	b.block.timestamp = header.Timestamp
	if addr, err := newProposer(header.Proposer); err != nil {
		return err
	} else {
		b.block.proposer = addr
	}
	b.block.prevID = header.PrevID
	b.block.logsBloom = txresult.NewLogsBloomFromCompressed(header.LogsBloom)
	b.block.patchTransactions = transaction.NewTransactionListWithBuilder(builder, header.PatchTransactionsHash)
	b.block.normalTransactions = transaction.NewTransactionListWithBuilder(builder, header.NormalTransactionsHash)
	b.block.nextValidatorsHash = header.NextValidatorsHash
	b.block.result = header.Result
	if vs, err := state.NewValidatorSnapshotWithBuilder(builder, header.NextValidatorsHash); err != nil {
		return err
	} else {
		b.block._nextValidators = vs
	}
	builder.RequestData(db.BytesByHash, header.VotesHash, voteSetBuilder{b})
	return nil
}

type voteSetBuilder struct {
	builder *blockBuilder
}

func (b voteSetBuilder) OnData(value []byte, builder merkle.Builder) error {
	b.builder.block.votes = b.builder.vld(value)
	return nil
}

func newBlockWithBuilder(builder merkle.Builder, vld module.CommitVoteSetDecoder, hash []byte) module.Block {
	blk := new(blockV2)
	blk._id = hash
	builder.RequestData(db.BytesByHash, hash, &blockBuilder{block: blk, vld: vld})
	return blk
}
