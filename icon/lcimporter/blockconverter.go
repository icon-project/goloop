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
	"path"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/sync"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	ContractPath = "contract"
	EESocketPath = "ee.sock"
)

const (
	KeyLastBlockHeight = "block.lastHeight"
)

type BlockConverter struct {
	baseDir  string
	cs       Store
	database db.Database
	cm       contract.ContractManager
	em       eeproxy.Manager
	chain    module.Chain
	log      log.Logger
	plt      service.Platform
	trace    log.Logger

	sHeight int64

	blkIndex    db.Bucket
	blkByHash   db.Bucket
	chainBucket db.Bucket

	syncMan service.SyncManager

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

func NewBlockConverter(chain module.Chain, plt service.Platform, cs Store, data string) (*BlockConverter, error) {
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
	var syncMan service.SyncManager
	if chain.NetworkManager() != nil {
		syncMan = sync.NewSyncManager(database, chain.NetworkManager(), plt, logger)
		if syncMan != nil {
			return nil, errors.Errorf("Fail to create sync manager")
		}
	}
	ex := &BlockConverter{
		baseDir:     data,
		cs:          cs,
		log:         logger,
		chain:       chain,
		plt:         plt,
		blkIndex:    blkIndex,
		blkByHash:   blkByHash,
		chainBucket: chainBucket,
		syncMan:     syncMan,
	}
	ex.trace = logger.WithFields(log.Fields{
		log.FieldKeyModule: "TRACE",
	})
	ex.database = database
	return ex, nil
}

const ChanBuf = 2048

func (e *BlockConverter) Start(from, to int64) (<-chan interface{}, error) {
	return e.execute(from, to, nil)
}

// Rebase re-bases blocks and returns a channel of
// *BlockTransaction or error.
func (e *BlockConverter) Rebase(from, to int64, firstNForcedResults []*BlockTransaction) (<-chan interface{}, error) {
	return e.execute(from, to, firstNForcedResults)
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
	if e.em == nil || e.cm == nil {
		if err := e.setupEE(); err != nil {
			return nil, err
		}
	}
	logger := trace.NewLogger(e.log, e)
	if height > 0 {
		blk, err := e.GetBlockByHeight(height)
		blkV0, err := e.cs.GetBlockByHeight(int(height))
		if err != nil {
			return nil, errors.Wrapf(err, "NoLastState(height=%d)", height)
		}
		tsc := service.NewTimestampChecker()
		tr, err := service.NewInitTransition(
			e.database,
			blk.Result(),
			nil,
			e.cm,
			e.em,
			e.chain,
			logger,
			e.plt,
			tsc,
		)
		if err != nil {
			return nil, err
		}
		return &Transition{tr, blkV0, nil, blk.Hash()}, nil
	} else {
		tsc := service.NewTimestampChecker()
		tr, err := service.NewInitTransition(
			e.database,
			nil,
			nil,
			e.cm,
			e.em,
			e.chain,
			logger,
			e.plt,
			tsc,
		)
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
	if height == 0 {
		csi = common.NewConsensusInfo(nil, nil, nil)
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
		csi = common.NewConsensusInfo(blkv0.Proposer(), voters, voted)
	}
	tr := service.NewTransition(
		last.Transition,
		nil,
		transaction.NewTransactionListFromSlice(e.database, blkv0.NormalTransactions()),
		common.NewBlockInfo(height, blkv0.Timestamp()),
		csi,
		true,
	)
	return &Transition{tr, blkv0, nil, nil}, nil
}

func (e *BlockConverter) getLastHeight() int64 {
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
	if eLogBloom != nil && !rLogBloom.Equal(eLogBloom) {
		return errors.Errorf("InvalidLogBloom(exp=%x,res=%x)",
			eLogBloom.LogBytes(), rLogBloom.LogBytes())
	}
	return nil
}

func TimestampToString(ts int64) string {
	tm := time.Unix(ts/1000000, (ts%1000000)*1000)
	return tm.Format("2006-01-02 15:04:05")
}

func (e *BlockConverter) setupEE() error {
	e.database = cache.AttachManager(e.database, path.Join(e.baseDir, "cache"), 5, 0)
	cm, err := e.plt.NewContractManager(e.database, path.Join(e.baseDir, ContractPath), e.log)
	if err != nil {
		return errors.Wrap(err, "NewContractManagerFailure")
	}
	ee, err := eeproxy.AllocEngines(e.log, "python")
	if err != nil {
		return errors.Wrap(err, "FailureInAllocEngines")
	}
	em, err := eeproxy.NewManager("unix", path.Join(e.baseDir, EESocketPath), e.log, ee...)
	if err != nil {
		return errors.Wrap(err, "FailureInAllocProxyManager")
	}

	go em.Loop()
	em.SetInstances(1, 1, 1)

	e.cm = cm
	e.em = em
	return nil
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
	if e.resCh != nil {
		e.stopCh <- struct{}{}
		// unblocks writer
		switch e.resCh {
		case <-e.resCh:
		default:
		}
	}
	resCh := make(chan interface{}, ChanBuf)
	e.resCh = resCh
	stopCh := make(chan struct{}, 1)
	e.stopCh = stopCh
	go func() {
		err := e.doExecute(from, to, firstNForcedResults, resCh, stopCh)
		if err != nil {
			resCh <- err
		}
		close(resCh)
	}()
	return resCh, nil
}

func (e *BlockConverter) doExecute(
	from, to int64,
	firstNForcedResults []*BlockTransaction,
	resCh chan<- interface{},
	stopCh <-chan struct{},
) error {
	Statusf(e.log, "Executing Blocks from=%d, to=%d", from, to)
	if from < 0 {
		from = e.getLastHeight() + 1
	}
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
		case <- stopCh:
			return nil
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
		Statusf(
			e.log,
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
			tr.Transition = service.NewSyncTransition(tr, e.syncMan, fr.Result, fr.ValidatorHash)
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
		blk, err := blockv1.NewFromV0(tr.block, e.database, prevTR.blockHash, prevTR)
		if err != nil {
			return err
		}
		tr.blockHash = blk.Hash()
		if err = blk.WriteTo(e.database); err != nil {
			return err
		}
		// TODO pile block merkle tree
		if err = service.FinalizeTransition(tr.Transition,
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
