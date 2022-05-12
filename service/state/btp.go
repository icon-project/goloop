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

package state

import (
	"container/list"
	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	ActiveNetworkTypeIDsKey = "activeNetworkTypeIDs"
	NetworkTypeIDKey        = "networkTypeID"
	NetworkTypeIDByUIDKey   = "networkTypeIDByUID"
	NetworkTypeByIDKey      = "networkTypeByID"
	NetworkIDKey            = "networkID"
	NetworkByIDKey          = "networkByID"
)

type BTPContext interface {
	btp.StateView
	Store() containerdb.BytesStoreState
	GetNetworkTypeIdByName(name string) int64
}

type BTPSnapshot interface {
	Bytes() []byte
	Flush() error
	NewState() BTPState
}

type BTPState interface {
	GetSnapshot() BTPSnapshot
	Reset(snapshot BTPSnapshot)
	BuildAndApplySection(bc BTPContext, btpMsgs list.List) error
}

type btpContext struct {
	store containerdb.BytesStoreState
}

func (bc *btpContext) Store() containerdb.BytesStoreState {
	return bc.store
}

func (bc *btpContext) GetNetworkTypeIDs() ([]int64, error) {
	ret, _, err := bc.getNetworkTypeIDs()
	return ret, err
}

func (bc *btpContext) GetNetworkView(nid int64) (btp.NetworkView, error) {
	ret, _, err := bc.getNetwork(nid)
	return ret, err
}

func (bc *btpContext) GetNetworkTypeView(ntid int64) (btp.NetworkTypeView, error) {
	ret, _, err := bc.getNetworkType(ntid)
	return ret, err
}

func (bc *btpContext) GetNetworkTypeIdByName(name string) int64 {
	if ntm.ForUID(name) == nil {
		return -1
	}
	ret, _ := bc.getNetworkTypeIdByName(name)
	return ret
}

func (bc *btpContext) getNetwork(nid int64) (*btp.Network, *containerdb.DictDB, error) {
	dbase := scoredb.NewDictDB(bc.store, NetworkByIDKey, 1)
	if value := dbase.Get(nid); value == nil {
		return nil, nil, errors.Errorf("There is no network for %d", nid)
	} else {
		if nw, err := btp.NewNetworkFromBytes(value.Bytes()); err != nil {
			return nil, nil, err
		} else {
			return nw, dbase, nil
		}
	}
}

func (bc *btpContext) getNetworkType(ntid int64) (*btp.NetworkType, *containerdb.DictDB, error) {
	dbase := scoredb.NewDictDB(bc.store, NetworkTypeByIDKey, 1)
	if value := dbase.Get(ntid); value == nil {
		return nil, dbase, errors.Errorf("No network type for %d", ntid)
	} else {
		return btp.NewNetworkTypeFromBytes(value.Bytes()), dbase, nil
	}
}

func (bc *btpContext) getNetworkTypeIdByName(name string) (int64, *containerdb.DictDB) {
	dbase := scoredb.NewDictDB(bc.store, NetworkTypeIDByUIDKey, 1)
	if value := dbase.Get(name); value == nil {
		return 0, dbase
	} else {
		return value.Int64(), dbase
	}
}

func (bc *btpContext) getNetworkTypeIDs() ([]int64, *containerdb.ArrayDB, error) {
	dbase := scoredb.NewArrayDB(bc.store, ActiveNetworkTypeIDsKey)
	ids := make([]int64, 0)
	for i := 0; i < dbase.Size(); i++ {
		ids = append(ids, dbase.Get(i).Int64())
	}
	return ids, dbase, nil
}

func (bc *btpContext) getNewNetworkTypeID() (int64, *containerdb.VarDB) {
	dbase := scoredb.NewVarDB(bc.store, NetworkTypeIDKey)
	return dbase.Int64() + 1, dbase
}

func (bc *btpContext) getNewNetworkID() (int64, *containerdb.VarDB) {
	dbase := scoredb.NewVarDB(bc.store, NetworkIDKey)
	return dbase.Int64() + 1, dbase
}

func NewBTPContext(store containerdb.BytesStoreState) BTPContext {
	return &btpContext{store: store}
}

type btpData struct {
	readonly         bool
	validatorChanged map[int64]bool // for network type
	networkModified  map[int64]bool // for network
	digestHash       []byte
}

func (b *btpData) setValidatorChanged(ntid int64) {
	if b.validatorChanged == nil {
		b.validatorChanged = make(map[int64]bool)
	}
	b.validatorChanged[ntid] = true
}

func (b *btpData) setNetworkModified(nid int64) {
	if b.networkModified == nil {
		b.networkModified = make(map[int64]bool)
	}
	b.networkModified[nid] = true
}

type btpSnapshot struct {
	btpData
	store containerdb.BytesStoreSnapshot
}

func (bs *btpSnapshot) Bytes() []byte {
	return bs.digestHash
}

func (bs *btpSnapshot) Flush() error {
	return nil
}

func (bs *btpSnapshot) NewState() BTPState {
	state := new(BTPStateImpl)
	state.readonly = true
	state.digestHash = bs.digestHash
	if bs.validatorChanged != nil {
		state.validatorChanged = make(map[int64]bool)
		for k, v := range bs.validatorChanged {
			state.validatorChanged[k] = v
		}
	}
	if bs.validatorChanged != nil {
		state.networkModified = make(map[int64]bool)
		for k, v := range bs.networkModified {
			state.networkModified[k] = v
		}
	}
	return state
}

func NewBTPSnapshot(hash []byte) BTPSnapshot {
	ss := new(btpSnapshot)
	ss.digestHash = hash
	return ss
}

type BTPStateImpl struct {
	btpData
}

func (bs *BTPStateImpl) GetSnapshot() BTPSnapshot {
	ss := new(btpSnapshot)
	ss.readonly = true
	if bs.validatorChanged != nil {
		ss.validatorChanged = make(map[int64]bool)
		for k, v := range bs.validatorChanged {
			ss.validatorChanged[k] = v
		}
	}
	if bs.networkModified != nil {
		ss.networkModified = make(map[int64]bool)
		for k, v := range bs.networkModified {
			ss.networkModified[k] = v
		}
	}
	ss.digestHash = bs.digestHash
	return ss
}

func (bs *BTPStateImpl) Reset(snapshot BTPSnapshot) {
	ss, ok := snapshot.(*btpSnapshot)
	if !ok {
		return
	}
	if ss.networkModified != nil {
		bs.networkModified = make(map[int64]bool)
		for k, v := range ss.networkModified {
			bs.networkModified[k] = v
		}
	}
}

func (bs *BTPStateImpl) OpenNetwork(
	bc BTPContext, networkTypeName string, name string, owner module.Address,
) (ntid int64, nid int64, err error) {
	mod := ntm.ForUID(networkTypeName)
	if mod == nil {
		err = scoreresult.InvalidParameterError.Errorf("Not supported BTP network type %s", networkTypeName)
		return
	}
	var varDB *containerdb.VarDB
	var nt *btp.NetworkType
	bci := bc.(*btpContext)
	ntid, ntidDB := bci.getNetworkTypeIdByName(networkTypeName)
	if ntid == 0 {
		ntid, varDB = bci.getNewNetworkTypeID()
		if err = varDB.Set(ntid); err != nil {
			return
		}
		if err = ntidDB.Set(networkTypeName, ntid); err != nil {
			return
		}

		// TODO make ProofContext from NextValidators
		//pc, err := mod.NewProofContext(keys)
		//if err != nil {
		//	return
		//}
		nt = btp.NewNetworkType(networkTypeName, nil)
	} else {
		if nt, _, err = bci.getNetworkType(ntid); err != nil {
			return
		}
	}

	store := bc.Store()
	nid, varDB = bci.getNewNetworkID()
	if err = varDB.Set(nid); err != nil {
		return
	}

	nw := btp.NewNetwork(ntid, name, owner, true)
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	if err = nwDB.Set(nid, nw.Bytes()); err != nil {
		return
	}

	nt.AddOpenNetworkID(nid)
	ntDB := scoredb.NewDictDB(store, NetworkTypeByIDKey, 1)
	if err = ntDB.Set(ntid, nt.Bytes()); err != nil {
		return
	}

	// TODO uncomment after implementing setPublicKey
	//bs.setNetworkModified(nid)
	return
}

func (bs *BTPStateImpl) CloseNetwork(bc BTPContext, nid int64) (int64, error) {
	store := bc.Store()
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	nwValue := nwDB.Get(nid)
	if nwValue == nil {
		return 0, scoreresult.InvalidParameterError.Errorf("There is no network for %d", nid)
	}
	nw, err := btp.NewNetworkFromBytes(nwValue.Bytes())
	if err != nil {
		return 0, err
	}
	nw.SetOpen(false)
	if err := nwDB.Set(nid, nw.Bytes()); err != nil {
		return 0, err
	}

	ntDB := scoredb.NewDictDB(store, NetworkTypeByIDKey, 1)
	if ntValue := ntDB.Get(nw.NetworkTypeID()); ntValue == nil {
		return 0, scoreresult.InvalidInstanceError.Errorf("There is no network type for %d", nw.NetworkTypeID())
	} else {
		nt := btp.NewNetworkTypeFromBytes(ntValue.Bytes())
		if err := nt.RemoveOpenNetworkID(nid); err != nil {
			return 0, scoreresult.InvalidParameterError.Wrapf(err, "There is no open network %d in %d", nid, nw.NetworkTypeID())
		}
		if err := ntDB.Set(nw.NetworkTypeID(), nt.Bytes()); err != nil {
			return 0, err
		}
	}

	return nw.NetworkTypeID(), nil
}

func (bs *BTPStateImpl) HandleMessageSN(bc BTPContext, from module.Address, nid int64) error {
	store := bc.Store()
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	nwValue := nwDB.Get(nid)
	if nwValue == nil {
		return scoreresult.InvalidParameterError.Errorf("There is no network for %d", nid)
	}
	nw, err := btp.NewNetworkFromBytes(nwValue.Bytes())
	if err != nil {
		return err
	}
	if !from.Equal(nw.Owner()) {
		return scoreresult.AccessDeniedError.Errorf("Only owner can send BTP message")
	}
	nw.IncreaseNextMessageSN()
	if err := nwDB.Set(nid, nw); err != nil {
		return err
	}

	return nil
}

func (bs *BTPStateImpl) applyBTPSection(bc BTPContext, btpSection module.BTPSection) error {
	for _, nts := range btpSection.NetworkTypeSections() {
		ntid := nts.NetworkTypeID()
		if _, ok := bs.validatorChanged[ntid]; ok {
			if err := bs.updateProofContext(bc, ntid, nts.NextProofContext()); err != nil {
				return err
			}
		}
		for nid, _ := range bs.networkModified {
			ns, err := nts.NetworkSectionFor(nid)
			if err != nil {
				return err
			}
			if err := bs.setNetworkSectionHash(bc, ns); err != nil {
				return err
			}
		}
	}
	bs.digestHash = btpSection.Digest().Hash()
	return nil
}

func (bs *BTPStateImpl) updateProofContext(bc BTPContext, ntid int64, proof module.BTPProofContext) error {
	bci := bc.(*btpContext)
	if nt, ntDB, err := bci.getNetworkType(ntid); err != nil {
		return err
	} else {
		nt.SetNextProofContext(proof.Bytes())
		nt.SetNextProofContextHash(proof.Hash())
		if err = ntDB.Set(nt.Bytes()); err != nil {
			return err
		}
		for _, nid := range nt.OpenNetworkIDs() {
			nw, nwDB, err := bci.getNetwork(nid)
			if err != nil {
				return err
			}
			nw.SetNextProofContextChanged(true)
			if err = nwDB.Set(nw.Bytes()); err != nil {
				return err
			}
			bs.setNetworkModified(nid)
		}
		return nil
	}
}

func (bs *BTPStateImpl) setNetworkSectionHash(bc BTPContext, ns module.NetworkSection) error {
	bci := bc.(*btpContext)
	if nw, nDB, err := bci.getNetwork(ns.NetworkID()); err != nil {
		return err
	} else {
		nw.SetPrevNetworkSectionHash(nw.LastNetworkSectionHash())
		nw.SetLastNetworkSectionHash(ns.Hash())
		return nDB.Set(nw.Bytes())
	}
}

func (bs *BTPStateImpl) BuildAndApplySection(bc BTPContext, btpMsgs list.List) error {
	sb := btp.NewSectionBuilder(bc)

	for nid, _ := range bs.networkModified {
		//bci := bc.(*btpContext)
		//nw, _, err := bci.getNetwork(nid)
		//log.Tracef("BTP Ensure %d %+v %+v", nid, nw, err)
		sb.EnsureSection(nid)
	}

	for i := btpMsgs.Front(); i != nil; i = i.Next() {
		e := i.Value.(*bTPMsg)
		sb.SendMessage(e.nid, e.message)
	}

	if section, err := sb.Build(); err != nil {
		return err
	} else {
		if err = bs.applyBTPSection(bc, section); err != nil {
			return err
		}
		return nil
	}
}

func NewBTPState(hash []byte) BTPState {
	state := new(BTPStateImpl)
	state.digestHash = hash
	return state
}

type bTPMsg struct {
	nid     int64
	message []byte
}

func NewBTPMsg(nid int64, msg []byte) *bTPMsg {
	return &bTPMsg{
		nid:     nid,
		message: msg,
	}
}
