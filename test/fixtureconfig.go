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
	"path"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/platform/basic"
)

type FixtureConfig struct {
	T          T
	MerkleRoot []byte
	MerkleLeaves      int64
	MerkleLastVotes   []byte
	Prefix            string
	Dbase             func() db.Database
	CVSD              module.CommitVoteSetDecoder
	NewPlatform       func(ctx *NodeContext) base.Platform
	NewSM             func(ctx *NodeContext) module.ServiceManager
	NewBM             func(ctx *NodeContext) module.BlockManager
	NewCS             func(ctx *NodeContext) module.Consensus
	AddValidatorNodes int
	Genesis           string
	GenesisStorage    module.GenesisStorage
	Wallet            module.Wallet
	AddDefaultNode    *bool
	WAL               func() consensus.WALManager
}

func NewFixtureConfig(t T, o ...FixtureOption) *FixtureConfig {
	tru := true
	cf := &FixtureConfig{
		T:      t,
		Prefix: "goloop-block-fixture",
		Dbase: func() db.Database {
			return db.NewMapDB()
		},
		CVSD: consensus.NewCommitVoteSetFromBytes,
		NewPlatform: func(ctx *NodeContext) base.Platform {
			return basic.Platform
		},
		NewSM: func(ctx *NodeContext) module.ServiceManager {
			return NewServiceManager(ctx.C, ctx.Platform, ctx.CM, ctx.EM)
		},
		NewBM: func(ctx *NodeContext) module.BlockManager {
			bm, err := block.NewManager(ctx.C, nil, nil)
			assert.NoError(ctx.Config.T, err)
			return bm
		},
		NewCS: func(ctx *NodeContext) module.Consensus {
			wm := ctx.Config.WAL()
			wal := path.Join(ctx.Base, "wal")
			cs := consensus.New(
				ctx.C, wal, wm, nil, nil, nil,
			)
			assert.NotNil(ctx.Config.T, cs)
			return cs
		},
		AddValidatorNodes: 0,
		Genesis:           defaultGenesis,
		GenesisStorage:    nil,
		AddDefaultNode:    &tru,
		WAL: func() consensus.WALManager {
			return consensus.NewTestWAL()
		},
	}
	return cf.ApplyOption(o...)
}

func (cf *FixtureConfig) ApplyOption(o ...FixtureOption) *FixtureConfig {
	res := cf
	for _, op := range o {
		res = op(res)
	}
	return res
}

func (cf *FixtureConfig) Override(cf2 *FixtureConfig) *FixtureConfig {
	res := *cf
	if cf2.T != nil {
		res.T = cf2.T
	}
	if cf2.MerkleRoot != nil {
		res.MerkleRoot = cf2.MerkleRoot
		res.MerkleLeaves = cf2.MerkleLeaves
		res.MerkleLastVotes = cf2.MerkleLastVotes
	}
	if len(cf2.Prefix) != 0 {
		res.Prefix = cf2.Prefix
	}
	if cf2.Dbase != nil {
		res.Dbase = cf2.Dbase
	}
	if cf2.CVSD != nil {
		res.CVSD = cf2.CVSD
	}
	if cf2.NewPlatform != nil {
		res.NewPlatform = cf2.NewPlatform
	}
	if cf2.NewSM != nil {
		res.NewSM = cf2.NewSM
	}
	if cf2.NewBM != nil {
		res.NewBM = cf2.NewBM
	}
	if cf2.NewCS != nil {
		res.NewCS = cf2.NewCS
	}
	if cf2.AddValidatorNodes != 0 {
		res.AddValidatorNodes = cf2.AddValidatorNodes
	}
	if len(cf2.Genesis) != 0 {
		res.Genesis = cf2.Genesis
	}
	if cf2.GenesisStorage != nil {
		res.GenesisStorage = cf2.GenesisStorage
	}
	if cf2.Wallet != nil {
		res.Wallet = cf2.Wallet
	}
	if cf2.AddDefaultNode != nil {
		res.AddDefaultNode = cf2.AddDefaultNode
	}
	if cf2.WAL != nil {
		res.WAL = cf2.WAL
	}
	return &res
}
