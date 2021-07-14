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
	"fmt"
	"math/big"
	"path"
	"sync"
	"time"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/lcimporter"
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
	TraceBFC = false
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
	BlockV1ByID db.BucketID = "B"
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
	SetReceiptParameter(dbase db.Database, rev module.Revision)
}

type GetTPSer interface {
	GetTPS() float32
}

func NewExecutor(logger log.Logger, cs Store, data string, dbtype string) (*Executor, error) {
	database, err := db.Open(data, dbtype, "database")
	if err != nil {
		return nil, errors.Wrapf(err, "DatabaseFailure(path=%s)", data)
	}
	cs.SetReceiptParameter(database, module.LatestRevision)
	chain, err := NewChain(database, logger)
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
	blkByID, err := database.GetBucket(BlockV1ByID)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=BlockV1ByID)")
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
			blk.NextValidators(),
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
		if err := service.FinalizeTransition(tr, module.FinalizeResult, true); err != nil {
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

const (
	maxBlockWorkers = 8
	maxBlockCache   = 32
)

type blockTask struct {
	height int64
	chn    chan interface{}
}

type blockForwardCache struct {
	e       *Executor

	lock    sync.Mutex
	active  int
	blocks  map[int64]*blockTask
	tasks   chan *blockTask
	pool    sync.Pool
}

func (bp *blockForwardCache) Tracef(s string, args ...interface{}) {
	if TraceBFC {
		bp.e.log.Tracef(s, args...)
	}
}

func (bp *blockForwardCache) runTasks() {
	defer func() {
		func() {
			bp.lock.Lock()
			defer bp.lock.Unlock()
			bp.active -= 1
			bp.Tracef("BFC.exitTask(active=%d)", bp.active)
		}()
	}()

	for {
		select {
		case task := <-bp.tasks:
			if blk, err := bp.e.GetBlockByHeight(task.height); err != nil {
				task.chn <- err
			} else if blk != nil {
				bp.Tracef("BFC.done(height=%d)", task.height)
				bp.scheduleBlocksFor(task.height)
				// preload transactions and receipts to the memory
				txs := blk.Transactions()
				for itr := txs.Iterator(); itr.Has(); itr.Next() {
					// do nothing
				}
				rts := blk.Receipts()
				for itr := rts.Iterator(); itr.Has(); itr.Next() {
					// do nothing
				}
				task.chn <- blk
			} else {
				task.chn <- blk
			}
		default:
			return
		}
	}
}

func (bp *blockForwardCache) scheduleBlockInLock(height int64) bool {
	if len(bp.blocks) >= maxBlockCache {
		return false
	}
	if bt, ok := bp.blocks[height]; !ok {
		bp.Tracef("BFC.schedule(height=%d)", height)
		bt = bp.allocBlockTask(height)
		bp.blocks[height] = bt
		bp.tasks <- bt
		return true
	}
	return false
}

func (bp *blockForwardCache) scheduleBlocksFor(height int64) {
	bp.lock.Lock()
	defer bp.lock.Unlock()

	for h := height + 1; h < height+1+maxBlockCache; h += 1 {
		if bp.scheduleBlockInLock(h) {
			bp.tryNewWorkerInLock()
		} else {
			continue
		}
	}
}

func (bp *blockForwardCache) tryNewWorkerInLock() {
	if bp.active < maxBlockWorkers {
		bp.active += 1
		bp.Tracef("BFC.newTask(active=%d)", bp.active)
		go bp.runTasks()
	}
}

func (bp *blockForwardCache) allocBlockTask(height int64) *blockTask {
	bt := bp.pool.Get().(*blockTask)
	bt.height = height
	return bt
}

func (bp *blockForwardCache) deallocBlockTask(bt *blockTask) {
	bp.pool.Put(bt)
}

// fetchBlockTask fetch blockTask from scheduled blockTasks or make
// new one with scheduling
func (bp *blockForwardCache) fetchBlockTask(height int64) *blockTask {
	bp.lock.Lock()
	defer bp.lock.Unlock()

	if bt, ok := bp.blocks[height]; ok {
		delete(bp.blocks, height)
		return bt
	} else {
		bt = bp.allocBlockTask(height)
		bp.Tracef("BFC.schedule2(height=%d)", height)
		bp.tasks <- bt
		bp.tryNewWorkerInLock()
		return bt
	}
}

func (bp *blockForwardCache) GetBlock(height int64) (*Block, error){
	bt := bp.fetchBlockTask(height)
	res := <-bt.chn
	bp.deallocBlockTask(bt)
	switch obj := res.(type) {
	case *Block:
		bp.scheduleBlocksFor(height)
		return obj, nil
	case error:
		return nil, obj
	default:
		return nil, errors.InvalidStateError.Errorf("InvalidObjectReturned(obj=%+v)", res)
	}
}

func newBlockForwardCache(e *Executor) *blockForwardCache {
	return &blockForwardCache{
		e:      e,
		blocks: make(map[int64]*blockTask),
		tasks:  make(chan *blockTask, maxBlockCache),
		pool: sync.Pool{
			New: func() interface{} {
				return &blockTask{
					chn: make(chan interface{}, 1),
				}
			},
		},
	}
}

func (e *Executor) PrefetchBlocks(last *Block, from, to int64, noStored bool) <-chan interface{} {
	bp  := newBlockForwardCache(e)
	chn := make(chan interface{}, maxBlockWorkers)
	go func() {
		for height := from; to < 0 || height <= to; height = height + 1 {
			var blk *Block
			if !noStored {
				if b, err := bp.GetBlock(height); err != nil {
					chn <- err
					break
				} else {
					if b != nil {
						blk = b
					} else {
						noStored = true
					}
				}
			}
			if blk == nil {
				if b, err := e.LoadBlockByHeight(last, height); err != nil {
					chn <- err
					break
				} else {
					blk = b
				}
			}
			if blk != nil {
				chn <- blk
				last = blk
			} else {
				break
			}
		}
		close(chn)
	}()
	return chn
}

func FetchBlock(chn <- chan interface{}) (*Block, error) {
	out, ok := <- chn
	if !ok {
		return nil, errors.InvalidStateError.New("NoMoreBlock")
	}
	switch obj := out.(type) {
	case *Block:
		return obj, nil
	case error:
		return nil, obj
	default:
		panic("InvalidObjectType")
	}
}

func (e *Executor) consensusInfoFor(block_ blockv0.Block, prev_ blockv0.Block) (module.ConsensusInfo, error) {
	var voters module.ValidatorList
	var voted []bool
	var err error
	switch block := block_.(type) {
	case *blockv0.BlockV01a:
	case *blockv0.BlockV03:
		switch prev := prev_.(type) {
		case *blockv0.BlockV01a:
			voters, err = block.Validators().GetValidatorList(e.database)
			if err != nil {
				return nil, err
			}
			voted = make([]bool, voters.Len())
			err = block.PrevVotes().CheckVoters(block.Validators(), voted)
			if err != nil {
				return nil, err
			}
		case *blockv0.BlockV03:
			voters, err = prev.Validators().GetValidatorList(e.database)
			if err != nil {
				return nil, err
			}
			voted = make([]bool, voters.Len())
			err = block.PrevVotes().CheckVoters(prev.Validators(), voted)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.UnsupportedError.New("Unsupported")
		}
	default:
		return nil, errors.UnsupportedError.New("Unsupported")
	}
	return common.NewConsensusInfo(prev_.Proposer(), voters, voted), nil
}

func (e *Executor) ProposeTransition(last *Transition, chn <- chan interface{}) (*Transition, error) {
	var height int64
	if last.Block != nil {
		height = last.Block.Height() + 1
	} else {
		height = 0
	}
	blk, err := FetchBlock(chn)
	if err != nil {
		return nil, err
	}
	var csi module.ConsensusInfo
	if height == 0 {
		csi = common.NewConsensusInfo(nil, nil, nil)
	} else {
		csi, err = e.consensusInfoFor(blk.Original(), last.Block.Original())
		if err != nil {
			return nil, err
		}
	}
	tr := service.NewTransition(
		last.Transition,
		nil,
		blk.Transactions(),
		common.NewBlockInfo(height, blk.Timestamp()),
		csi,
		true,
	)
	return &Transition{tr, blk}, nil
}

func (e *Executor) ResetTransition(tr *Transition) {
	tr.Transition = service.PatchTransition(tr.Transition, nil, nil)
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
		false,
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
			if err := lcimporter.CheckReceipt(e.log, rct1, rct2); err != nil {
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
	if reps := tr.Block.Original().NextValidators(); reps != nil {
		rs := reps.Size()
		validators := tr.Transition.NextValidators()
		vs := validators.Len()
		if vs > 0 {
			if vs != rs {
				return errors.Errorf("InvalidValidatorLen(exp=%d,calc=%d)", rs, vs)
			}
			for i:=0 ; i< rs; i++ {
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
	if blkv03, ok := blkv0.(*blockv0.BlockV03); ok {
		eReceiptListHash := blkv03.ReceiptsHash()
		rReceiptListHash := blockv0.CalcMerkleRootOfReceiptSlice(rcts, txs, blkv0.Height())
		if !bytes.Equal(eReceiptListHash, rReceiptListHash) {
			return nil, errors.Errorf("DifferentReceiptListHash(stored=%#x,real=%#x)",
				eReceiptListHash, rReceiptListHash)
		}
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
	e.database = cache.AttachManager(e.database, path.Join(e.baseDir, chain.DefaultCacheDir), 0, 0)
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

func (e *Executor) Execute(from, to int64, noStored, dryRun bool) error {
	Statusf(e.log, "Executing Blocks from=%d, to=%d", from, to)
	if from < 0 {
		from = e.getLastHeight() + 1
	}
	getTPSer, _ := e.cs.(GetTPSer)
	stored := e.getStoredHeight()
	if to >= 0 && to < from {
		return errors.IllegalArgumentError.Errorf("InvalidArgument(from=%d,to=%d)", from, to)
	}
	prevTR, err := e.InitTransitionFor(from)
	if err != nil {
		return err
	}
	chn := e.PrefetchBlocks(prevTR.Block, from, to, noStored)
	callback := make(transitionCallback, 1)
	var rps, tps float32
	tm := new(lcimporter.TPSMeasure).Init(100)
	for height := from; to < 0 || height <= to; height = height + 1 {
		tr, err := e.ProposeTransition(prevTR, chn)
		if err != nil {
			return errors.Wrapf(err, "FailureInPropose(height=%d)", height)
		}
		txTotal := new(big.Int).Add(prevTR.Block.TxTotal(), tr.Block.TxCount())
		rps = getTPSer.GetTPS()
		tps = tm.GetTPS()
		Statusf(
			e.log,
			"[%s] Executing Block[ %10s ] Tx[ %11s ] %s RPS[ %6.1f ] TPS[ %6.1f ]",
			spinner(height, stored),
			D(height),
			D(txTotal),
			TimestampToString(tr.Block.Timestamp()),
			rps,
			tps,
		)

		// repeating execution if Reward Calculator has not finished its job.
		for true {
			if _, err = tr.Execute(callback); err != nil {
				return errors.Wrapf(err, "FailureInExecute(height=%d)", height)
			}
			err = <-callback
			if err != nil {
				return errors.Wrapf(err, "PreValidationFail(height=%d)", height)
			}
			err = <-callback
			if err != nil {
				if icmodule.CalculationNotFinishedError.Equals(err) {
					e.ResetTransition(tr)
					time.Sleep(time.Second*5)
					continue
				}
				return errors.Wrapf(err, "ExecutionFailure(height=%d)", height)
			} else {
				break
			}
		}

		if err := e.CheckResult(tr); err != nil {
			return err
		}

		if dryRun {
			e.log.Infof("Check Block[ %9d ] Tx[ %9d ]", height, txTotal)
			if err := tr.Block.CheckResult(e.log, tr.Result(), tr.NextValidators(), tr.NormalReceipts(), txTotal); err != nil {
				service.FinalizeTransition(tr.Transition, module.FinalizeResult, false)
				erv, _ := ParseResult(tr.Block.Result())
				rrv, _ := ParseResult(tr.Transition.Result())
				if erv != nil && rrv != nil {
					showResultDiff(e.database, e.log, erv, rrv)
				}
				return err
			}
			service.FinalizeTransition(tr.Transition, module.FinalizeResult, true)
		} else {
			e.log.Infof("Finalize Block[ %9d ] Tx[ %9d ]", height, txTotal)
			tr.Block.SetResult(tr.Result(), tr.NextValidators(), tr.NormalReceipts(), txTotal)
			if err := e.FinalizeTransition(tr); err != nil {
				return errors.Wrapf(err, "FinalizationFailure(height=%d)", height)
			}
			if height > stored {
				if err := e.setStoredHeight(height); err != nil {
					return err
				}
			}
		}
		tm.OnTransactions(tr.Block.TxCount())
		prevTR = tr
	}
	return nil
}

func (e *Executor) Download(from, to int64) error {
	e.log.Infof("Downloading Blocks from=%d, to=%d", from, to)
	tpser, _ := e.cs.(GetTPSer)
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
	var tps float32
	for height := from; to < 0 || height <= to; height++ {
		if tpser != nil {
			tps = tpser.GetTPS()
		}
		Statusf(
			e.log,
			"[%s] Downloading Block[ %10s ]  Tx[ %11s ] RPS [ %6.1f ]",
			spinner(height, stored),
			D(height),
			D(prevBlk.TxTotal()),
			tps,
		)
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
