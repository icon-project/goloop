package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/atomic"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
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

type blockV2 struct {
	// immutables
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

	// caches
	_id         atomic.Cache[[]byte]
	_btpSection atomic.Cache[module.BTPSection]
	_btpDigest  atomic.Cache[module.BTPDigest]
	_nextPCM    atomic.Cache[module.BTPProofContextMap]
}

func (b *blockV2) Version() int {
	return module.BlockVersion2
}

func (b *blockV2) ID() []byte {
	return b._id.Get(func() []byte {
		bs := v2Codec.MustMarshalToBytes(b._headerFormat())
		return crypto.SHA3Sum256(bs)
	})
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
	blk := *b
	blk._nextValidators = tr.NextValidators()
	blk._btpSection.Set(tr.BTPSection())
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
	return b._btpDigest.TryGet(func() (module.BTPDigest, error) {
		bs, err := b.BTPSection()
		if err != nil {
			return nil, err
		}
		return bs.Digest(), nil
	})
}

func (b *blockV2) BTPSection() (module.BTPSection, error) {
	return b._btpSection.TryGet(func() (module.BTPSection, error) {
		return b.sm.BTPSectionFromResult(b.result)
	})
}

func (b *blockV2) NextProofContextMap() (module.BTPProofContextMap, error) {
	return b._nextPCM.TryGet(func() (module.BTPProofContextMap, error) {
		return b.sm.NextProofContextMapFromResult(b.result)
	})
}

func (b *blockV2) NTSHashEntryList() (module.NTSHashEntryList, error) {
	return b.BTPDigest()
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
