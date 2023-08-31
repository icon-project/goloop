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
	"github.com/icon-project/goloop/common/cache"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
)

type bpmKey struct {
	Hash  string
	Index uint16
}

type bpmCache struct {
	c cache.CosterLRU[bpmKey, *BlockPartMessage]
}

func makeBPMCache(cap int) bpmCache {
	return bpmCache{
		c: cache.MakeCosterLRU[bpmKey, *BlockPartMessage](cap),
	}
}

func (c *bpmCache) Put(msg *BlockPartMessage) error {
	var pb partBinary
	if _, err := codec.UnmarshalFromBytes(msg.BlockPart, &pb); err != nil {
		return err
	}
	hash := crypto.SHA3Sum256(pb.Proof[0])
	c.c.Put(bpmKey{
		string(hash), msg.Index,
	}, msg)
	return nil
}

func (c *bpmCache) Get(hash []byte, idx uint16) *BlockPartMessage {
	msg, _ := c.c.Get(bpmKey{string(hash), idx})
	return msg
}
