package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/icon-project/goloop/btp"
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
	NSFilter               []byte
	NTSDProofHashListHash  []byte
}

func (bh *blockV2HeaderFormat) RLPEncodeSelf(e codec.Encoder) error {
	if bh.NSFilter == nil && bh.NTSDProofHashListHash == nil {
		return e.EncodeListOf(
			bh.Version,
			bh.Height,
			bh.Timestamp,
			bh.Proposer,
			bh.PrevID,
			bh.VotesHash,
			bh.NextValidatorsHash,
			bh.PatchTransactionsHash,
			bh.NormalTransactionsHash,
			bh.LogsBloom,
			bh.Result,
		)
	}
	return e.EncodeListOf(
		bh.Version,
		bh.Height,
		bh.Timestamp,
		bh.Proposer,
		bh.PrevID,
		bh.VotesHash,
		bh.NextValidatorsHash,
		bh.PatchTransactionsHash,
		bh.NormalTransactionsHash,
		bh.LogsBloom,
		bh.Result,
		bh.NSFilter,
		bh.NTSDProofHashListHash,
	)
}

func (bh *blockV2HeaderFormat) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	cnt, err := d2.DecodeMulti(
		&bh.Version,
		&bh.Height,
		&bh.Timestamp,
		&bh.Proposer,
		&bh.PrevID,
		&bh.VotesHash,
		&bh.NextValidatorsHash,
		&bh.PatchTransactionsHash,
		&bh.NormalTransactionsHash,
		&bh.LogsBloom,
		&bh.Result,
		&bh.NSFilter,
		&bh.NTSDProofHashListHash,
	)
	if cnt == 11 && err == io.EOF {
		bh.NSFilter = nil
		bh.NTSDProofHashListHash = nil
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

type blockV2BodyFormat struct {
	PatchTransactions                [][]byte
	NormalTransactions               [][]byte
	Votes                            []byte
	NetworkTypeSectionDecisionProves [][]byte
}

func (bb *blockV2BodyFormat) RLPEncodeSelf(e codec.Encoder) error {
	if bb.NetworkTypeSectionDecisionProves == nil {
		return e.EncodeListOf(
			bb.PatchTransactions,
			bb.NormalTransactions,
			bb.Votes,
		)
	}
	return e.EncodeListOf(
		bb.PatchTransactions,
		bb.NormalTransactions,
		bb.Votes,
		bb.NetworkTypeSectionDecisionProves,
	)
}

func (bb *blockV2BodyFormat) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	cnt, err := d2.DecodeMulti(
		&bb.PatchTransactions,
		&bb.NormalTransactions,
		&bb.Votes,
		&bb.NetworkTypeSectionDecisionProves,
	)
	if cnt == 3 && err == io.EOF {
		bb.NetworkTypeSectionDecisionProves = nil
		return nil
	}
	if err != nil {
		return err
	}
	return nil
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
	_id                []byte
	_btpSection        module.BTPSection
	nsFilter           module.BitSetFilter
	_btpDigest         module.BTPDigest
	ntsdProofListHash  []byte
	ntsdProofList      module.NTSDProofList
	sm                 ServiceManager
	dbase              db.Database
}

func (b *blockV2) Version() int {
	return module.BlockVersion2
}

func (b *blockV2) ID() []byte {
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
		NSFilter:               b.nsFilter.Bytes(),
		NTSDProofHashListHash:  b.NTSDProofHashListHash(),
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
	ntsdProves, err := b.ntsdProofList.Proves()
	if err != nil {
		return nil, err
	}
	return &blockV2BodyFormat{
		PatchTransactions:                ptBss,
		NormalTransactions:               ntBss,
		Votes:                            b.votes.Bytes(),
		NetworkTypeSectionDecisionProves: ntsdProves,
	}, nil
}

func (b *blockV2) NewBlock(tr module.Transition) module.Block {
	blk := *b
	blk._nextValidators = tr.NextValidators()
	blk._btpSection = tr.BTPSection()
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
	if err = b.ntsdProofList.Flush(); err != nil {
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

func (b *blockV2) NetworkSectionFilter() module.BitSetFilter {
	return b.nsFilter
}

func (b *blockV2) BTPDigest() (module.BTPDigest, error) {
	if b._btpDigest == nil {
		bs, err := b.BTPSection()
		if err != nil {
			return nil, err
		}
		b._btpDigest = bs.Digest()
	}
	return b._btpDigest, nil
}

func (b *blockV2) BTPSection() (module.BTPSection, error) {
	if b._btpSection == nil {
		bs, err := b.sm.BTPSectionFromResult(b.result)
		if err != nil {
			return nil, err
		}
		b._btpSection = bs
	}
	return b._btpSection, nil
}

func (b *blockV2) NextProofContextMap() (module.BTPProofContextMap, error) {
	return b.sm.NextProofContextMapFromResult(b.result)
}

func (b *blockV2) NTSDProofHashListHash() []byte {
	return b.ntsdProofListHash
}

func (b *blockV2) NTSDProofList() module.NTSDProofList {
	return b.ntsdProofList
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
	b.block.nsFilter = module.BitSetFilterFromBytes(header.NSFilter, btp.NSFilterCap)
	b.block.ntsdProofListHash = header.NTSDProofHashListHash
	builder.RequestData(db.BytesByHash, header.VotesHash, voteSetBuilder{b})
	if header.NTSDProofHashListHash == nil {
		b.block.ntsdProofList = module.ZeroNTSDProofList{}
	} else {
		builder.RequestData(
			db.BytesByHash, header.NTSDProofHashListHash,
			&ntsdProofListBuilder{builder: b},
		)
	}
	return nil
}

type voteSetBuilder struct {
	builder *blockBuilder
}

func (b voteSetBuilder) OnData(value []byte, builder merkle.Builder) error {
	b.builder.block.votes = b.builder.vld(value)
	return nil
}

type ntsdProofListBuilder struct {
	builder *blockBuilder
	hashes  [][]byte
	proves  [][]byte
	nProves int
}

func (b *ntsdProofListBuilder) OnData(value []byte, builder merkle.Builder) error {
	format := ntsdProofHashListFormat{}
	codec.MustUnmarshalFromBytes(value, &format)
	b.hashes = format.NtsdProofHashes
	b.proves = make([][]byte, len(b.hashes))
	for _, hash := range b.hashes {
		builder.RequestData(db.BytesByHash, hash, &ntsdProofBuilder{b})
	}
	return nil
}

type ntsdProofBuilder struct {
	builder *ntsdProofListBuilder
}

func (b *ntsdProofBuilder) OnData(value []byte, builder merkle.Builder) error {
	valueHash := crypto.SHA3Sum256(value)
	for i, hash := range b.builder.hashes {
		if bytes.Equal(hash, valueHash) {
			b.builder.proves[i] = value
			b.builder.nProves++
			if b.builder.nProves == len(b.builder.hashes) {
				blk := b.builder.builder.block
				blk.ntsdProofList = newNTSDProofList(blk.dbase, b.builder.proves)
			}
		}
	}
	return nil
}

func newBlockWithBuilder(builder merkle.Builder, vld module.CommitVoteSetDecoder, c Chain, hash []byte) module.Block {
	blk := new(blockV2)
	blk._id = hash
	blk.sm = c.ServiceManager()
	blk.dbase = c.Database()
	builder.RequestData(db.BytesByHash, hash, &blockBuilder{block: blk, vld: vld})
	return blk
}
