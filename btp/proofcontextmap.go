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

package btp

import (
	"sync"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type proofContextMap struct {
	mu    sync.Mutex
	pcMap map[int64]module.BTPProofContext
}

func (m *proofContextMap) ProofContextFor(ntid int64) (module.BTPProofContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pc, ok := m.pcMap[ntid]
	if !ok {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found ntid=%d", ntid)
	}
	return pc, nil
}

func (m *proofContextMap) copy() *proofContextMap {
	res := &proofContextMap{
		pcMap: make(map[int64]module.BTPProofContext),
	}
	for k, v := range m.pcMap {
		res.pcMap[k] = v
	}
	return res
}

func (m *proofContextMap) Update(src module.BTPProofContextMapUpdateSource) (module.BTPProofContextMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := m
	btpSection, err := src.BTPSection()
	if err != nil {
		return nil, err
	}
	if btpSectionByBuilder, ok := btpSection.(*btpSectionByBuilder); ok {
		for _, nts_ := range btpSectionByBuilder.NetworkTypeSections() {
			nts := nts_.(*networkTypeSectionByBuilder)
			if nts.nsNPCChanged {
				if res == m {
					res = m.copy()
				}
				res.pcMap[nts.NetworkTypeID()] = nts.NextProofContext()
			}
		}
		for _, ntid := range btpSectionByBuilder.inactivatedNTs {
			if _, ok := res.pcMap[ntid]; ok {
				if res == m {
					res = m.copy()
				}
				delete(res.pcMap, ntid)
			}
		}
		return res, nil
	}
	npcm, err := src.NextProofContextMap()
	if err != nil {
		return nil, err
	}
	return npcm, nil
}

func (m *proofContextMap) Verify(
	srcUID []byte,
	height int64,
	round int32,
	bd module.BTPDigest,
	ntsdProves module.NTSDProofList,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cnt := 0
	for _, ntd := range bd.NetworkTypeDigests() {
		if _, ok := m.pcMap[ntd.NetworkTypeID()]; ok {
			cnt++
		}
	}
	if cnt != ntsdProves.NTSDProofCount() {
		return errors.Errorf(
			"invalid len networkTypeLen=%d expProvesLen=%d provesLen=%d height=%d round=%d",
			len(bd.NetworkTypeDigests()), cnt, ntsdProves.NTSDProofCount(),
			height, round,
		)
	}
	i := 0
	for _, ntd := range bd.NetworkTypeDigests() {
		ntid := ntd.NetworkTypeID()
		pc, ok := m.pcMap[ntid]
		if !ok {
			continue
		}
		d := pc.NewDecision(srcUID, ntid, height, round, ntd.NetworkTypeSectionHash())
		proof, err := pc.NewProofFromBytes(ntsdProves.NTSDProofAt(i))
		if err != nil {
			return errors.Wrapf(
				err, "new proof fail voteIndex=%d ntid=%d height=%d round=%d",
				i, ntid, height, round,
			)
		}
		err = pc.Verify(d.Hash(), proof)
		if err != nil {
			return errors.Wrapf(
				err, "verify fail voteIndex=%d ntid=%d height=%d round=%d",
				i, ntid, height, round,
			)
		}
		i++
	}
	return nil
}

func NewProofContextMap(view StateView) (module.BTPProofContextMap, error) {
	res := &proofContextMap{
		pcMap: make(map[int64]module.BTPProofContext),
	}
	ntidSlice, err := view.GetNetworkTypeIDs()
	if err != nil {
		return nil, err
	}
	for _, ntid := range ntidSlice {
		nt, err := view.GetNetworkTypeView(ntid)
		if err != nil {
			return nil, err
		}
		mod := ntm.ForUID(nt.UID())
		pcBytes := nt.NextProofContext()
		if pcBytes != nil {
			pc, err := mod.NewProofContextFromBytes(nt.NextProofContext())
			if err != nil {
				return nil, err
			}
			res.pcMap[ntid] = pc
		}
	}
	return res, nil
}

var ZeroProofContextMap = &proofContextMap{
	pcMap: make(map[int64]module.BTPProofContext),
}
