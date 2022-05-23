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
	"io/ioutil"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/module"
)

const (
	importBlockInterval = time.Second * 1
)

type Importer struct {
	chain  *chainImpl
	sm     *lcimporter.ServiceManager
	bm     module.BlockManager
	waiter chan interface{}
}

func (i *Importer) OnResult(err error) {
	i.waiter <- err
}

func (i *Importer) commitAndFinalize(c module.BlockCandidate) error {
	if err := i.bm.Commit(c); err != nil {
		return err
	}
	if err := i.bm.Finalize(c); err != nil {
		return err
	}
	Statusf(i.chain.Logger(), "Importer finalized [ %8d ] [ %8d ]", c.Height(), i.sm.GetImportedBlocks())
	return nil
}

func (i *Importer) OnBlock(candidate module.BlockCandidate, err error) {
	if err != nil {
		i.waiter <- err
		return
	}
	if err := i.commitAndFinalize(candidate); err != nil {
		i.waiter <- err
	}
	time.Sleep(importBlockInterval)
	if _, err := i.bm.Propose(candidate.ID(), NewCommitVotes(), i.OnBlock); err != nil {
		i.waiter <- err
	}
}

func (i *Importer) Run() error {
	i.sm.Start()
	blk, err := i.bm.GetLastBlock()
	if err != nil {
		return err
	}
	votes := NewCommitVotes()
	_, err = i.bm.Propose(blk.ID(), votes, i.OnBlock)
	if err != nil {
		return err
	}
	result := <-i.waiter
	switch obj := result.(type) {
	case error:
		return obj
	default:
		return nil
	}
}

func NewImporter(
	base string,
	dbType string,
	storeURI string,
	maxRPS int,
	genesis string,
	cacheConfig *lcstore.CacheConfig,
	logger log.Logger,
) (*Importer, error) {
	cdb, err := db.Open(base, dbType, "import_db")
	if err != nil {
		return nil, err
	}
	rdb, err := db.Open(base, dbType, "real_db")
	if err != nil {
		return nil, err
	}
	gnsBytes, err := ioutil.ReadFile(genesis)
	if err != nil {
		return nil, err
	}
	gns, err := gs.New(gnsBytes)
	if err != nil {
		return nil, err
	}
	chain, err := NewChain(cdb, gns, logger)
	if err != nil {
		return nil, err
	}
	plt, err := icon.NewPlatform(base, chain.CID())
	cfg := &lcimporter.Config{
		Validators:  nil,
		StoreURI:    storeURI,
		MaxRPS:      maxRPS,
		CacheConfig: *cacheConfig,
		BaseDir:     base,
		Platform:    plt,
	}
	im := new(Importer)
	sm, err := lcimporter.NewServiceManager(chain, rdb, cfg, im)
	if err != nil {
		return nil, err
	}
	chain.sm = sm

	bm, err := block.NewManager(chain, nil, nil)
	if err != nil {
		return nil, err
	}

	im.chain = chain
	im.sm = sm
	im.bm = bm
	im.waiter = make(chan interface{}, 1)
	return im, nil
}
