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
	"context"
	"encoding/json"
	"time"

	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/module"
)

type chainImpl struct {
	database  db.Database
	wallet    module.Wallet
	log       log.Logger
	regulator module.Regulator
	sm        module.ServiceManager
	gs        module.GenesisStorage
}

func (c *chainImpl) Database() db.Database {
	return c.database
}

func (c *chainImpl) Wallet() module.Wallet {
	return c.wallet
}

func (c *chainImpl) NID() int {
	return 1
}

func (c *chainImpl) CID() int {
	return 1
}

func (c *chainImpl) NetID() int {
	return 1
}

func (c *chainImpl) Channel() string {
	return "icon"
}

func (c *chainImpl) ConcurrencyLevel() int {
	return 1
}

func (c *chainImpl) NormalTxPoolSize() int {
	return 5000
}

func (c *chainImpl) PatchTxPoolSize() int {
	return 2
}

func (c *chainImpl) MaxBlockTxBytes() int {
	return 2 * 1024 * 1024
}

func (c *chainImpl) DefaultWaitTimeout() time.Duration {
	panic("implement me")
}

func (c *chainImpl) MaxWaitTimeout() time.Duration {
	panic("implement me")
}

func (c *chainImpl) TransactionTimeout() time.Duration {
	return time.Second * 5
}

var genesis = []byte("{\n  \"accounts\": [\n    {\n      \"name\": \"god\",\n      \"address\": \"hx54f7853dc6481b670caf69c5a27c7c8fe5be8269\",\n      \"balance\": \"0x2961fff8ca4a62327800000\"\n    },\n    {\n      \"name\": \"treasury\",\n      \"address\": \"hx1000000000000000000000000000000000000000\",\n      \"balance\": \"0x0\"\n    }\n  ],\n  \"message\": \"A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \\\"to be\\\" but the fabric of the rhizome is the conjunction, \\\"and ... and ...and...\\\"This conjunction carries enough force to shake and uproot the verb \\\"to be.\\\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\\n\\n - Mille Plateaux, Gilles Deleuze & Felix Guattari\\n\\n\\\"Hyperconnect the world\\\"\"\n}\n")

func (c *chainImpl) Genesis() []byte {
	return genesis
}

func (c *chainImpl) GenesisStorage() module.GenesisStorage {
	return c.gs
}

func (c *chainImpl) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return DecodeCommitVotes
}

func (c *chainImpl) PatchDecoder() module.PatchDecoder {
	return consensus.DecodePatch
}

func (c *chainImpl) BlockManager() module.BlockManager {
	panic("implement me")
}

func (c *chainImpl) Consensus() module.Consensus {
	panic("implement me")
}

func (c *chainImpl) ServiceManager() module.ServiceManager {
	return c.sm
}

type networkManager struct {
	module.NetworkManager
}

func (nm *networkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	return nil, nil
}

func (c *chainImpl) NetworkManager() module.NetworkManager {
	return &networkManager {
	}
}

func (c *chainImpl) Regulator() module.Regulator {
	return c.regulator
}

func (c *chainImpl) Init() error {
	panic("implement me")
}

func (c *chainImpl) Start() error {
	panic("implement me")
}

func (c *chainImpl) Stop() error {
	panic("implement me")
}

func (c *chainImpl) Import(src string, height int64) error {
	panic("implement me")
}

func (c *chainImpl) Prune(gs string, dbt string, height int64) error {
	panic("implement me")
}

func (c *chainImpl) Backup(file string, extra []string) error {
	panic("implement me")
}

func (c *chainImpl) RunTask(task string, params json.RawMessage) error {
	panic("implement me")
}

func (c *chainImpl) Term() error {
	panic("implement me")
}

func (c *chainImpl) State() (string, int64, error) {
	panic("implement me")
}

func (c *chainImpl) IsStarted() bool {
	panic("implement me")
}

func (c *chainImpl) IsStopped() bool {
	panic("implement me")
}

func (c *chainImpl) Reset() error {
	panic("implement me")
}

func (c *chainImpl) Verify() error {
	panic("implement me")
}

func (c *chainImpl) MetricContext() context.Context {
	panic("implement me")
}

func (c *chainImpl) Logger() log.Logger {
	return c.log
}

func NewChain(database db.Database, logger log.Logger) (*chainImpl, error) {
	w := wallet.New()
	return &chainImpl{
		database:  database,
		wallet:    w,
		log:       logger,
		regulator: lcimporter.NewRegulator(),
		gs:        gs.NewFromTx(genesis),
	}, nil
}
