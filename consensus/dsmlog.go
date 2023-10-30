/*
 * Copyright 2023 ICON Foundation
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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/cache"
	"github.com/icon-project/goloop/module"
)

type dsmCacheKey struct {
	Type     string   // DST*
	VoteType VoteType // 0 for proposal
	Address  common.Address
	Height   int64
	Round    int32
}

type dsmLog struct {
	c cache.CosterRandom[dsmCacheKey, cache.Coster]
}

func makeDSMLog(cap int) dsmLog {
	return dsmLog{
		c: cache.MakeCosterRandom[dsmCacheKey, cache.Coster](cap, nil),
	}
}

func (c *dsmLog) putVoteMessage(msg *VoteMessage) {
	c.c.Put(dsmCacheKey{
		module.DSTVote, msg.Type, *msg.address(),
		msg.Height, msg.Round,
	}, msg)
}

func (c *dsmLog) putProposalMessage(msg *ProposalMessage) {
	c.c.Put(dsmCacheKey{
		module.DSTProposal, 0, *msg.address(),
		msg.Height, msg.Round,
	}, msg)
}

func (c *dsmLog) getVoteMessage(vt VoteType, addr common.Address, h int64, r int32) *VoteMessage {
	val, _ := c.c.Get(dsmCacheKey{
		module.DSTVote, vt, addr, h, r,
	})
	if val == nil {
		return nil
	}
	return val.(*VoteMessage)
}

func (c *dsmLog) getProposalMessage(addr common.Address, h int64, r int32) *ProposalMessage {
	val, _ := c.c.Get(dsmCacheKey{
		module.DSTProposal, 0, addr, h, r,
	})
	if val == nil {
		return nil
	}
	return val.(*ProposalMessage)
}

func (c *dsmLog) LogAndCheckVoteMessage(msg *VoteMessage) []module.DoubleSignData {
	omsg := c.getVoteMessage(msg.Type, *msg.address(), msg.Height, msg.Round)
	if omsg != nil {
		dsv1 := dsVote{omsg}
		dsv2 := dsVote{msg}
		if dsv1.IsConflictWith(&dsv2) {
			return []module.DoubleSignData{&dsv1, &dsv2}
		}
	}
	c.putVoteMessage(msg)
	return nil
}

func (c *dsmLog) LogAndCheckProposalMessage(msg *ProposalMessage) []module.DoubleSignData {
	omsg := c.getProposalMessage(*msg.address(), msg.Height, msg.Round)
	if omsg != nil {
		dsv1 := dsProposal{omsg}
		dsv2 := dsProposal{msg}
		if dsv1.IsConflictWith(&dsv2) {
			return []module.DoubleSignData{&dsv1, &dsv2}
		}
		return nil
	}
	c.putProposalMessage(msg)
	return nil
}
