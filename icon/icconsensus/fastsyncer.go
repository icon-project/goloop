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

package icconsensus

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
)

const sleepInterval = 3 * time.Second

type fastSyncer struct {
	mu            sync.Mutex
	height        int64
	to            int64
	c             base.Chain
	parent        *wrapper
	bpp           *bpp
	fsm           fastsync.Manager
	fetchCanceler func() bool
	blockCanceler module.Canceler
	log           log.Logger
	running       bool
}

func newFastSyncer(
	height int64,
	to int64,
	c base.Chain,
	parent *wrapper,
	bpp *bpp,
) *fastSyncer {
	f := &fastSyncer{
		height: height,
		to:     to,
		c:      c,
		parent: parent,
		bpp:    bpp,
	}
	f.log = c.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "CS|V1|",
	})
	return f
}

func (f *fastSyncer) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	fsm, err := fastsync.NewManager(
		f.c.NetworkManager(),
		f.c.BlockManager(),
		f,
		f.c.Logger(),
	)
	if err != nil {
		return err
	}
	f.fsm = fsm
	f.fsm.StartServer()
	canceler, err := f.fsm.FetchBlocks(f.height, f.to, f)
	if err != nil {
		return err
	}
	f.fetchCanceler = canceler
	f.running = true
	return nil
}

func (f *fastSyncer) Term() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fsm.StopServer()
	if f.fetchCanceler != nil {
		f.fetchCanceler()
	}
	if f.blockCanceler != nil {
		f.blockCanceler.Cancel()
	}
	f.running = false
}

func (f *fastSyncer) GetStatus() *module.ConsensusStatus {
	f.mu.Lock()
	defer f.mu.Unlock()

	return &module.ConsensusStatus{
		Height:   f.height,
		Round:    0,
		Proposer: false,
	}
}

func (f *fastSyncer) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	return nil, errors.NotFoundError.New("not found")
}

func (f *fastSyncer) GetBlockProof(height int64, opt int32) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.bpp.GetBlockProof(height, opt)
}

func (f *fastSyncer) OnBlock(br fastsync.BlockResult) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return
	}

	blk := br.Block()
	f.log.Debugf("ReceiveBlock Height:%d\n", blk.Height())

	f.processBlock(br)
}

func (f *fastSyncer) processBlock(br fastsync.BlockResult) {
	blk := br.Block()
	var proof [][]byte
	_, err := codec.UnmarshalFromBytes(br.Votes(), &proof)
	if err != nil {
		br.Reject()
	}
	err = f.bpp.Add(blk.Height(), blk.Hash(), proof)
	if err != nil {
		br.Reject()
	}
	canceler, err := f.c.BlockManager().ImportBlock(
		br.Block(),
		module.ImportByForce,
		func(bc module.BlockCandidate, err error) {
			f.mu.Lock()
			defer f.mu.Unlock()

			if !f.running {
				return
			}
			if err != nil {
				f.log.Panicf("import cb error %+v", err)
			}
			err = f.c.BlockManager().Finalize(bc)
			if err != nil {
				f.log.Panicf("finalize error %+v", err)
			}
			br.Consume()
			f.height++
			f.blockCanceler = nil
		},
	)
	if err != nil {
		f.log.Panicf("import returned %+v", err)
	}
	f.blockCanceler = canceler
}

func (f *fastSyncer) RepeatTryFetch() {
	for {
		time.Sleep(sleepInterval)

		f.mu.Lock()
		canceler, err := f.fsm.FetchBlocks(f.height, f.to, f)
		if err != nil {
			f.mu.Unlock()
			continue
		}
		f.fetchCanceler = canceler
		f.mu.Unlock()
		return
	}
}

func (f *fastSyncer) OnEnd(err error) {
	ul := common.Lock(&f.mu)
	defer ul.Unlock()

	if f.height < f.to {
		f.log.Warnf("fast syncer failed: %+v", err)
		go f.RepeatTryFetch()
		return
	}
	parent := f.parent
	ul.Unlock()

	parent.Upgrade(f.bpp)
}
