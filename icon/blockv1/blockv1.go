/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockv1

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type headerFormat struct {
	// V10 and V20 common
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte
	PrevHash               []byte
	BlockVotesHash         []byte
	NextValidatorsHash     []byte
	PatchTransactionsHash  []byte
	NormalTransactionsHash []byte
	LogsBloom              []byte
	Result                 []byte

	// V1X common
	PrevID    []byte
	VersionV0 string
	Signature common.Signature

	// V13 only, final value (after executing TXes in this block)
	StateHashV0     []byte
	ReceiptRoot     []byte
	RepsRoot        []byte
	NextRepsRoot    []byte
	LogsBloomV0     []byte
	LeaderVotesHash []byte
	NextLeader      []byte
}

type bodyFormat struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	BlockVotes         *blockv0.BlockVoteList
	LeaderVotes        *blockv0.LeaderVoteList
}

type format struct {
	headerFormat
	bodyFormat
}

type blockDetail interface {
	headerFormat() *headerFormat
	bodyFormat() (*bodyFormat, error)
	id() []byte
	transactionsRoot() []byte
	BlockVotes() *blockv0.BlockVoteList
	LeaderVotes() *blockv0.LeaderVoteList
	NextValidatorsHash() []byte
	NextValidators() module.ValidatorList
	NewBlock(vl module.ValidatorList) module.Block
}

type Block struct {
	blockDetail
	height             int64
	timestamp          int64
	proposer           module.Address
	prevHash           []byte
	logsBloom          module.LogsBloom
	result             []byte
	signature          common.Signature
	prevID             []byte
	versionV0          string
	_id                []byte
	_hash              []byte
	_transactionsRoot  []byte
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
}

func (b *Block) Version() int {
	return module.BlockVersion1
}

func (b *Block) Height() int64 {
	return b.height
}

func (b *Block) Timestamp() int64 {
	return b.timestamp
}

func (b *Block) Proposer() module.Address {
	return b.proposer
}

func (b *Block) PrevHash() []byte {
	if b == nil {
		return nil
	}
	return b.prevHash
}

func (b *Block) LogsBloom() module.LogsBloom {
	return b.logsBloom
}

func (b *Block) Result() []byte {
	return b.result
}

func (b *Block) PrevID() []byte {
	return b.prevID
}

func (b *Block) VersionV0() string {
	return b.versionV0
}

func (b *Block) Signature() common.Signature {
	return b.signature
}

func (b *Block) ID() []byte {
	if b._id == nil {
		b._id = b.id()
	}
	return b._id
}

func (b *Block) Hash() []byte {
	if b._hash == nil {
		bs := codec.BC.MustMarshalToBytes(b.headerFormat())
		b._hash = crypto.SHA3Sum256(bs)
	}
	return b._hash
}

func (b *Block) TransactionsRoot() []byte {
	if b._transactionsRoot == nil {
		b._transactionsRoot = b.transactionsRoot()
	}
	return b._transactionsRoot
}

func (b *Block) PatchTransactions() module.TransactionList {
	return b.patchTransactions
}

func (b *Block) NormalTransactions() module.TransactionList {
	return b.normalTransactions
}

func (b *Block) MarshalHeader(w io.Writer) error {
	return codec.BC.Marshal(w, b.headerFormat())
}

func (b *Block) MarshalBody(w io.Writer) error {
	bf, err := b.bodyFormat()
	if err != nil {
		return err
	}
	return codec.BC.Marshal(w, bf)
}

func (b *Block) Marshal(w io.Writer) error {
	if err := b.MarshalHeader(w); err != nil {
		return err
	}
	return b.MarshalBody(w)
}

func (b *Block) ToJSON(version module.JSONVersion) (interface{}, error) {
	res := make(map[string]interface{})
	res["version"] = b.versionV0
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

func (b *Block) Votes() module.CommitVoteSet {
	return b.BlockVotes()
}

type blockV11 struct {
	Block
}

func (b *blockV11) headerFormat() *headerFormat {
	var proposerBS []byte
	if b.proposer != nil {
		proposerBS = b.proposer.Bytes()
	}
	return &headerFormat{
		Version:                b.Version(),
		Height:                 b.height,
		Timestamp:              b.timestamp,
		Proposer:               proposerBS,
		PrevHash:               b.prevHash,
		BlockVotesHash:         nil,
		NextValidatorsHash:     nil,
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogsBloom:              b.logsBloom.CompressedBytes(),
		Result:                 b.result,

		PrevID:    b.prevID,
		VersionV0: b.versionV0,
		Signature: b.signature,

		StateHashV0:     nil,
		ReceiptRoot:     nil,
		RepsRoot:        nil,
		NextRepsRoot:    nil,
		LogsBloomV0:     nil,
		LeaderVotesHash: nil,
		NextLeader:      nil,
	}
}

func hexBytes(data []byte) []byte {
	return []byte(common.RawHexBytes(data).String())
}

func (b *blockV11) id() []byte {
	bs := make([]byte, 0, 128+8)
	bs = append(bs, hexBytes(b.prevID)...)
	bs = append(bs, hexBytes(b.TransactionsRoot())...)
	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, uint64(b.timestamp))
	bs = append(bs, ts...)
	return crypto.SHA3Sum256(bs)
}

func (b *blockV11) bodyFormat() (*bodyFormat, error) {
	ptBss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntBss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	return &bodyFormat{
		PatchTransactions:  ptBss,
		NormalTransactions: ntBss,
	}, nil
}

func (b *blockV11) transactionsRoot() []byte {
	var txs []module.Transaction
	for it := b.normalTransactions.Iterator(); it.Has(); _ = it.Next() {
		tx, _, _ := it.Get()
		txs = append(txs, tx)
	}
	return transaction.NewTransactionListV1FromSlice(txs).Hash()
}

func (b *blockV11) BlockVotes() *blockv0.BlockVoteList {
	return nil
}

func (b *blockV11) LeaderVotes() *blockv0.LeaderVoteList {
	return nil
}

func (b *blockV11) NextValidatorsHash() []byte {
	return nil
}

func (b *blockV11) NextValidators() module.ValidatorList {
	return nil
}

func (b *blockV11) NewBlock(vl module.ValidatorList) module.Block {
	res := *b
	return &res
}

type blockV13 struct {
	Block
	nextValidatorsHash []byte
	stateHashV0        []byte
	receiptsRoot       []byte
	repsRoot           []byte
	nextRepsRoot       []byte
	logsBloomV0        module.LogsBloom
	nextLeader         module.Address
	_nextValidators    module.ValidatorList
	_receipts          module.ReceiptList
	blockVotes         *blockv0.BlockVoteList
	leaderVotes        *blockv0.LeaderVoteList
}

func (b *blockV13) headerFormat() *headerFormat {
	var proposerBS []byte
	if b.proposer != nil {
		proposerBS = b.proposer.Bytes()
	}
	return &headerFormat{
		Version:                b.Version(),
		Height:                 b.height,
		Timestamp:              b.timestamp,
		Proposer:               proposerBS,
		PrevHash:               b.prevHash,
		BlockVotesHash:         b.blockVotes.Hash(),
		NextValidatorsHash:     b._nextValidators.Hash(),
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogsBloom:              b.logsBloom.CompressedBytes(),
		Result:                 b.result,

		PrevID:    b.prevID,
		VersionV0: b.versionV0,
		Signature: b.signature,

		StateHashV0:     b.stateHashV0,
		ReceiptRoot:     b.receiptsRoot,
		RepsRoot:        b.repsRoot,
		NextRepsRoot:    b.nextRepsRoot,
		LogsBloomV0:     b.logsBloomV0.CompressedBytes(),
		LeaderVotesHash: b.leaderVotes.Hash(),
		NextLeader:      b.nextLeader.Bytes(),
	}
}

func (b *blockV13) id() []byte {
	items := make([]merkle.Item, 0, 13)
	items = append(items,
		merkle.HashedItem(b.prevID),
		merkle.HashedItem(b.TransactionsRoot()),
		merkle.HashedItem(b.receiptsRoot),
		merkle.HashedItem(b.stateHashV0),
		merkle.HashedItem(b.repsRoot),
		merkle.HashedItem(b.nextRepsRoot),
		merkle.HashedItem(b.leaderVotes.Root()),
		merkle.HashedItem(b.blockVotes.Root()),
		merkle.ValueItem(b.logsBloomV0.LogBytes()),
		merkle.ValueItem(intconv.SizeToBytes(uint64(b.height))),
		merkle.ValueItem(intconv.SizeToBytes(uint64(b.timestamp))),
		merkle.ValueItem(b.proposer.ID()),
		merkle.ValueItem(b.nextLeader.ID()),
	)
	return merkle.CalcHashOfList(items)
}

func (b *blockV13) transactionsRoot() []byte {
	var items []merkle.Item
	for iter := b.normalTransactions.Iterator(); iter.Has(); _ = iter.Next() {
		tx, _, _ := iter.Get()
		items = append(items, merkle.HashedItem(tx.ID()))
	}
	return merkle.CalcHashOfList(items)
}

func (b *blockV13) NextValidatorsHash() []byte {
	return b.nextValidatorsHash
}

func (b *blockV13) NextLeader() module.Address {
	return b.nextLeader
}

func (b *blockV13) BlockVotes() *blockv0.BlockVoteList {
	return b.blockVotes
}

func (b *blockV13) NextValidators() module.ValidatorList {
	return b._nextValidators
}

func (b *blockV13) NewBlock(vl module.ValidatorList) module.Block {
	res := *b
	res._nextValidators = vl
	return &res
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

func (b *blockV13) bodyFormat() (*bodyFormat, error) {
	ptBss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntBss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	return &bodyFormat{
		PatchTransactions:  ptBss,
		NormalTransactions: ntBss,
		BlockVotes:         b.blockVotes,
		LeaderVotes:        b.leaderVotes,
	}, nil
}

func newProposer(bs []byte) (module.Address, error) {
	if bs != nil {
		addr, err := common.NewAddress(bs)
		if err != nil {
			return nil, errors.CriticalFormatError.Wrapf(err,
				"InvalidProposer(bs=%#x)", bs)
		} else {
			return addr, nil
		}
	}
	return nil, nil
}

func newTransactionListFromBSS(
	dbase db.Database, bss [][]byte,
) (module.TransactionList, error) {
	ts := make([]module.Transaction, len(bss))
	for i, bs := range bss {
		if tx, err := transaction.NewTransaction(bs); err != nil {
			return nil, err
		} else {
			ts[i] = tx
		}
	}
	return transaction.NewTransactionListFromSlice(dbase, ts), nil
}

func newBlockV11FromHeader(dbase db.Database, header *headerFormat, patches module.TransactionList,
	normalTxs module.TransactionList) (*Block, error) {
	proposer, err := newProposer(header.Proposer)
	if err != nil {
		return nil, err
	}
	var blk = &blockV11{
		Block: Block{
			height:             header.Height,
			timestamp:          header.Timestamp,
			proposer:           proposer,
			prevHash:           header.PrevHash,
			logsBloom:          txresult.NewLogsBloomFromCompressed(header.LogsBloom),
			result:             header.Result,
			signature:          header.Signature,
			prevID:             header.PrevID,
			versionV0:          header.VersionV0,
			patchTransactions:  patches,
			normalTransactions: normalTxs,
		},
	}
	blk.Block.blockDetail = blk
	return &blk.Block, nil
}

func newBlockV11FromBlockFormat(dbase db.Database, format *format) (*Block, error) {
	patches, err := newTransactionListFromBSS(dbase, format.PatchTransactions)
	if err != nil {
		return nil, err
	}
	normalTxs, err := newTransactionListFromBSS(dbase, format.NormalTransactions)
	if err != nil {
		return nil, err
	}
	return newBlockV11FromHeader(dbase, &format.headerFormat, patches, normalTxs)
}

func newBlockV11FromHeaderFormat(dbase db.Database, header *headerFormat) (*Block, error) {
	patches := transaction.NewTransactionListFromHash(dbase, header.PatchTransactionsHash)
	if patches == nil {
		return nil, errors.Errorf("TranscationListFromHash(%x) failed", header.PatchTransactionsHash)
	}
	normalTxs := transaction.NewTransactionListFromHash(dbase, header.NormalTransactionsHash)
	if normalTxs == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", header.NormalTransactionsHash)
	}
	return newBlockV11FromHeader(dbase, header, patches, normalTxs)
}

func newBlockV13FromHeader(
	dbase db.Database,
	header *headerFormat,
	patches module.TransactionList,
	normalTxs module.TransactionList,
	nextValidators module.ValidatorList,
	blockVotes *blockv0.BlockVoteList,
	leaderVotes *blockv0.LeaderVoteList,
) (*Block, error) {
	proposer, err := newProposer(header.Proposer)
	if err != nil {
		return nil, err
	}
	nextLeader, err := newProposer(header.NextLeader)
	if err != nil {
		return nil, err
	}
	var blk = &blockV13{
		Block: Block{
			height:             header.Height,
			timestamp:          header.Timestamp,
			proposer:           proposer,
			prevHash:           header.PrevHash,
			logsBloom:          txresult.NewLogsBloomFromCompressed(header.LogsBloom),
			result:             header.Result,
			signature:          header.Signature,
			prevID:             header.PrevID,
			versionV0:          header.VersionV0,
			patchTransactions:  patches,
			normalTransactions: normalTxs,
		},
		nextValidatorsHash: header.NextValidatorsHash,
		stateHashV0:        header.StateHashV0,
		receiptsRoot:       header.ReceiptRoot,
		repsRoot:           header.RepsRoot,
		nextRepsRoot:       header.NextRepsRoot,
		logsBloomV0:        txresult.NewLogsBloomFromCompressed(header.LogsBloomV0),
		nextLeader:         nextLeader,
		_nextValidators:    nextValidators,
		blockVotes:         blockVotes,
		leaderVotes:        leaderVotes,
	}
	blk.Block.blockDetail = blk
	return &blk.Block, nil
}

func newBlockV13FromBlockFormat(dbase db.Database, format *format) (*Block, error) {
	patches := transaction.NewTransactionListFromHash(dbase, format.PatchTransactionsHash)
	if patches == nil {
		return nil, errors.Errorf("TranscationListFromHash(%x) failed", format.PatchTransactionsHash)
	}
	normalTxs := transaction.NewTransactionListFromHash(dbase, format.NormalTransactionsHash)
	if normalTxs == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", format.NormalTransactionsHash)
	}
	return newBlockV13FromHeader(
		dbase,
		&format.headerFormat,
		patches,
		normalTxs,
		nil,
		format.BlockVotes,
		format.LeaderVotes,
	)
}

func newBlockV13FromHeaderFormat(dbase db.Database, header *headerFormat) (*Block, error) {
	patches := transaction.NewTransactionListFromHash(dbase, header.PatchTransactionsHash)
	if patches == nil {
		return nil, errors.Errorf("TranscationListFromHash(%x) failed", header.PatchTransactionsHash)
	}
	normalTxs := transaction.NewTransactionListFromHash(dbase, header.NormalTransactionsHash)
	if normalTxs == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", header.NormalTransactionsHash)
	}
	nextValidators, err := state.ValidatorSnapshotFromHash(dbase, header.NextValidatorsHash)
	if err != nil {
		return nil, err
	}
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	bs, err := bk.Get(header.BlockVotesHash)
	if err != nil {
		return nil, err
	}
	if bs == nil {
		return nil, errors.NotFoundError.New("block vote not found")
	}
	var blockVotes blockv0.BlockVoteList
	_, err = codec.BC.UnmarshalFromBytes(bs, &blockVotes)
	if err != nil {
		return nil, err
	}
	bs, err = bk.Get(header.LeaderVotesHash)
	if err != nil {
		return nil, err
	}
	if bs == nil {
		return nil, errors.NotFoundError.New("block vote not found")
	}
	var leaderVotes blockv0.LeaderVoteList
	_, err = codec.BC.UnmarshalFromBytes(bs, &leaderVotes)
	if err != nil {
		return nil, err
	}
	return newBlockV13FromHeader(
		dbase,
		header,
		patches,
		normalTxs,
		nextValidators,
		&blockVotes,
		&leaderVotes,
	)
}

func NewBlockFromHeaderReader(database db.Database, r io.Reader) (*Block, error) {
	var header headerFormat
	err := codec.BC.Unmarshal(r, &header)
	if err != nil {
		return nil, err
	}
	switch header.VersionV0 {
	case "0.1a":
		return newBlockV11FromHeaderFormat(database, &header)
	case "0.3", "0.4":
		return newBlockV13FromHeaderFormat(database, &header)
	}
	return nil, errors.UnsupportedError.Errorf("block version %s", header.VersionV0)
}

func NewBlockFromReader(dbase db.Database, r io.Reader) (*Block, error) {
	var blockFormat format
	err := codec.BC.Unmarshal(r, &blockFormat.headerFormat)
	if err != nil {
		return nil, err
	}
	err = codec.BC.Unmarshal(r, &blockFormat.bodyFormat)
	if err != nil {
		return nil, err
	}
	switch blockFormat.VersionV0 {
	case "0.1a":
		return newBlockV11FromBlockFormat(dbase, &blockFormat)
	case "0.3", "0.4":
		return newBlockV13FromBlockFormat(dbase, &blockFormat)
	}
	return nil, errors.UnsupportedError.Errorf("block version %s", blockFormat.VersionV0)
}

func NewFromV0(
	blkV0 blockv0.Block,
	dbase db.Database,
	prevHash []byte,
	tr module.Transition,
) (*Block, error) {
	txs := blkV0.NormalTransactions()
	switch blk := blkV0.(type) {
	case *blockv0.BlockV01a:
		blkV1 := &blockV11{
			Block: Block{
				height:             blk.Height(),
				timestamp:          blk.Timestamp(),
				proposer:           blk.Proposer(),
				prevHash:           prevHash,
				logsBloom:          tr.LogsBloom(),
				result:             tr.Result(),
				signature:          blk.Signature,
				prevID:             blk.PrevID(),
				versionV0:          blk.Version(),
				patchTransactions:  transaction.NewTransactionListFromSlice(dbase, nil),
				normalTransactions: transaction.NewTransactionListFromSlice(dbase, txs),
			},
		}
		blkV1.blockDetail = blkV1
		return &blkV1.Block, nil
	case *blockv0.BlockV03:
		nl := blk.NextLeader()
		blkV1 := &blockV13{
			Block: Block{
				height:             blk.Height(),
				timestamp:          blk.Timestamp(),
				proposer:           blk.Proposer(),
				prevHash:           prevHash,
				logsBloom:          tr.LogsBloom(),
				result:             tr.Result(),
				signature:          blk.Signature(),
				prevID:             blk.PrevID(),
				versionV0:          blk.Version(),
				patchTransactions:  transaction.NewTransactionListFromSlice(dbase, nil),
				normalTransactions: transaction.NewTransactionListFromSlice(dbase, txs),
			},
			nextValidatorsHash: tr.NextValidators().Hash(),
			stateHashV0:        blk.StateHash(),
			receiptsRoot:       blk.ReceiptsHash(),
			repsRoot:           blk.RepsHash(),
			nextRepsRoot:       blk.NextRepsHash(),
			logsBloomV0:        blk.LogsBloom(),
			nextLeader:         &nl,
			_nextValidators:    tr.NextValidators(),
			_receipts:          tr.NormalReceipts(),
			blockVotes:         blk.PrevVotes(),
			leaderVotes:        blk.LeaderVotes(),
		}
		blkV1.blockDetail = blkV1
		return &blkV1.Block, nil
	}
	return nil, errors.UnsupportedError.Errorf("Unknown block type %s", blkV0.Version())
}

func (b *Block) WriteTo(dbase db.Database) error {
	bk, err := db.NewCodedBucket(dbase, db.BytesByHash, nil)
	if err != nil {
		return err
	}
	if err = bk.Put(b.headerFormat()); err != nil {
		return err
	}
	if b.BlockVotes() != nil {
		if err = bk.Put(b.BlockVotes()); err != nil {
			return err
		}
	}
	if b.LeaderVotes() != nil {
		if err = bk.Put(b.LeaderVotes()); err != nil {
			return err
		}
	}
	bk, err = db.NewCodedBucket(dbase, db.BlockHeaderHashByHeight, nil)
	if err != nil {
		return err
	}
	if err = bk.Set(b.Height(), db.Raw(b.Hash())); err != nil {
		return err
	}
	ibk, err := dbase.GetBucket(icdb.IDToHash)
	if err != nil {
		return err
	}
	if err = ibk.Set(b.ID(), b.Hash()); err != nil {
		return err
	}
	if err = block.WriteTransactionLocators(
		dbase,
		b.Height(),
		b.patchTransactions,
		b.normalTransactions,
	); err != nil {
		return err
	}
	return nil
}
