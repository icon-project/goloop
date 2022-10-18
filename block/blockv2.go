package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

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

const V2String = "2.0"

type V2HeaderFormat struct {
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
}

func (bh *V2HeaderFormat) RLPEncodeSelf(e codec.Encoder) error {
	if bh.NSFilter == nil {
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
	)
}

func (bh *V2HeaderFormat) RLPDecodeSelf(d codec.Decoder) error {
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
	)
	if cnt == 11 && err == io.EOF {
		bh.NSFilter = nil
		return nil
	}
	return err
}

type V2BodyFormat struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	Votes              []byte
	BTPDigest          []byte
}

func (bb *V2BodyFormat) RLPEncodeSelf(e codec.Encoder) error {
	if bb.BTPDigest == nil {
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
		bb.BTPDigest,
	)
}

func (bb *V2BodyFormat) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	cnt, err := d2.DecodeMulti(
		&bb.PatchTransactions,
		&bb.NormalTransactions,
		&bb.Votes,
		&bb.BTPDigest,
	)
	if cnt == 3 && err == io.EOF {
		bb.BTPDigest = nil
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

type blockV2Immut struct {
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
	nsFilter           module.BitSetFilter
	sm                 ServiceManager
}

type blockV2Mut struct {
	_id         []byte
	_btpSection module.BTPSection
	_btpDigest  module.BTPDigest
	_nextPCM    module.BTPProofContextMap
}

type blockV2 struct {
	blockV2Immut

	mu sync.Mutex
	blockV2Mut
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

func (b *blockV2) _headerFormat() *V2HeaderFormat {
	var proposerBS []byte
	if b.proposer != nil {
		proposerBS = b.proposer.Bytes()
	}
	return &V2HeaderFormat{
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
	}
}

func (b *blockV2) ToJSON(version module.JSONVersion) (interface{}, error) {
	res := make(map[string]interface{})
	res["version"] = V2String
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

func (b *blockV2) _bodyFormat() (*V2BodyFormat, error) {
	ptBss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntBss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	bd, err := b.BTPDigest()
	if err != nil {
		return nil, err
	}
	return &V2BodyFormat{
		PatchTransactions:  ptBss,
		NormalTransactions: ntBss,
		Votes:              b.votes.Bytes(),
		BTPDigest:          bd.Bytes(),
	}, nil
}

func (b *blockV2) NewBlock(tr module.Transition) module.Block {
	blk := blockV2{
		blockV2Immut: b.blockV2Immut,
		blockV2Mut:   b.blockV2Mut,
	}
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

func (b *blockV2) NetworkSectionFilter() module.BitSetFilter {
	return b.nsFilter
}

func (b *blockV2) BTPDigest() (module.BTPDigest, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b._btpDigest == nil {
		bs, err := b.btpSectionInLock()
		if err != nil {
			return nil, err
		}
		b._btpDigest = bs.Digest()
	}
	return b._btpDigest, nil
}

func (b *blockV2) btpSectionInLock() (module.BTPSection, error) {
	if b._btpSection == nil {
		bs, err := b.sm.BTPSectionFromResult(b.result)
		if err != nil {
			return nil, err
		}
		b._btpSection = bs
	}
	return b._btpSection, nil
}

func (b *blockV2) BTPSection() (module.BTPSection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.btpSectionInLock()
}

func (b *blockV2) NextProofContextMap() (module.BTPProofContextMap, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b._nextPCM == nil {
		nextPCM, err := b.sm.NextProofContextMapFromResult(b.result)
		if err != nil {
			return nil, err
		}
		b._nextPCM = nextPCM
	}
	return b._nextPCM, nil
}

func (b *blockV2) NTSHashEntryList() (module.NTSHashEntryList, error) {
	return b.BTPDigest()
}

type blockBuilder struct {
	vld   module.CommitVoteSetDecoder
	block *blockV2
}

func (b *blockBuilder) OnData(value []byte, builder merkle.Builder) error {
	header := new(V2HeaderFormat)
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

func newBlockWithBuilder(builder merkle.Builder, vld module.CommitVoteSetDecoder, c Chain, hash []byte) module.Block {
	blk := new(blockV2)
	blk._id = hash
	blk.sm = c.ServiceManager()
	builder.RequestData(db.BytesByHash, hash, &blockBuilder{block: blk, vld: vld})
	return blk
}

func NewBlockReaderFromFormat(hf *V2HeaderFormat, bf *V2BodyFormat) io.Reader {
	var buf bytes.Buffer
	err := v2Codec.Marshal(&buf, hf)
	if err != nil {
		log.Panicf("fail to marshal: %+v", err)
	}
	err = v2Codec.Marshal(&buf, bf)
	if err != nil {
		log.Panicf("fail to marshal: %+v", err)
	}
	return &buf
}

func FormatFromBlock(blk module.Block) (*V2HeaderFormat, *V2BodyFormat, error) {
	if v2, ok := blk.(*blockV2); ok {
		bodyFmt, err := v2._bodyFormat()
		if err != nil {
			return nil, nil, err
		}
		return v2._headerFormat(), bodyFmt, nil
	}
	return nil, nil, errors.Errorf("not block v2 height=%d version=%d", blk.Height(), blk.Version())
}
