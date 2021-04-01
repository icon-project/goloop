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
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/module"
)

type chainImpl struct {
	database  db.Database
	wallet    module.Wallet
	log       log.Logger
	regulator module.Regulator
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

func (c *chainImpl) Genesis() []byte {
	panic("implement me")
}

func (c *chainImpl) GenesisStorage() module.GenesisStorage {
	panic("implement me")
}

func (c *chainImpl) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	panic("implement me")
}

func (c *chainImpl) PatchDecoder() module.PatchDecoder {
	panic("implement me")
}

func (c *chainImpl) BlockManager() module.BlockManager {
	panic("implement me")
}

func (c *chainImpl) Consensus() module.Consensus {
	panic("implement me")
}

func (c *chainImpl) ServiceManager() module.ServiceManager {
	panic("implement me")
}

func (c *chainImpl) NetworkManager() module.NetworkManager {
	panic("implement me")
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

func NewChain(database db.Database, logger log.Logger) (module.Chain, error) {
	w := wallet.New()
	return &chainImpl{
		database:  database,
		wallet:    w,
		log:       logger,
		regulator: lcimporter.NewRegulator(),
	}, nil
}
