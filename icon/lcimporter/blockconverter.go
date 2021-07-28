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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	KeyLastBlockHeight = "block.lastHeight"
	ChanBuf = 2048
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
}

type Transition struct {
	module.Transition
	block       blockv0.Block
	oldReceipts module.ReceiptList
	blockHash   []byte
}

type Store interface {
	GetRepsByHash(id []byte) (*blockv0.RepsList, error)
	GetBlockByHeight(height int) (blockv0.Block, error)
	GetReceipt(id []byte) (module.Receipt, error)
}

type GetTPSer interface {
	GetTPS() float32
}

func NewBlockConverter(c module.Chain, plt service.Platform, cs Store, data string) (*BlockConverter, error) {
	svc, err := NewService(c, plt, data)
	if err != nil {
		return nil, err
	}
	return NewBlockConverterWithService(c, plt, cs, data, svc)
}

func NewBlockConverterWithService(
	chain module.Chain,
	plt service.Platform,
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
// *BlockTransaction or error.
func (e *BlockConverter) Rebase(from, to int64, firstNForcedResults []*BlockTransaction) (<-chan interface{}, error) {
	return e.execute(from, to, firstNForcedResults)
}

func (e *BlockConverter) Term() {
	if e.resCh != nil {
		e.stopCh <- struct{}{}
		// unblocks writer
		switch e.resCh {
		case <-e.resCh:
		default:
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
	logger := trace.NewLogger(e.log, e)
	if height > 0 {
		blk, err := e.GetBlockByHeight(height - 1)
		blkV0, err := e.cs.GetBlockByHeight(int(height - 1))
		if err != nil {
			return nil, errors.Wrapf(err, "NoLastState(height=%d)", height)
		}
		tr, err := e.svc.NewInitTransition(blk.Result(), nil, logger)
		if err != nil {
			return nil, err
		}
		return &Transition{tr, blkV0, nil, blk.Hash()}, nil
	} else {
		tr, err := e.svc.NewInitTransition(nil, nil, logger)
		if err != nil {
			return nil, err
		} else {
			return &Transition{tr, nil, nil, nil}, nil
		}
	}
}

func (e *BlockConverter) proposeTransition(last *Transition) (*Transition, error) {
	var height int64
	if last.block != nil {
		height = last.block.Height() + 1
	} else {
		height = 0
	}
	blkv0, err := e.cs.GetBlockByHeight(int(height))
	// TODO handle EOF
	if err != nil {
		return nil, err
	}
	// TODO add old receipts
	var csi module.ConsensusInfo
	var tr module.Transition
	if height == 0 {
		csi = common.NewConsensusInfo(nil, nil, nil)
		tr = e.svc.NewTransition(
			last.Transition,
			nil,
			transaction.NewTransactionListFromSlice(e.database, nil),
			blkv0,
			csi,
			true,
		)
	} else {
		var voters module.ValidatorList
		var err error
		var voted []bool
		if prev, ok := last.block.(*blockv0.BlockV03); ok {
			voters, err = prev.Validators().GetValidatorList(e.database)
			if err != nil {
				return nil, err
			}
			voted = make([]bool, voters.Len())
			err = blkv0.(*blockv0.BlockV03).PrevVotes().CheckVoters(prev.Validators(), voted)
			if err != nil {
				return nil, err
			}
		}
		csi = common.NewConsensusInfo(last.block.Proposer(), voters, voted)
		tr = e.svc.NewTransition(
			last.Transition,
			nil,
			transaction.NewTransactionListFromSlice(e.database, last.block.NormalTransactions()),
			blkv0,
			csi,
			true,
		)
	}
	return &Transition{tr, blkv0, nil, nil}, nil
}

func (e *BlockConverter) GetLastHeight() int64 {
	bs, err := e.chainBucket.Get([]byte(KeyLastBlockHeight))
	if err != nil || len(bs) == 0 {
		e.log.Debugf("Fail to get last block height")
		return -1
	}
	var height int64
	if _, err := codec.BC.UnmarshalFromBytes(bs, &height); err != nil {
		e.log.Debugf("Fail to parse last block height")
		return -1
	} else {
		e.log.Tracef("Last block height:%d", height)
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
	if expects == nil {
		return nil
	}
	idx := 0
	if !bytes.Equal(expects.Hash(), results.Hash()) {
		for expect, result := expects.Iterator(), results.Iterator(); expect.Has() && result.Has(); _, _, idx = expect.Next(), result.Next(), idx+1 {
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
	rLogBloom := tr.Transition.LogsBloom()
	eLogBloom := tr.block.LogsBloom()
	if err := CheckLogsBloom(e.log, eLogBloom, rLogBloom); err != nil {
		return err
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
			for i := from; i<=last; i++ {
				blk, err := e.GetBlockByHeight(i)
				if err != nil {
					resCh <- err
					return
				}
				blkv0, err := e.cs.GetBlockByHeight(int(i))
				if err != nil {
					resCh <- err
					return
				}
				resCh <- &BlockTransaction{
					Height: blk.Height(),
					BlockID: blk.ID(),
					Result: blk.Result(),
					ValidatorHash: blk.NextValidatorsHash(),
					TXCount: int32(len(blkv0.NormalTransactions())),
				}
			}
		}
		err := e.doExecute(from, to, firstNForcedResults, resCh, stopCh)
		if err != nil {
			resCh <- err
		}
	}()
	return resCh, nil
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
		tr.blockHash = blk.Hash()
		if err = blk.WriteTo(e.database); err != nil {
			return err
		}
		if err = e.svc.FinalizeTransition(tr.Transition,
			module.FinalizeNormalTransaction|module.FinalizeResult,
			false,
		); err != nil {
			return errors.Wrapf(err, "FinalizationFailure(height=%d)", height)
		}
		bk, err := db.NewCodedBucket(e.database, db.ChainProperty, nil)
		if err != nil {
			return err
		}
		if err = bk.Set(KeyLastBlockHeight, blk.Height()); err != nil {
			return err
		}
		resCh <- &BlockTransaction{
			blk.Height(),
			blk.ID(),
			blk.Result(),
			blk.NextValidatorsHash(),
			int32(len(tr.block.NormalTransactions())),
			nil,
		}
		prevTR = tr
	}
	return nil
}
