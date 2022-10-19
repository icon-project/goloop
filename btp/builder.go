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
	"sort"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/module"
)

type SectionBuilder interface {
	SendMessage(nid int64, msg []byte)
	EnsureSection(nid int64)
	NotifyInactivated(ntid int64)
	Build() (module.BTPSection, error)
}

// NewSectionBuilder returns new SectionBuilder. view shall have the final value
// for a transition except Network's PrevNetworkSectionHash and
// LastNetworkSectionHash fields. The two fields shall have initial value
// for the transition.
func NewSectionBuilder(view StateView) SectionBuilder {
	return &sectionBuilder{
		view:           view,
		networkEntries: make(map[int64]*networkEntry),
	}
}

type networkEntry struct {
	messages [][]byte
}

type sectionBuilder struct {
	view                    StateView
	inactivatedNetworkTypes []int64
	networkEntries          map[int64]*networkEntry
}

func (sb *sectionBuilder) SendMessage(nid int64, msg []byte) {
	ne, ok := sb.networkEntries[nid]
	if !ok {
		ne = &networkEntry{}
		sb.networkEntries[nid] = ne
	}
	ne.messages = append(ne.messages, msg)
}

func (sb *sectionBuilder) EnsureSection(nid int64) {
	_, ok := sb.networkEntries[nid]
	if !ok {
		sb.networkEntries[nid] = &networkEntry{}
	}
}

func (sb *sectionBuilder) NotifyInactivated(ntid int64) {
	sb.inactivatedNetworkTypes = append(sb.inactivatedNetworkTypes, ntid)
}

func (sb *sectionBuilder) Build() (module.BTPSection, error) {
	nsMap := make(map[int64]networkSectionSlice, len(sb.networkEntries))
	npcChanged := make(map[int64]bool)
	for nid, ne := range sb.networkEntries {
		nw, err := sb.view.GetNetworkView(nid)
		if err != nil {
			return nil, err
		}
		ntid := nw.NetworkTypeID()
		nt, err := sb.view.GetNetworkTypeView(ntid)
		if err != nil {
			return nil, err
		}
		ns := newNetworkSection(nid, nw, ne, ntm.ForUID(nt.UID()))
		nsMap[ntid] = nsMap[ntid].SortedInsert(ns)
		if ns.NextProofContextChanged() {
			npcChanged[ntid] = true
		}
	}
	ntsSlice := networkTypeSectionSlice(make([]module.NetworkTypeSection, 0, len(nsMap)))
	for ntid, nsSlice := range nsMap {
		nt, err := sb.view.GetNetworkTypeView(ntid)
		if err != nil {
			return nil, err
		}
		nts, err := newNetworkTypeSection(ntid, nt, nsSlice, npcChanged[ntid])
		if err != nil {
			return nil, err
		}
		ntsSlice = ntsSlice.SortedInsert(nts)
	}
	return newBTPSection(ntsSlice, sb.inactivatedNetworkTypes), nil
}

type networkTypeSectionSlice []module.NetworkTypeSection

func (ntss networkTypeSectionSlice) Len() int {
	return len(ntss)
}

func (ntss networkTypeSectionSlice) Get(i int) []byte {
	return ntss[i].Hash()
}

func (ntss networkTypeSectionSlice) SortedInsert(
	nts module.NetworkTypeSection,
) networkTypeSectionSlice {
	i := sort.Search(len(ntss), func(i int) bool {
		return ntss[i].NetworkTypeID() >= nts.NetworkTypeID()
	})
	if i == len(ntss) {
		return append(ntss, nts)
	}
	ntss = append(ntss[:i+1], ntss[i:]...)
	ntss[i] = nts
	return ntss
}

func (ntss networkTypeSectionSlice) Search(ntid int64) module.NetworkTypeSection {
	i := sort.Search(len(ntss), func(i int) bool {
		return ntss[i].NetworkTypeID() >= ntid
	})
	if i < len(ntss) && ntss[i].NetworkTypeID() == ntid {
		return ntss[i]
	}
	return nil
}

type networkSectionSlice []module.NetworkSection

func (nss networkSectionSlice) Len() int {
	return len(nss)
}

func (nss networkSectionSlice) Get(i int) []byte {
	return nss[i].Hash()
}

func (nss networkSectionSlice) SortedInsert(
	ns module.NetworkSection,
) networkSectionSlice {
	i := sort.Search(len(nss), func(i int) bool {
		return nss[i].NetworkID() >= ns.NetworkID()
	})
	if i == len(nss) {
		return append(nss, ns)
	}
	nss = append(nss[:i+1], nss[i:]...)
	nss[i] = ns
	return nss
}

func (nss networkSectionSlice) Search(nid int64) (module.NetworkSection, int) {
	i := sort.Search(len(nss), func(i int) bool {
		return nss[i].NetworkID() >= nid
	})
	if i < len(nss) && nss[i].NetworkID() == nid {
		return nss[i], i
	}
	return nil, -1
}
