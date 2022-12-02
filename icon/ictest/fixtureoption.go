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
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/icon/icconsensus"
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/platform/basic"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/test"
)

type platform struct {
	base.Platform
	mh          hexary.MerkleHeader
	mtLastVotes *blockv0.BlockVoteList
}

func NewPlatform() base.Platform {
	return &platform{
		Platform:    basic.Platform,
		mh:          hexary.MerkleHeader{},
		mtLastVotes: nil,
	}
}

func (plt *platform) GetBlockV1Proof() (
	*hexary.MerkleHeader, *blockv0.BlockVoteList, error,
) {
	return &plt.mh, plt.mtLastVotes, nil
}

func (plt *platform) SetBlockV1Proof(root []byte, size int64, votes *blockv0.BlockVoteList) error {
	plt.mh = hexary.MerkleHeader{RootHash: root, Leaves: size}
	plt.mtLastVotes = votes
	return nil
}

func (plt *platform) DefaultBlockVersionFor(cid int) int {
	return module.BlockVersion1
}

func (plt *platform) NewBlockHandlers(c base.Chain) []base.BlockHandler {
	return nil
}

type contractManager struct {
	contract.ContractManager
}

func (cm *contractManager) GenesisTo() module.Address {
	return state.ZeroAddress
}

func (plt *platform) NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	if cm, err := plt.Platform.NewContractManager(dbase, dir, logger); err != nil {
		return nil, err
	} else {
		return &contractManager{cm}, nil
	}
}

func UseMerkle(header *hexary.MerkleHeader, lastVote []byte) test.FixtureOption {
	return test.UseConfig(&test.FixtureConfig{
		MerkleRoot:   header.RootHash,
		MerkleLeaves: header.Leaves,
		MerkleLastVotes: lastVote,
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
		NewPlatform: func(ctx *test.NodeContext) base.Platform {
			var bv *blockv0.BlockVoteList
			var err error
			if ctx.Config.MerkleLastVotes != nil {
				bv, err = blockv0.NewBlockVotesFromBytes(ctx.Config.MerkleLastVotes)
				assert.NoError(ctx.Config.T, err)
			}
			return &platform{
				basic.Platform,
				hexary.MerkleHeader{
					RootHash: ctx.Config.MerkleRoot,
					Leaves: ctx.Config.MerkleLeaves,
				},
				bv,
			}
		},
		NewBM: func(ctx *test.NodeContext) module.BlockManager {
			c := ctx.C
			handlers := []base.BlockHandler{
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
			iplt := ctx.Platform.(lcimporter.BlockV1ProofStorage)
			header, lastVotes, err := iplt.GetBlockV1Proof()
			assert.NoError(t, err)
			wal := path.Join(ctx.Base, "wal")
			wm := consensus.NewTestWAL()
			cs, err := icconsensus.New(
				ctx.C,
				wal,
				wm,
				nil,
				header,
				lastVotes,
			)
			assert.NoError(t, err)
			return cs
		},
	})
}
