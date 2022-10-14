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

package test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
)

type Chain struct {
	t         *testing.T
	database  db.Database
	wallet    module.Wallet
	log       log.Logger
	regulator module.Regulator
	nm        *NetworkManager
	bm        module.BlockManager
	sm        module.ServiceManager
	cs        module.Consensus
	gs        module.GenesisStorage
	cvd       module.CommitVoteSetDecoder
	gsBytes   []byte
}

func (c *Chain) Database() db.Database {
	return c.database
}

func (c *Chain) Wallet() module.Wallet {
	return c.wallet
}

func (c *Chain) NID() int {
	return 1
}

func (c *Chain) CID() int {
	return 1
}

func (c *Chain) NetID() int {
	return 1
}

func (c *Chain) Channel() string {
	return "icon"
}

func (c *Chain) ConcurrencyLevel() int {
	return 1
}

func (c *Chain) NormalTxPoolSize() int {
	return 5000
}

func (c *Chain) PatchTxPoolSize() int {
	return 2
}

func (c *Chain) MaxBlockTxBytes() int {
	return 2 * 1024 * 1024
}

func (c *Chain) DefaultWaitTimeout() time.Duration {
	panic("implement me")
}

func (c *Chain) MaxWaitTimeout() time.Duration {
	panic("implement me")
}

func (c *Chain) TransactionTimeout() time.Duration {
	return time.Second * 5
}

func (c *Chain) ChildrenLimit() int {
	panic("implement me")
}

func (c *Chain) NephewsLimit() int {
	panic("implement me")
}

func (c *Chain) ValidateTxOnSend() bool {
	panic("implement me")
}

var defaultGenesis = "{\n  \"accounts\": [\n    {\n      \"name\": \"god\",\n      \"address\": \"hx54f7853dc6481b670caf69c5a27c7c8fe5be8269\",\n      \"balance\": \"0x2961fff8ca4a62327800000\"\n    },\n    {\n      \"name\": \"treasury\",\n      \"address\": \"hx1000000000000000000000000000000000000000\",\n      \"balance\": \"0x0\"\n    }\n  ],\n  \"message\": \"A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \\\"to be\\\" but the fabric of the rhizome is the conjunction, \\\"and ... and ...and...\\\"This conjunction carries enough force to shake and uproot the verb \\\"to be.\\\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\\n\\n - Mille Plateaux, Gilles Deleuze & Felix Guattari\\n\\n\\\"Hyperconnect the world\\\"\"\n}\n"

func (c *Chain) Genesis() []byte {
	return c.gsBytes
}

func (c *Chain) GenesisStorage() module.GenesisStorage {
	return c.gs
}

func (c *Chain) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return c.cvd
}

func (c *Chain) PatchDecoder() module.PatchDecoder {
	return consensus.DecodePatch
}

func (c *Chain) BlockManager() module.BlockManager {
	return c.bm
}

func (c *Chain) Consensus() module.Consensus {
	return c.cs
}

func (c *Chain) ServiceManager() module.ServiceManager {
	return c.sm
}

func (c *Chain) NetworkManager() module.NetworkManager {
	return c.nm
}

func (c *Chain) Regulator() module.Regulator {
	return c.regulator
}

func (c *Chain) Init() error {
	panic("implement me")
}

func (c *Chain) Start() error {
	panic("implement me")
}

func (c *Chain) Stop() error {
	panic("implement me")
}

func (c *Chain) Import(src string, height int64) error {
	panic("implement me")
}

func (c *Chain) Prune(gs string, dbt string, height int64) error {
	panic("implement me")
}

func (c *Chain) Backup(file string, extra []string) error {
	panic("implement me")
}

func (c *Chain) RunTask(task string, params json.RawMessage) error {
	panic("implement me")
}

func (c *Chain) Term() error {
	panic("implement me")
}

func (c *Chain) State() (string, int64, error) {
	panic("implement me")
}

func (c *Chain) IsStarted() bool {
	panic("implement me")
}

func (c *Chain) IsStopped() bool {
	panic("implement me")
}

func (c *Chain) Reset(gs string, height int64, blockHash []byte) error {
	panic("implement me")
}

func (c *Chain) Verify() error {
	panic("implement me")
}

func (c *Chain) MetricContext() context.Context {
	return nil
}

func (c *Chain) Logger() log.Logger {
	return c.log
}

func (c *Chain) SetBlockManager(bm module.BlockManager) {
	c.bm = bm
}

func (c *Chain) SetServiceManager(sm module.ServiceManager) {
	c.sm = sm
}

func (c *Chain) DoDBTask(f func(database db.Database)) {
	panic("implement me")
}

func (c *Chain) Close() {
	c.nm.Close()
}

func NewChain(
	t *testing.T,
	w module.Wallet,
	database db.Database,
	logger log.Logger,
	cvd module.CommitVoteSetDecoder,
	gsStr string,
) (*Chain, error) {
	logger = logger.WithFields(log.Fields{
		log.FieldKeyWallet: hex.EncodeToString(w.Address().ID()),
	})
	gsBytes := []byte(gsStr)
	return &Chain{
		database:  database,
		wallet:    w,
		log:       logger,
		regulator: NewRegulator(),
		nm:        NewNetworkManager(t, w.Address()),
		gs:        gs.NewFromTx(gsBytes),
		cvd:       cvd,
		gsBytes:   gsBytes,
	}, nil
}
