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

package block

import (
	"github.com/icon-project/goloop/module"
)

type btpBlockHeaderFormat struct {
	MainHeight             int64
	Round                  int32
	NextProofContextHash   []byte
	NetworkSectionToRoot   []module.MerkleNode
	NetworkID              int64
	UpdateNumber           int64
	PrevNetworkSectionHash []byte
	MessageCount           int64
	MessagesRoot           []byte
	Proof                  []byte
	NextProofContext       []byte
}

type btpBlockFormat struct {
	btpBlockHeaderFormat
	Messages [][]byte
}

type btpBlock struct {
	btpBlockFormat
}

func NewBTPBlock(
	mainHeight int64,
	round int32,
	nts module.NetworkTypeSection,
	nid int64,
	proof []byte,
) (*btpBlock, error) {
	bb := &btpBlock{}
	bb.btpBlockFormat.MainHeight = mainHeight
	bb.btpBlockFormat.Round = round
	bb.btpBlockFormat.NextProofContextHash = nts.NextProofContext().Hash()
	pf, err := nts.NetworkSectionToRoot(nid)
	if err != nil {
		return nil, err
	}
	bb.btpBlockFormat.NetworkSectionToRoot = pf
	bb.btpBlockFormat.NetworkID = nid
	ns, err := nts.NetworkSectionFor(nid)
	if err != nil {
		return nil, err
	}
	bb.btpBlockFormat.UpdateNumber = ns.UpdateNumber()
	bb.btpBlockFormat.PrevNetworkSectionHash = ns.PrevHash()
	bb.btpBlockFormat.MessageCount = ns.MessageCount()
	bb.btpBlockFormat.MessagesRoot = ns.MessagesRoot()
	bb.btpBlockFormat.Proof = proof
	bb.btpBlockFormat.NextProofContext = nts.NextProofContext().Bytes()
	return bb, nil
}
