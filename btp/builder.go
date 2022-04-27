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
	Build() (module.BTPSection, error)
}

type StateView interface {
	// GetNetwork returns Network. Requirement for the fields of the Network
	// is different field by field. PrevNetworkSectionHash and
	// LastNetworkSectionHash field shall have initial value before the
	// transactions of a transition is executed. Other fields shall have
	// final value after the transactions of a transition is executed.
	GetNetwork(nid int64) (*Network, error)

	// GetNetworkType returns final value of NetworkType
	GetNetworkType(ntid int64) (*NetworkType, error)
}

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
	view           StateView
	networkEntries map[int64]*networkEntry
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

func (sb *sectionBuilder) Build() (module.BTPSection, error) {
	nsMap := make(map[int64][]module.NetworkSection, len(sb.networkEntries))
	for nid, ne := range sb.networkEntries {
		nw, err := sb.view.GetNetwork(nid)
		if err != nil {
			return nil, err
		}
		ntid := nw.NetworkTypeID
		nt, err := sb.view.GetNetworkType(ntid)
		ns := newNetworkSection(nid, nw, ne, ntm.ForUID(nt.UID))
		nsMap[ntid] = sortedInsertNS(nsMap[ntid], ns)
	}
	ntsSlice := make([]module.NetworkTypeSection, 0, len(nsMap))
	for ntid, nsSlice := range nsMap {
		nt, err := sb.view.GetNetworkType(ntid)
		if err != nil {
			return nil, err
		}
		nts := newNetworkTypeSection(ntid, nt, nsSlice)
		ntsSlice = sortedInsertNTS(ntsSlice, nts)
	}
	return &btpSection{
		networkTypeSections: ntsSlice,
	}, nil
}

func sortedInsertNTS(
	slice []module.NetworkTypeSection,
	nts *networkTypeSection,
) []module.NetworkTypeSection {
	i := sort.Search(len(slice), func(i int) bool {
		return slice[i].NetworkTypeID() >= nts.NetworkTypeID()
	})
	if i == len(slice) {
		return append(slice, nts)
	}
	slice = append(slice[:i+1], slice[i:]...)
	slice[i] = nts
	return slice
}

func sortedInsertNS(
	slice []module.NetworkSection,
	ns *networkSection,
) []module.NetworkSection {
	i := sort.Search(len(slice), func(i int) bool {
		return slice[i].NetworkID() >= ns.NetworkID()
	})
	if i == len(slice) {
		return append(slice, ns)
	}
	slice = append(slice[:i+1], slice[i:]...)
	slice[i] = ns
	return slice
}
