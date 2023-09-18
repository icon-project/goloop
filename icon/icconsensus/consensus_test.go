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

package icconsensus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/ictest"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func TestConsensus_BasicsWithAccumulator(t *testing.T) {
	gen := test.NewNode(t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1)
	defer gen.Close()

	const height = 10
	for i:=1; i<height; i++ {
		gen.ProposeFinalizeBlock((*blockv0.BlockVoteList)(nil))
	}
	header := ictest.NodeFinalizeMerkle(gen)

	gen = test.NewNode(
		t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1,
		ictest.UseMerkle(header, nil), test.UseDB(gen.Chain.Database()),
	)
	defer gen.Close()

	var err error
	for i:=0; i<height; i++ {
		_, err = gen.BM.GetBlockByHeight(int64(i))
		assert.NoError(t, err)
	}

	f := test.NewNode(
		t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1,
		ictest.UseMerkle(header, nil),
		test.SetTimeoutPropose(4*time.Second),
	)
	defer f.Close()

	err = gen.CS.Start()
	assert.NoError(t, err)

	err = f.CS.Start()
	assert.NoError(t, err)

	f.NM.Connect(gen.NM)

	chn, err := f.BM.WaitForBlock(height-1)
	assert.NoError(t, err)
	blk := <-chn
	assert.EqualValues(t, height-1, blk.Height())
	assert.EqualValues(t, height, f.CS.GetStatus().Height)
}

func TestConsensus_UpgradeWithAccumulator(t *testing.T) {
	gen := test.NewNode(t, ictest.UseBMForBlockV1)
	defer gen.Close()

	nilVotes := (*blockv0.BlockVoteList)(nil)
	gen.ProposeFinalizeBlockWithTX(
		nilVotes,
		test.NewTx().SetValidators(gen.Chain.Wallet().Address()).String(),
	)
	gen.ProposeFinalizeBlock(nilVotes)
	nextBlockVersion := int32(module.BlockVersion2)
	gen.ProposeFinalizeBlockWithTX(
		ictest.NodeNewVoteListV1ForLastBlock(gen),
		test.NewTx().SetNextBlockVersion(&nextBlockVersion).String(),
	)

	wallets := make([]module.Wallet, 3)
	for i := range wallets {
		wallets[i] = wallet.New()
	}
	gen.ProposeFinalizeBlockWithTX(
		ictest.NodeNewVoteListV1ForLastBlock(gen),
		test.NewTx().SetValidatorsAddresser(
			gen.Chain.Wallet(), wallets[0], wallets[1], wallets[2],
		).String(),
	)
	header := ictest.NodeFinalizeMerkle(gen)
	lastVotes := ictest.NodeNewVoteListV1ForLastBlock(gen)

	f := test.NewFixture(
		t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1,
		ictest.UseMerkle(header, lastVotes.Bytes()),
		test.AddDefaultNode(false),
		test.SetTimeoutPropose(4*time.Second),
	)
	defer f.Close()

	f.AddNode(
		test.UseWallet(gen.Chain.Wallet()), test.UseDB(gen.Chain.Database()),
	)
	for _, w := range wallets {
		f.AddNode(test.UseWallet(w))
	}
	test.NodeInterconnect(f.Nodes)

	for _, n := range f.Nodes {
		err := n.CS.Start()
		assert.NoError(t, err)
	}
	chn, err := f.BM.WaitForBlock(10)
	assert.NoError(t, err)
	blk := <-chn
	assert.EqualValues(t, 10, blk.Height())
	assert.EqualValues(t, 11, f.CS.GetStatus().Height)
}
