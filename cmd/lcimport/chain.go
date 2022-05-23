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
	nid, _ := c.gs.NID()
	return nid
}

func (c *chainImpl) CID() int {
	cid, _ := c.gs.CID()
	return cid
}

func (c *chainImpl) NetID() int {
	return c.CID()
}

func (c *chainImpl) Channel() string {
	return "icon_dex"
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

func (c *chainImpl) ChildrenLimit() int {
	panic("implement me")
}

func (c *chainImpl) NephewsLimit() int {
	panic("implement me")
}

func (c *chainImpl) Genesis() []byte {
	return c.gs.Genesis()
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

func (nm *networkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	return nil, nil
}

func (c *chainImpl) NetworkManager() module.NetworkManager {
	return &networkManager{}
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

func (c *chainImpl) ValidateTxOnSend() bool {
	return false
}

func (c *chainImpl) WalletFor(dsa string) module.BaseWallet {
	return nil
}

func NewChain(database db.Database, gns module.GenesisStorage, logger log.Logger) (*chainImpl, error) {
	w := wallet.New()
	return &chainImpl{
		database:  database,
		wallet:    w,
		log:       logger,
		regulator: lcimporter.NewRegulator(),
		gs:        gns,
	}, nil
}
