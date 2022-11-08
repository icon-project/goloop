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
	"github.com/icon-project/goloop/common/atomic"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

// NewSection returns a new Section. view shall have the final value for a
// transition.
func NewSection(
	digest module.BTPDigest,
	view StateView,
	dbase db.Database,
) (module.BTPSection, error) {
	return &btpSectionFromDigest{
		digest:                digest,
		view:                  view,
		dbase:                 dbase,
		networkTypeSectionFor: make(map[int64]*networkTypeSectionFromDigest),
	}, nil
}

type btpSectionFromDigest struct {
	digest module.BTPDigest
	view   StateView
	dbase  db.Database

	mu                    sync.Mutex
	networkTypeSectionFor map[int64]*networkTypeSectionFromDigest
	networkTypeSections   atomic.Cache[[]module.NetworkTypeSection]
}

func (bs *btpSectionFromDigest) Digest() module.BTPDigest {
	return bs.digest
}

func (bs *btpSectionFromDigest) NetworkTypeSections() []module.NetworkTypeSection {
	return bs.networkTypeSections.Get(func() []module.NetworkTypeSection {
		ntdSlice := bs.digest.NetworkTypeDigests()
		ntsSlice := make([]module.NetworkTypeSection, 0, len(ntdSlice))
		for _, ntd := range ntdSlice {
			nts, err := bs.NetworkTypeSectionFor(ntd.NetworkTypeID())
			log.Must(err)
			ntsSlice = append(ntsSlice, nts)
		}
		return ntsSlice
	})
}

func (bs *btpSectionFromDigest) NetworkTypeSectionFor(ntid int64) (module.NetworkTypeSection, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.networkTypeSectionFor[ntid] == nil {
		nt, err := bs.view.GetNetworkTypeView(ntid)
		if err != nil {
			return nil, err
		}
		mod := ntm.ForUID(nt.UID())
		npc, err := mod.NewProofContextFromBytes(nt.NextProofContext())
		if err != nil {
			return nil, err
		}
		ntd := bs.digest.NetworkTypeDigestFor(ntid)
		if ntd == nil {
			return nil, errors.Errorf("not found ntid=%d", ntid)
		}
		nts := &networkTypeSectionFromDigest{
			view:             bs.view,
			dbase:            bs.dbase,
			mod:              mod,
			nt:               nt,
			ntd:              ntd,
			nextProofContext: npc,
		}
		bs.networkTypeSectionFor[ntid] = nts
	}
	return bs.networkTypeSectionFor[ntid], nil
}

type networkTypeSectionFromDigest struct {
	// immutables
	view             StateView
	dbase            db.Database
	mod              module.NetworkTypeModule
	nt               NetworkTypeView
	ntd              module.NetworkTypeDigest
	nextProofContext module.BTPProofContext
}

func (nts *networkTypeSectionFromDigest) NetworkTypeID() int64 {
	return nts.ntd.NetworkTypeID()
}

func (nts *networkTypeSectionFromDigest) Hash() []byte {
	return nts.ntd.NetworkTypeSectionHash()
}

func (nts *networkTypeSectionFromDigest) NetworkSectionsRoot() []byte {
	return nts.ntd.NetworkSectionsRootWithMod(nts.mod)
}

func (nts *networkTypeSectionFromDigest) NextProofContext() module.BTPProofContext {
	return nts.nextProofContext
}

func (nts *networkTypeSectionFromDigest) NetworkSectionFor(nid int64) (module.NetworkSection, error) {
	nw, err := nts.view.GetNetworkView(nid)
	if err != nil {
		return nil, err
	}
	nd := nts.ntd.NetworkDigestFor(nid)
	if nd == nil {
		return nil, errors.Errorf("not found nid=%d", nid)
	}
	ns, err := newNetworkSectionFromDigest(nts.dbase, nts.mod, nw, nd)
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func (nts *networkTypeSectionFromDigest) NewDecision(
	srcNetworkUID []byte,
	height int64,
	round int32,
) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           srcNetworkUID,
		DstType:                nts.ntd.NetworkTypeID(),
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: nts.Hash(),
		mod:                    nts.mod,
	}
}

func (nts *networkTypeSectionFromDigest) NetworkSectionToRoot(nid int64) ([]module.MerkleNode, error) {
	return nts.ntd.NetworkSectionToRootWithMod(nts.mod, nid)
}

type networkSectionFromDigest struct {
	// immutables
	nw           NetworkView
	nd           module.NetworkDigest
	updateNumber int64
	messageCount int64
}

func newNetworkSectionFromDigest(
	dbase db.Database,
	mod module.NetworkTypeModule,
	nw NetworkView,
	nd module.NetworkDigest,
) (*networkSectionFromDigest, error) {
	ml, err := nd.MessageList(dbase, mod)
	if err != nil {
		return nil, err
	}
	updateNumber := (nw.NextMessageSN() - ml.Len()) << 1
	if nw.NextProofContextChanged() {
		updateNumber |= 1
	}
	return &networkSectionFromDigest{
		nw:           nw,
		nd:           nd,
		updateNumber: updateNumber,
		messageCount: ml.Len(),
	}, nil
}

func (ns *networkSectionFromDigest) Hash() []byte {
	return ns.nw.LastNetworkSectionHash()
}

func (ns *networkSectionFromDigest) NetworkID() int64 {
	return ns.nd.NetworkID()
}

func (ns *networkSectionFromDigest) UpdateNumber() int64 {
	return ns.updateNumber
}

func (ns *networkSectionFromDigest) FirstMessageSN() int64 {
	return ns.nw.NextMessageSN() - ns.MessageCount()
}

func (ns *networkSectionFromDigest) NextProofContextChanged() bool {
	return ns.nw.NextProofContextChanged()
}

func (ns *networkSectionFromDigest) PrevHash() []byte {
	return ns.nw.PrevNetworkSectionHash()
}

func (ns *networkSectionFromDigest) MessageCount() int64 {
	return ns.messageCount
}

func (ns *networkSectionFromDigest) MessagesRoot() []byte {
	return ns.nd.MessagesRoot()
}
