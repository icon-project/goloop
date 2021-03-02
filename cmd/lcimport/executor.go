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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
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
	KeyLastBlockHeight = "block.lastHeight"
)

const (
	JSONByHash db.BucketID = "J"
)

// executeTransactions executes transactions from lc and confirm results.
// then it stores actual results.
// If from is negative, it executes from
type Executor struct {
	lc       *lcstore.Store
	cs       *CacheStore
	database db.Database
	cm       contract.ContractManager
	em       eeproxy.Manager
	chain    module.Chain
	log      log.Logger
	plt      service.Platform

	jsBucket    db.Bucket
	blkIndex    db.Bucket
	blkByID     db.Bucket
	chainBucket db.Bucket
}

type Transition struct {
	module.Transition
	Block *Block
}

func NewExecutor(logger log.Logger, lc *lcstore.Store, data string) (*Executor, error) {
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
	cm, err := plt.NewContractManager(database, path.Join(data, ContractPath), logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewContractManagerFailure")
	}
	ee, err := eeproxy.AllocEngines(logger, "python")
	if err != nil {
		return nil, errors.Wrap(err, "FailureInAllocEngines")
	}
	em, err := eeproxy.NewManager("unix", path.Join(data, EESocketPath), logger, ee...)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInAllocProxyManager")
	}

	go em.Loop()
	em.SetInstances(1, 1, 1)

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
		lc:          lc,
		cs:          NewCacheStore(logger, lc),
		log:         logger,
		chain:       chain,
		plt:         plt,
		cm:          cm,
		em:          em,
		jsBucket:    jsBucket,
		blkIndex:    blkIndex,
		blkByID:     blkByID,
		chainBucket: chainBucket,
	}
	ex.database = db.WithFlags(database, db.Flags{
		FlagExecutor: ex,
	})
	logger.Infoln("Initialize executor : SUCCESS")
	return ex, nil
}

func (e *Executor) GetRepsByHash(hash []byte) (*blockv0.RepsList, error) {
	if js, err := e.jsBucket.Get(hash); err != nil || js == nil {
		return e.lc.GetRepsByHash(hash)
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

func (e *Executor) InitTransitionFor(height int64) (*Transition, error) {
	if height < 0 {
		return nil, errors.Errorf("InvalidHeight(height=%d)", height)
	}
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
			e.log,
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
			e.log,
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

func (e *Executor) ProposeTransition(last *Transition) (*Transition, error) {
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
	if blk == nil {
		e.log.Tracef("get the block from the store height=%d", height)
		blkv0, err := e.cs.GetBlockByHeight(int(height))
		if err != nil {
			return nil, err
		}
		e.log.Tracef("verify retrieved the block height=%d", height)
		if err := blkv0.Verify(last.Block.Original()); err != nil {
			return nil, err
		}

		e.log.Tracef("get receipts of the block height=%d", height)
		txs := blkv0.NormalTransactions()
		rcts := make([]txresult.Receipt, len(txs))
		for idx, tx := range txs {
			if err := tx.Verify(); err != nil {
				return nil, err
			}
			rct, err := e.cs.GetReceiptByTransaction(tx.ID())
			if err != nil {
				return nil, errors.Wrapf(err, "FailureInGetReceipts(txid=%#x)", tx.ID())
			}
			rcts[idx] = rct.(txresult.Receipt)
		}
		blk = &Block{
			height: height,
			txs:    transaction.NewTransactionListFromSlice(e.database, txs),
			rcts:   txresult.NewReceiptListFromSlice(e.database, rcts),
			blk:    blkv0,
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

func (e *Executor) FinalizeTransition(tr *Transition) error {
	service.FinalizeTransition(tr.Transition,
		module.FinalizeNormalTransaction|module.FinalizeResult,
	)
	blkv0 := tr.Block.Original()
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

	height := tr.Block.Height()
	bid := tr.Block.ID()
	if err := e.blkByID.Set(bid, tr.Block.Bytes()); err != nil {
		return err
	}
	if err := e.blkIndex.Set(BlockIndexKey(height), bid); err != nil {
		return err
	}
	if err := e.setLastHeight(height); err != nil {
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

func (e *Executor) CheckResult(tr *Transition) error {
	results := tr.NormalReceipts()
	expects := tr.Block.Receipts()
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
			if err := rct1.Check(rct2); err != nil {
				rct1js, _ := JSONMarshalIndent(rct1)
				rct2js, _ := JSONMarshalIndent(rct2)
				var txjs []byte
				if tx, err := tr.Transition.NormalTransactions().Get(idx); err == nil {
					txjs, _ = JSONMarshalIndent(tx)
				}
				e.log.Warnf("Failed Transaction[%d]:%s", idx, txjs)
				e.log.Warnf("Expected Receipt[%d]:%s", idx, rct1js)
				e.log.Warnf("Returned Receipt[%d]:%s", idx, rct2js)
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

func (e *Executor) Execute(from, to int64) error {
	Statusf(e.log, "Executing Blocks from=%d, to=%d", from, to)
	if from < 0 {
		from = e.getLastHeight() + 1
	}
	if to >= 0 && to < from {
		return errors.IllegalArgumentError.Errorf("InvalidArgument(from=%d,to=%d)", from, to)
	}
	prevTR, err := e.InitTransitionFor(from)
	if err != nil {
		return err
	}
	callback := make(transitionCallback, 1)
	for height := from; to < 0 || height <= to; height = height + 1 {
		Statusf(e.log, "Executing Block[ %8d ] Tx[ %16d ]",
			height, prevTR.Block.TxTotal())
		tr, err := e.ProposeTransition(prevTR)
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
		prevTR = tr
	}
	return nil
}
