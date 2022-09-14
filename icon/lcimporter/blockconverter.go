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

package lcimporter

import (
	"bytes"
	"fmt"
	"time"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	KeyLastBlockHeight = "block.lastHeight"
	ChanBuf            = 2048
)

var ErrAfterLastBlock = errors.NewBase(errors.IllegalArgumentError, "AfterLastBlock")

type BlockConverter struct {
	baseDir  string
	cs       Store
	database db.Database
	log      log.Logger
	trace    log.Logger

	blkIndex    db.Bucket
	blkByHash   db.Bucket
	chainBucket db.Bucket
	svc         Service

	stopCh chan<- struct{}
	resCh  <-chan interface{}

	module.TraceCallback
}

type Transition struct {
	module.Transition
	block       blockv0.Block
	prevBlock   blockv0.Block
	oldReceipts module.ReceiptList
	blockHash   []byte
}

type Store interface {
	GetRepsByHash(id []byte) (*blockv0.RepsList, error)
	GetBlockByHeight(height int) (blockv0.Block, error)
	GetReceipt(id []byte) (module.Receipt, error)
	GetVotesByHeight(h int) (*blockv0.BlockVoteList, error)
}

type GetTPSer interface {
	GetTPS() float32
}

func NewBlockConverter(c module.Chain, plt base.Platform, pm eeproxy.Manager, cs Store, data string) (*BlockConverter, error) {
	svc, err := NewService(c, plt, pm, data)
	if err != nil {
		return nil, err
	}
	return NewBlockConverterWithService(c, plt, cs, data, svc)
}

func NewBlockConverterWithService(
	chain module.Chain,
	plt base.Platform,
	cs Store,
	data string,
	svc Service,
) (*BlockConverter, error) {
	database := chain.Database()
	logger := chain.Logger()
	blkIndex, err := database.GetBucket(db.BlockHeaderHashByHeight)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=HashByHeight)")
	}
	blkByHash, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=BlockV1ByID)")
	}
	chainBucket, err := database.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}
	ex := &BlockConverter{
		baseDir:     data,
		cs:          cs,
		log:         logger,
		blkIndex:    blkIndex,
		blkByHash:   blkByHash,
		chainBucket: chainBucket,
		svc:         svc,
	}
	ex.trace = logger.WithFields(log.Fields{
		log.FieldKeyModule: "TRACE",
	})
	ex.database = database
	return ex, nil
}

func (e *BlockConverter) Start(from, to int64) (<-chan interface{}, error) {
	return e.execute(from, to, nil)
}

// Rebase re-bases blocks and returns a channel of
// *BlockTransaction or error. ErrAfterLastBlock is sent if from is beyond
// the last block.
func (e *BlockConverter) Rebase(from, to int64, firstNForcedResults []*BlockTransaction) (<-chan interface{}, error) {
	return e.execute(from, to, firstNForcedResults)
}

func (e *BlockConverter) Term() {
	if e.resCh != nil {
		e.stopCh <- struct{}{}
		// unblocks writer and wait for close
		for _ = range e.resCh {
		}
		e.resCh = nil
		e.stopCh = nil
	}
}

func BlockIndexKey(height int64) []byte {
	return codec.BC.MustMarshalToBytes(height)
}

func (e *BlockConverter) GetBlockByHeight(h int64) (*blockv1.Block, error) {
	hash, err := e.blkIndex.Get(BlockIndexKey(h))
	if err != nil {
		return nil, err
	}
	if len(hash) > 0 {
		bs, err := e.blkByHash.Get(hash)
		if err != nil {
			return nil, err
		}
		blk, err := blockv1.NewBlockFromHeaderReader(e.database,
			bytes.NewReader(bs))
		if err != nil {
			return nil, err
		}
		return blk, nil
	}
	return nil, nil
}

func (e *BlockConverter) OnLog(level module.TraceLevel, msg string) {
	switch level {
	case module.TSystemLevel:
		e.trace.Trace(msg)
	default:
		// others are already printed by logger
	}
}

func (e *BlockConverter) OnEnd(err error) {
	e.trace.Tracef("Result=%+v ", err)
}

func (e *BlockConverter) initTransitionFor(height int64) (*Transition, error) {
	if height < 0 {
		return nil, errors.Errorf("InvalidHeight(height=%d)", height)
	}
	ti := &module.TraceInfo{
		Callback: e,
	}
	logger := trace.NewLogger(e.log, ti)
	if height > 0 {
		var prevV0 blockv0.Block
		var err error
		if height > 1 {
			prevV0, err = e.cs.GetBlockByHeight(int(height - 2))
			if err != nil {
				return nil, errors.Wrapf(err, "NoLastState(height=%d)", height)
			}
		}
		blk, err := e.GetBlockByHeight(height - 1)
		blkV0, err := e.cs.GetBlockByHeight(int(height - 1))
		if err != nil {
			return nil, errors.Wrapf(err, "NoLastState(height=%d)", height)
		}
		tr, err := e.svc.NewInitTransition(blk.Result(), blk.NextValidators(), logger)
		if err != nil {
			return nil, err
		}
		return &Transition{tr, blkV0, prevV0, nil, blk.Hash()}, nil
	} else {
		tr, err := e.svc.NewInitTransition(nil, nil, logger)
		if err != nil {
			return nil, err
		} else {
			return &Transition{tr, nil, nil, nil, nil}, nil
		}
	}
}

func (e *BlockConverter) originalReceipts(txs []module.Transaction) ([]txresult.Receipt, error) {
	rcts := make([]txresult.Receipt, len(txs))
	for idx, tx := range txs {
		if err := tx.Verify(); err != nil {
			return nil, err
		}
		rct, err := e.cs.GetReceipt(tx.ID())
		if err != nil {
			return nil, errors.Wrapf(err, "FailureInGetReceipts(txid=%#x)", tx.ID())
		}
		rcts[idx] = rct.(txresult.Receipt)
	}
	return rcts, nil
}

func (e *BlockConverter) newConsensusInfoForTxsInBlock(
	blk blockv0.Block,
	prevBlk blockv0.Block,
) (module.ConsensusInfo, error) {
	var csi module.ConsensusInfo
	if prevBlk == nil {
		csi = common.NewConsensusInfo(nil, nil, nil)
	} else {
		var voters module.ValidatorList
		var err error
		var voted []bool
		if prevV03, ok := prevBlk.(*blockv0.BlockV03); ok {
			voters, err = prevV03.Validators().GetValidatorList(e.database)
			if err != nil {
				return nil, err
			}
			voted = make([]bool, voters.Len())
			err = blk.(*blockv0.BlockV03).PrevVotes().CheckVoters(prevV03.Validators(), voted)
			if err != nil {
				return nil, err
			}
		}
		csi = common.NewConsensusInfo(prevBlk.Proposer(), voters, voted)
	}
	return csi, nil
}

func (e *BlockConverter) proposeTransition(last *Transition) (*Transition, error) {
	var height int64
	if last.block != nil {
		height = last.block.Height() + 1
	} else {
		height = 0
	}
	prevV0 := last.block
	blkv0, err := e.cs.GetBlockByHeight(int(height))
	if err != nil {
		return nil, err
	}
	if err := blkv0.Verify(last.block); err != nil {
		return nil, err
	}
	var rcts []txresult.Receipt
	if last.block != nil {
		txs := last.block.NormalTransactions()
		rcts, err = e.originalReceipts(txs)
		if err != nil {
			return nil, err
		}
		if lastV03, ok := last.block.(*blockv0.BlockV03); ok {
			eReceiptListHash := lastV03.ReceiptsHash()
			rReceiptListHash := blockv0.CalcMerkleRootOfReceiptSlice(rcts, txs, lastV03.Height())
			if !bytes.Equal(eReceiptListHash, rReceiptListHash) {
				return nil, errors.Errorf("DifferentReceiptListHash(stored=%#x,real=%#x)",
					eReceiptListHash, rReceiptListHash)
			}
		}
	}
	csi, err := e.newConsensusInfoForTxsInBlock(last.block, last.prevBlock)
	if err != nil {
		return nil, err
	}
	var tr module.Transition
	if height == 0 {
		tr = e.svc.NewTransition(
			last.Transition,
			nil,
			transaction.NewTransactionListFromSlice(e.database, nil),
			blkv0, // to avoid nil access
			csi,
			true,
		)
	} else {
		tr = e.svc.NewTransition(
			last.Transition,
			nil,
			transaction.NewTransactionListFromSlice(e.database, last.block.NormalTransactions()),
			last.block,
			csi,
			true,
		)
	}
	rctList := txresult.NewReceiptListFromSlice(e.database, rcts)
	return &Transition{tr, blkv0, prevV0, rctList, nil}, nil
}

func (e *BlockConverter) GetLastHeight() int64 {
	bs, err := e.chainBucket.Get([]byte(KeyLastBlockHeight))
	if err != nil || len(bs) == 0 {
		e.log.Warn("Fail to get last block height")
		return -1
	}
	var height int64
	if _, err := codec.BC.UnmarshalFromBytes(bs, &height); err != nil {
		e.log.Error("Fail to parse last block height")
		return -1
	} else {
		e.log.Debugf("Last block height:%d", height)
		return height
	}
}

type transitionCallback chan error

func (cb transitionCallback) OnValidate(transition module.Transition, err error) {
	cb <- err
}

func (cb transitionCallback) OnExecute(transition module.Transition, err error) {
	cb <- err
}

func (e *BlockConverter) checkResult(tr *Transition) error {
	results := tr.NormalReceipts()
	expects := tr.oldReceipts
	idx := 0
	if !bytes.Equal(expects.Hash(), results.Hash()) {
		for expect, result := expects.Iterator(), results.Iterator(); expect.Has() || result.Has(); _, _, idx = expect.Next(), result.Next(), idx+1 {
			rct1, err := expect.Get()
			if err != nil {
				return errors.Wrapf(err, "ExpectReceiptGetFailure(idx=%d)", idx)
			}
			rct2, err := result.Get()
			if err != nil {
				return errors.Wrapf(err, "ResultReceiptGetFailure(idx=%d)", idx)
			}
			if err := CheckReceipt(e.log, rct1, rct2); err != nil {
				rct1js, _ := JSONMarshalIndent(rct1)
				rct2js, _ := JSONMarshalIndent(rct2)

				tx, _ := tr.Transition.NormalTransactions().Get(idx)
				txjs, _ := JSONMarshalIndent(tx)

				e.log.Errorf("Failed Block[ %9d ] TxID[ %#x ]", tr.block.Height(), tx.ID())
				e.log.Errorf("Failed Transaction[%d]:%s", idx, txjs)
				e.log.Errorf("Expected Receipt[%d]:%s", idx, rct1js)
				e.log.Errorf("Returned Receipt[%d]:%s", idx, rct2js)
				return errors.Wrapf(err, "ReceiptComparisonFailure(idx=%d)", idx)
			}
		}
	}

	if tr.prevBlock == nil {
		return nil
	}

	rLogBloom := tr.Transition.LogsBloom()
	eLogBloom := tr.prevBlock.LogsBloom()
	if err := CheckLogsBloom(e.log, eLogBloom, rLogBloom); err != nil {
		e.log.Errorf("Failed Block[ %9d ] LogBloomError err=%+v", err)
		return err
	}

	if reps := tr.prevBlock.NextValidators(); reps != nil {
		rs := reps.Size()
		validators := tr.Transition.NextValidators()
		vs := validators.Len()
		if vs > 0 {
			if vs != rs {
				return errors.Errorf("InvalidValidatorLen(exp=%d,calc=%d)", rs, vs)
			}
			for i := 0; i < rs; i++ {
				rep := reps.Get(i)
				val, _ := validators.Get(i)
				if !rep.Equal(val.Address()) {
					return errors.Errorf("InvalidValidator(idx=%d,exp=%s,calc=%s)",
						i, rep.String(), val.Address().String())
				}
			}
		}
	}
	return nil
}

func TimestampToString(ts int64) string {
	tm := time.Unix(ts/1000000, (ts%1000000)*1000)
	return tm.Format("2006-01-02 15:04:05")
}

var diskSpin = []string{"⠁", "⠁", "⠉", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠤", "⠄", "⠄", "⠤", "⠠", "⠠", "⠤", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋", "⠉", "⠈", "⠈"}
var netSpin = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinner(height, stored int64) string {
	if stored > height {
		return diskSpin[int(height/4)%(len(diskSpin))]
	} else {
		return netSpin[int(height)%(len(netSpin))]
	}
}

func D(v interface{}) string {
	var ret string
	vs := fmt.Sprintf("%d", v)
	vLen := len(vs)
	for vLen > 0 {
		seg := (vLen-1)%3 + 1
		if len(ret) > 0 {
			ret += ","
		}
		ret += vs[0:seg]
		vs = vs[seg:]
		vLen -= seg
	}
	return ret
}

func (e *BlockConverter) execute(from, to int64, firstNForcedResults []*BlockTransaction) (<-chan interface{}, error) {
	e.Term()
	resCh := make(chan interface{}, ChanBuf)
	e.resCh = resCh
	stopCh := make(chan struct{}, 1)
	e.stopCh = stopCh
	go func() {
		defer close(resCh)
		e.log.Infof("Executing Blocks from=%d, to=%d", from, to)
		last := e.GetLastHeight()
		if from < 0 {
			from = last + 1
		}
		if last > 0 && len(firstNForcedResults) == 0 {
			if last < to {
				last = to
			}
			for i := from; i <= last; i++ {
				blk, err := e.GetBlockByHeight(i)
				if err != nil {
					resCh <- err
					return
				}
				ltxs := blk.NormalTransactions()
				txCount := 0
				for itr := ltxs.Iterator(); itr.Has(); itr.Next() {
					txCount += 1
				}
				resCh <- &BlockTransaction{
					Height:        blk.Height(),
					BlockHash:     blk.Hash(),
					Result:        blk.Result(),
					ValidatorHash: blk.NextValidatorsHash(),
					TXCount:       int32(txCount),
				}
			}
			from = last + 1
		}
		err := e.doExecute(from, to, firstNForcedResults, resCh, stopCh)
		if err != nil {
			resCh <- err
		}
	}()
	return resCh, nil
}

func check(bsv1 []byte, bsv0 []byte, name string) (bool, string) {
	if !bytes.Equal(bsv1, bsv0) {
		return false, fmt.Sprintf("%s is different\n\tv1:%v\n\tv0:%v\n", name, bsv1, bsv0)
	}
	return true, ""
}

func checkBlock(v1 *blockv1.Block, v0 blockv0.Block) error {
	if !bytes.Equal(v1.ID(), v0.ID()) {
		var msg string
		if ok, m := check(v1.PrevID(), v0.PrevID(), "PrevID"); !ok {
			msg += m
		}
		if ok, m := check(v1.TransactionsRoot(), v0.TransactionRoot(), "TransactionRoot"); !ok {
			msg += m
		}
		if ok, m := check(common.BytesOfAddress(v1.Proposer()), common.BytesOfAddress(v0.Proposer()), "Proposer"); !ok {
			msg += m
		}
		return errors.Errorf("block ID is different\n\tv1:%v\n\tv0:%v\n%s", v1.ID(), v0.ID(), msg)
	}
	return nil
}

func (e *BlockConverter) doExecute(
	from, to int64,
	firstNForcedResults []*BlockTransaction,
	resCh chan<- interface{},
	stopCh <-chan struct{},
) error {
	getTPSer, _ := e.cs.(GetTPSer)
	if to >= 0 && to < from {
		return errors.IllegalArgumentError.Errorf("InvalidArgument(from=%d,to=%d)", from, to)
	}
	prevTR, err := e.initTransitionFor(from)
	if err != nil {
		return err
	}
	nbv := e.svc.GetNextBlockVersion(prevTR.Result(), prevTR.NextValidators())
	if nbv == module.BlockVersion2 {
		return ErrAfterLastBlock
	}
	callback := make(transitionCallback, 1)
	var rps, tps float32
	tm := new(TPSMeasure).Init(100)
	forcedEnd := from + int64(len(firstNForcedResults))
	for height := from; to < 0 || height <= to; height = height + 1 {
		select {
		case <-stopCh:
			return errors.InterruptedError.Errorf("Execution interrupted")
		default:
		}
		if getTPSer != nil {
			rps = getTPSer.GetTPS()
		}
		tps = tm.GetTPS()
		var ts int64
		if prevTR.block != nil {
			ts = prevTR.block.Timestamp()
		}
		e.log.Infof(
			"[%s] Executing Block[ %10s ] %s RPS[ %6.2f ] TPS[ %6.2f ]",
			spinner(height, height),
			D(height),
			TimestampToString(ts),
			rps,
			tps,
		)
		tr, err := e.proposeTransition(prevTR)
		if err != nil {
			return errors.Wrapf(err, "FailureInPropose(height=%d)", height)
		}
		if height < forcedEnd {
			fr := firstNForcedResults[height-from]
			tr.Transition = e.svc.NewSyncTransition(tr, fr.Result, fr.ValidatorHash)
		}
		if _, err = tr.Execute(callback); err != nil {
			return errors.Wrapf(err, "FailureInExecute(height=%d)", height)
		}
		err = <-callback
		if err != nil {
			return errors.Wrapf(err, "PreValidationFail(height=%d)", height)
		}
		err = <-callback
		if err != nil {
			return errors.Wrapf(err, "ExecutionFailure(height=%d)", height)
		}

		if err := e.checkResult(tr); err != nil {
			return err
		}

		e.log.Infof("Finalize Block[ %9d ]", height)
		blk, err := blockv1.NewFromV0(tr.block, e.database, prevTR.blockHash, tr)
		if err != nil {
			return err
		}
		if err := checkBlock(blk, tr.block); err != nil {
			return err
		}
		tr.blockHash = blk.Hash()
		if err = blk.WriteTo(e.database); err != nil {
			return err
		}
		if err = blk.NormalTransactions().Flush(); err != nil {
			return err
		}
		if err = e.svc.FinalizeTransition(tr.Transition,
			module.FinalizeResult,
			false,
		); err != nil {
			return errors.Wrapf(err, "FinalizationFailure(height=%d)", height)
		}
		bk, err := db.NewCodedBucket(e.database, db.ChainProperty, nil)
		if err != nil {
			return err
		}
		if err = bk.Set(db.Raw(KeyLastBlockHeight), blk.Height()); err != nil {
			return err
		}
		resCh <- &BlockTransaction{
			blk.Height(),
			blk.Hash(),
			blk.Result(),
			blk.NextValidatorsHash(),
			int32(len(tr.block.NormalTransactions())),
			nil,
		}
		nbv := e.svc.GetNextBlockVersion(tr.Result(), prevTR.NextValidators())
		if nbv == module.BlockVersion2 {
			break
		}
		prevTR = tr
	}
	return nil
}

func (e *BlockConverter) GetBlockVotes(h int64) (*blockv0.BlockVoteList, error) {
	return e.cs.GetVotesByHeight(int(h))
}
