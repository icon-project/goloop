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

package main

import (
	"bytes"
	"encoding/json"
	"math/big"
	"path"
	"reflect"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	ContractPath = "contract"
	EESocketPath = "ee.sock"
)
const (
	FlagExecutor = "executor"
)

const (
	KeyLastBlockHeight   = "block.lastHeight"
	KeyStoredBlockHeight = "block.storedHeight"
)

const (
	JSONByHash db.BucketID = "J"
)

// executeTransactions executes transactions from lc and confirm results.
// then it stores actual results.
// If from is negative, it executes from
type Executor struct {
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

	jsBucket    db.Bucket
	blkIndex    db.Bucket
	blkByID     db.Bucket
	chainBucket db.Bucket
}

type Transition struct {
	module.Transition
	Block *Block
}

type Store interface {
	GetRepsByHash(id []byte) (*blockv0.RepsList, error)
	GetBlockByHeight(height int) (blockv0.Block, error)
	GetReceipt(id []byte) (module.Receipt, error)
}

func NewExecutor(logger log.Logger, cs Store, data string) (*Executor, error) {
	database, err := db.Open(data, "goleveldb", "database")
	if err != nil {
		return nil, errors.Wrapf(err, "DatabaseFailure(path=%s)", data)
	}
	chain, err := newChain(database, logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewChainFailure")
	}
	plt, err := icon.NewPlatform(data, chain.CID())
	if err != nil {
		return nil, errors.Wrap(err, "NewPlatformFailure")
	}
	jsBucket, err := database.GetBucket(JSONByHash)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInGetBucketForJSON")
	}
	blkIndex, err := database.GetBucket(db.BlockHeaderHashByHeight)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=HashByHeight)")
	}
	blkByID, err := database.GetBucket(db.BlockV1ByHash)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=BlockV1ByHash)")
	}
	chainBucket, err := database.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}
	ex := &Executor{
		baseDir:     data,
		cs:          cs,
		log:         logger,
		chain:       chain,
		plt:         plt,
		jsBucket:    jsBucket,
		blkIndex:    blkIndex,
		blkByID:     blkByID,
		chainBucket: chainBucket,
	}
	ex.trace = logger.WithFields(log.Fields{
		log.FieldKeyModule: "TRACE",
	})
	ex.database = db.WithFlags(database, db.Flags{
		FlagExecutor: ex,
	})
	return ex, nil
}

func (e *Executor) GetRepsByHash(hash []byte) (*blockv0.RepsList, error) {
	if js, err := e.jsBucket.Get(hash); err != nil || js == nil {
		return e.cs.GetRepsByHash(hash)
	} else {
		reps := new(blockv0.RepsList)
		if err := json.Unmarshal(js, reps); err != nil {
			return nil, err
		}
		return reps, nil
	}
}

func BlockIndexKey(height int64) []byte {
	return codec.BC.MustMarshalToBytes(height)
}

func (e *Executor) GetBlockByHeight(h int64) (*Block, error) {
	hash, err := e.blkIndex.Get(BlockIndexKey(h))
	if err != nil {
		return nil, err
	}
	if len(hash) > 0 {
		bs, err := e.blkByID.Get(hash)
		if err != nil {
			return nil, err
		}
		blk := new(Block)
		if err := blk.Reset(e.database, bs); err != nil {
			return nil, err
		}
		return blk, nil
	}
	return nil, nil
}

func (e *Executor) NewWorldSnapshot(height int64) (state.WorldSnapshot, error) {
	blk, err := e.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	return blk.NewWorldSnapshot(e.database, e.plt)
}

func (e *Executor) OnLog(level module.TraceLevel, msg string) {
	switch level {
	case module.TSystemLevel:
		e.trace.Trace(msg)
	default:
		// others are already printed by logger
	}
}

func (e *Executor) OnEnd(err error) {
	e.trace.Tracef("Result=%+v ", err)
}

func (e *Executor) InitTransitionFor(height int64) (*Transition, error) {
	if height < 0 {
		return nil, errors.Errorf("InvalidHeight(height=%d)", height)
	}
	if e.em == nil || e.cm == nil {
		if err := e.SetupEE(); err != nil {
			return nil, err
		}
	}
	logger := trace.NewLogger(e.log, e)
	if height > 0 {
		blk, err := e.GetBlockByHeight(height - 1)
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
		return &Transition{tr, blk}, nil
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
			return &Transition{tr, nil}, nil
		}
	}
}

func (e *Executor) ProposeTransition(last *Transition, noCache bool) (*Transition, error) {
	var height int64
	if last.Block != nil {
		height = last.Block.Height() + 1
	} else {
		height = 0
	}
	blk, err := e.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	if blk == nil || noCache {
		if blk, err = e.LoadBlockByHeight(last.Block, height); err != nil {
			return nil, err
		}
	}
	var csi module.ConsensusInfo
	if height == 0 {
		csi = common.NewConsensusInfo(nil, nil, nil)
	} else {
		// TODO need to fill up consensus information
		csi = common.NewConsensusInfo(nil, nil, nil)
	}
	tr := service.NewTransition(
		last.Transition,
		nil,
		blk.Transactions(),
		common.NewBlockInfo(height, blk.Timestamp()),
		csi,
		e.plt,
		true,
	)
	return &Transition{tr, blk}, nil
}

func (e *Executor) setLastHeight(height int64) error {
	e.log.Tracef("setLastHeight(%d)", height)
	return e.chainBucket.Set(
		[]byte(KeyLastBlockHeight),
		codec.BC.MustMarshalToBytes(height),
	)
}

func (e *Executor) getLastHeight() int64 {
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

func (e *Executor) setStoredHeight(height int64) error {
	e.log.Tracef("setStoredHeight(%d)", height)
	return e.chainBucket.Set(
		[]byte(KeyStoredBlockHeight),
		codec.BC.MustMarshalToBytes(height),
	)
}

func (e *Executor) getStoredHeight() int64 {
	bs, err := e.chainBucket.Get([]byte(KeyStoredBlockHeight))
	if err != nil || len(bs) == 0 {
		e.log.Debugf("Fail to get stored block height")
		return e.getLastHeight()
	}
	var height int64
	if _, err := codec.BC.UnmarshalFromBytes(bs, &height); err != nil {
		e.log.Debugf("Fail to parse stored block height")
		return -1
	} else {
		e.log.Tracef("Stored block height:%d", height)
		return height
	}
}

func (e *Executor) StoreBlock(blk *Block) error {
	blkv0 := blk.Original()
	if preps := blkv0.Validators(); preps != nil {
		if bs, err := JSONMarshalAndCompact(preps); err != nil {
			return err
		} else {
			e.jsBucket.Set(preps.Hash(), bs)
		}
	}
	if preps := blkv0.NextValidators(); preps != nil {
		if bs, err := JSONMarshalAndCompact(preps); err != nil {
			return err
		} else {
			e.jsBucket.Set(preps.Hash(), bs)
		}
	}
	if err := blk.Flush(); err != nil {
		return err
	}
	height := blk.Height()
	bid := blk.ID()
	if err := e.blkByID.Set(bid, blk.Bytes()); err != nil {
		return err
	}
	if err := e.blkIndex.Set(BlockIndexKey(height), bid); err != nil {
		return err
	}
	return nil
}

func (e *Executor) FinalizeTransition(tr *Transition) error {
	service.FinalizeTransition(tr.Transition,
		module.FinalizeNormalTransaction|module.FinalizeResult,
	)
	if err := e.StoreBlock(tr.Block); err != nil {
		return err
	}
	if err := e.setLastHeight(tr.Block.Height()); err != nil {
		return errors.Wrap(err, "FailToSetLastHeight")
	}
	return nil
}

type transitionCallback chan error

func (cb transitionCallback) OnValidate(transition module.Transition, err error) {
	cb <- err
}

func (cb transitionCallback) OnExecute(transition module.Transition, err error) {
	cb <- err
}

func FeePaymentEqual(p1, p2 module.FeePayment) bool {
	return common.AddressEqual(p1.Payer(), p2.Payer()) &&
		p1.Amount().Cmp(p2.Amount()) == 0
}

func EventLogEqual(e1, e2 module.EventLog) bool {
	return common.AddressEqual(e1.Address(), e2.Address()) &&
		reflect.DeepEqual(e1.Indexed(), e2.Indexed()) &&
		reflect.DeepEqual(e1.Data(), e2.Data())
}

func CheckStatus(logger log.Logger, s1, s2 module.Status) error {
	if s1 == s2 {
		return nil
	}
	if s1 == module.StatusUnknownFailure && s2 == module.StatusInvalidParameter {
		logger.Warnf("Ignore status difference(e=%s,r=%s)", s1, s2)
		return nil
	}
	return errors.InvalidStateError.Errorf("InvalidStatus(e=%s,r=%s)", s1, s2)
}

func CheckReceipt(logger log.Logger, r1, r2 module.Receipt) error {
	if err := CheckStatus(logger, r1.Status(), r2.Status()); err != nil {
		return err
	}

	if !(r1.To().Equal(r2.To()) &&
		r1.CumulativeStepUsed().Cmp(r2.CumulativeStepUsed()) == 0 &&
		r1.StepUsed().Cmp(r2.StepUsed()) == 0 &&
		r1.StepPrice().Cmp(r2.StepPrice()) == 0 &&
		common.AddressEqual(r1.SCOREAddress(), r2.SCOREAddress()) &&
		r1.LogsBloom().Equal(r2.LogsBloom())) {
		return errors.InvalidStateError.New("DifferentResultValue")
	}

	idx := 0
	for itr1, itr2 := r1.FeePaymentIterator(), r2.FeePaymentIterator(); itr1.Has() || itr2.Has(); _, _, idx = itr1.Next(), itr2.Next(), idx+1 {
		p1, err := itr1.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfPayments")
		}
		p2, err := itr2.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfPayments")
		}
		if !FeePaymentEqual(p1, p2) {
			return errors.InvalidStateError.New("DifferentPayment")
		}
	}

	idx = 0
	for itr1, itr2 := r1.EventLogIterator(), r2.EventLogIterator(); itr1.Has() || itr2.Has(); _, _, idx = itr1.Next(), itr2.Next(), idx+1 {
		e1, err := itr1.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfEvents")
		}
		e2, err := itr2.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfEvents")
		}

		if !EventLogEqual(e1, e2) {
			return errors.InvalidStateError.Errorf("DifferentEvent(idx=%d)", idx)
		}
	}
	return nil
}

func (e *Executor) CheckResult(tr *Transition) error {
	results := tr.NormalReceipts()
	expects := tr.Block.OldReceipts()
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

				e.log.Errorf("Failed Block[ %9d ] TxID[ %#x ]", tr.Block.Height(), tx.ID())
				e.log.Errorf("Failed Transaction[%d]:%s", idx, txjs)
				e.log.Errorf("Expected Receipt[%d]:%s", idx, rct1js)
				e.log.Errorf("Returned Receipt[%d]:%s", idx, rct2js)
				return errors.Wrapf(err, "ReceiptComparisonFailure(idx=%d)", idx)
			}
		}
	}
	rLogBloom := tr.Transition.LogsBloom()
	eLogBloom := tr.Block.LogBloom()
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

func (e *Executor) LoadBlockByHeight(prev *Block, height int64) (*Block, error) {
	blkv0, err := e.cs.GetBlockByHeight(int(height))
	if err != nil {
		return nil, err
	}
	if err := blkv0.Verify(prev.Original()); err != nil {
		return nil, err
	}
	txs := blkv0.NormalTransactions()
	rcts := make([]txresult.Receipt, len(txs))
	txTotal := big.NewInt(int64(len(txs)))
	txTotal = txTotal.Add(txTotal, prev.TxTotal())
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
	return &Block{
		height:  height,
		txs:     transaction.NewTransactionListFromSlice(e.database, txs),
		oldRcts: txresult.NewReceiptListFromSlice(e.database, rcts),
		blk:     blkv0,
		txTotal: txTotal,
	}, nil
}

func (e *Executor) SetupEE() error {
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

func (e *Executor) Execute(from, to int64, useCache bool) error {
	Statusf(e.log, "Executing Blocks from=%d, to=%d", from, to)
	if from < 0 {
		from = e.getLastHeight() + 1
	}
	stored := e.getStoredHeight()
	if to >= 0 && to < from {
		return errors.IllegalArgumentError.Errorf("InvalidArgument(from=%d,to=%d)", from, to)
	}
	prevTR, err := e.InitTransitionFor(from)
	if err != nil {
		return err
	}
	callback := make(transitionCallback, 1)
	for height := from; to < 0 || height <= to; height = height + 1 {
		Statusf(e.log, "[%s] Executing Block[ %9d ] Tx[ %9d ] %s",
			spinner(height, stored), height, prevTR.Block.TxTotal(), TimestampToString(prevTR.Block.Timestamp()))
		tr, err := e.ProposeTransition(prevTR, useCache)
		if err != nil {
			return errors.Wrapf(err, "FailureInPropose(height=%d)", height)
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

		if err := e.CheckResult(tr); err != nil {
			return err
		}

		txTotal := new(big.Int).Add(prevTR.Block.TxTotal(), tr.Block.TxCount())
		e.log.Infof("Finalize Block[ %8d ] Tx[ %16d ]", height, txTotal)
		tr.Block.SetResult(tr.Result(), tr.NextValidators(), tr.NormalReceipts(), txTotal)
		if err := e.FinalizeTransition(tr); err != nil {
			return errors.Wrapf(err, "FinalizationFailure(height=%d)", height)
		}
		if height > stored {
			if err := e.setStoredHeight(height); err != nil {
				return err
			}
		}
		prevTR = tr
	}
	return nil
}

func (e *Executor) Download(from, to int64) error {
	e.log.Infof("Downloading Blocks from=%d, to=%d", from, to)
	stored := e.getStoredHeight()
	last := e.getLastHeight()
	if from < 0 {
		from = stored + 1
	}
	if to >= 0 && to < from {
		return errors.IllegalArgumentError.Errorf("InvalidArgument(from=%d,to=%d)", from, to)
	}
	var prevBlk *Block
	if from > 0 {
		if blk, err := e.GetBlockByHeight(from - 1); err != nil {
			return err
		} else {
			prevBlk = blk
		}
	}
	for height := from; to < 0 || height <= to; height++ {
		Statusf(e.log, "[%s] Download Block [ %8d ]", spinner(height, stored), height)
		blk, err := e.LoadBlockByHeight(prevBlk, height)
		if err != nil {
			return err
		}
		if err := e.StoreBlock(blk); err != nil {
			return err
		}
		if height > stored {
			if err := e.setStoredHeight(height); err != nil {
				return err
			}
		}
		if height <= last {
			if err := e.setLastHeight(height - 1); err != nil {
				return err
			}
			last = height - 1
		}
		prevBlk = blk
	}
	return nil
}
