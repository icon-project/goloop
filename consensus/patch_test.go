/*
 * Copyright 2022 ICON Foundation
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

package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

type skipPatchTest struct {
	Assert *assert.Assertions
}

func newSkipPatchTest(t *testing.T) *skipPatchTest {
	return &skipPatchTest{
		Assert: assert.New(t),
	}
}

func (t *skipPatchTest) newSignedVote(w module.Wallet, bid int32, round int32) *VoteMessage {
	vm := newVoteMessage()
	vm.BlockID = codec.MustMarshalToBytes(bid)
	vm.Round = round
	err := vm.sign(w)
	t.Assert.NoError(err)
	return vm
}

func (t *skipPatchTest) newSignedVote2(w module.Wallet, bid int32, height int64, round int32) *VoteMessage {
	vm := newVoteMessage()
	vm.BlockID = codec.MustMarshalToBytes(bid)
	vm.Height = height
	vm.Round = round
	err := vm.sign(w)
	t.Assert.NoError(err)
	return vm
}

type validatorList []module.Address

func (v validatorList) Hash() []byte {
	//TODO implement me
	panic("implement me")
}

func (v validatorList) Bytes() []byte {
	//TODO implement me
	panic("implement me")
}

func (v validatorList) Flush() error {
	//TODO implement me
	panic("implement me")
}

func (v validatorList) IndexOf(address module.Address) int {
	for i := 0; i < len(v); i++ {
		if v[i].Equal(address) {
			return i
		}
	}
	return -1
}

func (v validatorList) Len() int {
	return len(v)
}

func (v validatorList) Get(i int) (module.Validator, bool) {
	//TODO implement me
	panic("implement me")
}

func TestSkipPatch_Verify(t_ *testing.T) {
	const nid = 7
	t := newSkipPatchTest(t_)
	w := make([]module.Wallet, 0, 4)
	valList := validatorList(make([]module.Address, 0, 4))
	for i := 0; i < 4; i++ {
		w = append(w, wallet.New())
		valList = append(valList, w[i].Address())
	}

	vs := newVoteSet(4)
	vl := vs.voteList()
	sp := newSkipPatch(vl)
	err := sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], nid, 1))
	vs.add(1, t.newSignedVote(w[1], nid, 1))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	ww := wallet.New()
	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(ww, nid, 10))
	vs.add(1, t.newSignedVote(w[1], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], nid, 10))
	vs.add(1, t.newSignedVote(w[0], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vm := t.newSignedVote(w[0], nid, 10)
	// modify message and sign again
	vm.BlockPartSetIDAndNTSVoteCount = &PartSetIDAndAppData{
		CountWord: 1,
		Hash:      make([]byte, 32),
	}
	err = vm.sign(w[0])
	t.Assert.NoError(err)
	vs.add(0, vm)
	vs.add(1, t.newSignedVote(w[1], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], 9, 10))
	vs.add(1, t.newSignedVote(w[1], 9, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], nid, 10))
	vs.add(1, t.newSignedVote(w[1], nid, 11))
	vs.add(2, t.newSignedVote(w[2], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.Error(err)

	vs = newVoteSet(4)
	vs.add(0, t.newSignedVote(w[0], nid, 10))
	vs.add(1, t.newSignedVote(w[1], nid, 10))
	vl = vs.voteList()
	sp = newSkipPatch(vl)
	err = sp.Verify(valList, 10, nid)
	t.Assert.NoError(err)
}

func TestSkipPatch_Basics(t_ *testing.T) {
	const nid = 7
	t := newSkipPatchTest(t_)
	w := make([]module.Wallet, 0, 4)
	valList := validatorList(make([]module.Address, 0, 4))
	for i := 0; i < 4; i++ {
		w = append(w, wallet.New())
		valList = append(valList, w[i].Address())
	}

	vs := newVoteSet(4)
	vs.add(0, t.newSignedVote2(w[0], nid, 5, 10))
	vs.add(1, t.newSignedVote2(w[1], nid, 5, 10))
	vl := vs.voteList()
	sp := newSkipPatch(vl)
	t.Assert.NoError(sp.Verify(valList, 10, nid))
	t.Assert.EqualValues(4, sp.Height())
	t.Assert.Equal(module.PatchTypeSkipTransaction, sp.Type())

	bs := sp.Data()
	p, err := DecodePatch(module.PatchTypeSkipTransaction, bs)
	t.Assert.NoError(err)
	sp2, ok := p.(module.SkipTransactionPatch)
	t.Assert.True(ok)
	t.Assert.NoError(sp2.Verify(valList, 10, nid))
	t.Assert.EqualValues(4, sp2.Height())
	t.Assert.Equal(module.PatchTypeSkipTransaction, sp2.Type())
	t.Assert.Equal(bs, sp2.Data())
}
