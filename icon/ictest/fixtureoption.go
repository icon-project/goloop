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

package ictest

import (
	"path"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/icon/icconsensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/platform/basic"
	"github.com/icon-project/goloop/test"
)

type MerkleInfo interface {
	MerkleRoot() []byte
	MerkleLeaves() int64
}

type platform struct {
	service.Platform
	mtRoot []byte
	mtCap  int64
}

func (plt *platform) DefaultBlockVersion() int {
	return module.BlockVersion1
}

func (plt *platform) MerkleRoot() []byte {
	return plt.mtRoot
}

func (plt *platform) MerkleLeaves() int64 {
	return plt.mtCap
}

func UseMerkle(root []byte, leaves int64) test.FixtureOption {
	return test.UseConfig(&test.FixtureConfig{
		MerkleRoot: root,
		MerkleLeaves: leaves,
	})
}

func UseBMForBlockV1(cf *test.FixtureConfig) *test.FixtureConfig {
	return cf.Override(&test.FixtureConfig{
		CVSD: func(bytes []byte) module.CommitVoteSet {
			vs := consensus.NewCommitVoteSetFromBytes(bytes)
			if vs != nil {
				return vs
			}
			vl, _ := blockv0.NewBlockVotesFromBytes(bytes)
			return vl
		},
		NewPlatform: func(ctx *test.NodeContext) service.Platform {
			return &platform{
				basic.Platform,
				ctx.Config.MerkleRoot,
				ctx.Config.MerkleLeaves,
			}
		},
		NewBM: func(ctx *test.NodeContext) module.BlockManager {
			c := ctx.C
			handlers := []block.Handler{
				blockv1.NewHandler(c),
				block.NewBlockV2Handler(c),
			}
			bm, err := block.NewManager(c, nil, handlers)
			assert.NoError(ctx.Config.T, err)
			return bm
		},
	})
}

func UseCSForBlockV1(cf *test.FixtureConfig) *test.FixtureConfig {
	return cf.Override(&test.FixtureConfig{
		NewCS: func(ctx *test.NodeContext) module.Consensus {
			t := ctx.Config.T
			iplt, ok := ctx.Platform.(MerkleInfo)
			assert.True(t, ok)
			wal := path.Join(ctx.Base, "wal")
			wm := test.NewWAL()
			cs, err := icconsensus.New(
				ctx.C,
				wal,
				wm,
				nil,
				iplt.MerkleRoot(),
				iplt.MerkleLeaves(),
			)
			assert.NoError(t, err)
			return cs
		},
	})
}
