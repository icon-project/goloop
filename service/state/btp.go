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
	"bytes"
	"container/list"
	"encoding/base64"
	"encoding/hex"
	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
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
	PubKeyByNameKey         = "pubKeyByName"
)

type BTPContext interface {
	btp.StateView
	Store() containerdb.BytesStoreState
	BlockHeight() int64
	GetValidatorState() ValidatorState
	GetNetworkTypeIDs() ([]int64, error)
	GetNetworkTypeIDByName(name string) int64
	GetNetworkType(ntid int64) (module.BTPNetworkType, error)
	GetNetwork(nid int64) (module.BTPNetwork, error)
	GetPublicKey(address module.Address, name string, exactMatch bool) ([]byte, bool)
}

type BTPSnapshot interface {
	Bytes() []byte
	Flush() error
	NewState() BTPState
}

type BTPState interface {
	GetSnapshot() BTPSnapshot
	Reset(snapshot BTPSnapshot)
	SetValidators(vs ValidatorState)
	BuildAndApplySection(bc BTPContext, btpMsgs *list.List) (module.BTPSection, error)
}

type btpContext struct {
	wc    WorldContext
	store containerdb.BytesStoreState
}

func (bc *btpContext) Store() containerdb.BytesStoreState {
	return bc.store
}

func (bc *btpContext) BlockHeight() int64 {
	if bc.wc == nil {
		return -1
	}
	return bc.wc.BlockHeight()
}

func (bc *btpContext) GetValidatorState() ValidatorState {
	if bc.wc == nil {
		return nil
	}
	return bc.wc.GetValidatorState()
}

func (bc *btpContext) GetNetworkTypeIDs() ([]int64, error) {
	ret, _, err := bc.getNetworkTypeIDs()
	return ret, err
}

func (bc *btpContext) GetNetworkView(nid int64) (btp.NetworkView, error) {
	ret, _ := bc.getNetwork(nid)
	if ret == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found nid=%d", nid)
	}
	return ret, nil
}

func (bc *btpContext) GetNetworkTypeView(ntid int64) (btp.NetworkTypeView, error) {
	ret, _ := bc.getNetworkType(ntid)
	if ret == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found ntid=%d", ntid)
	}
	return ret, nil
}

func (bc *btpContext) GetNetwork(nid int64) (module.BTPNetwork, error) {
	ret, _ := bc.getNetwork(nid)
	if ret == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found nid=%d", nid)
	}
	return ret, nil
}

func (bc *btpContext) GetNetworkType(ntid int64) (module.BTPNetworkType, error) {
	ret, _ := bc.getNetworkType(ntid)
	if ret == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found ntid=%d", ntid)
	}
	return ret, nil
}

func (bc *btpContext) GetNetworkTypeIDByName(name string) int64 {
	if ntm.ForUID(name) == nil {
		return -1
	}
	ret, _ := bc.getNetworkTypeIdByName(name)
	return ret
}

func (bc *btpContext) GetPublicKey(from module.Address, name string, exactMatch bool) (pubKey []byte, fromDSA bool) {
	dbase := scoredb.NewDictDB(bc.Store(), PubKeyByNameKey, 2)
	if value := dbase.Get(from, name); value == nil {
		if !exactMatch {
			if mod := ntm.ForUID(name); mod != nil {
				if value = dbase.Get(from, mod.DSA()); value != nil {
					return value.Bytes(), true
				}
			}
		}
		return nil, false
	} else {
		return value.Bytes(), false
	}
}

func (bc *btpContext) getNetwork(nid int64) (*network, *containerdb.DictDB) {
	dbase := scoredb.NewDictDB(bc.store, NetworkByIDKey, 1)
	if value := dbase.Get(nid); value == nil {
		return nil, dbase
	} else {
		return NewNetworkFromBytes(value.Bytes()), dbase
	}
}

func (bc *btpContext) getNetworkType(ntid int64) (*networkType, *containerdb.DictDB) {
	dbase := scoredb.NewDictDB(bc.store, NetworkTypeByIDKey, 1)
	if value := dbase.Get(ntid); value == nil {
		return nil, dbase
	} else {
		return NewNetworkTypeFromBytes(value.Bytes()), dbase
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

func NewBTPContext(wc WorldContext, store containerdb.BytesStoreState) BTPContext {
	return &btpContext{
		wc:    wc,
		store: store,
	}
}

type btpData struct {
	dbase               db.Database
	validators          map[string]bool     // key: address
	proofContextChanged map[int64]bool      // key: network type ID
	pubKeyChanged       map[string][]string // key: address. value: slice of network type UID
	networkModified     map[int64]bool      // key: network ID
	digest              module.BTPDigest
	digestHash          []byte
}

func (bd *btpData) clone() *btpData {
	n := new(btpData)

	n.dbase = bd.dbase
	n.digest = bd.digest
	if bd.digestHash != nil {
		copy(n.digestHash, bd.digestHash)
	}

	n.validators = make(map[string]bool)
	for k, v := range bd.validators {
		n.validators[k] = v
	}

	n.proofContextChanged = make(map[int64]bool)
	for k, v := range bd.proofContextChanged {
		n.proofContextChanged[k] = v
	}

	n.pubKeyChanged = make(map[string][]string)
	for k, v := range bd.pubKeyChanged {
		nv := make([]string, len(v))
		copy(nv, v)
		n.pubKeyChanged[k] = nv
	}

	n.networkModified = make(map[int64]bool)
	for k, v := range bd.networkModified {
		n.networkModified[k] = v
	}

	return n
}

type btpSnapshot struct {
	*btpData
}

func (bss *btpSnapshot) Bytes() []byte {
	if bss.digestHash == nil && bss.digest != nil {
		bss.digestHash = bss.digest.Hash()
	}
	return bss.digestHash
}

func (bss *btpSnapshot) Flush() error {
	if bss.digest != nil {
		return bss.digest.Flush(bss.dbase)
	}
	return nil
}

func (bss *btpSnapshot) NewState() BTPState {
	state := new(BTPStateImpl)
	state.btpData = bss.clone()
	return state
}

func NewBTPSnapshot(dbase db.Database, hash []byte) BTPSnapshot {
	ss := new(btpSnapshot)
	ss.btpData = new(btpData)
	ss.dbase = dbase
	ss.digestHash = hash
	return ss
}

type BTPStateImpl struct {
	*btpData
}

func (bs *BTPStateImpl) GetSnapshot() BTPSnapshot {
	bss := new(btpSnapshot)
	bss.btpData = bs.clone()
	return bss
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

func (bs *BTPStateImpl) SetValidators(vs ValidatorState) {
	vMap := make(map[string]bool)
	for i := 0; i < vs.Len(); i++ {
		v, _ := vs.Get(i)
		vMap[string(v.Address().Bytes())] = true
	}
	bs.validators = vMap
}

func (bs *BTPStateImpl) setProofContextChanged(ntid int64) {
	if bs.proofContextChanged == nil {
		bs.proofContextChanged = make(map[int64]bool)
	}
	bs.proofContextChanged[ntid] = true
}

func (bs *BTPStateImpl) setPubKeyChanged(address module.Address, name string) {
	if bs.pubKeyChanged == nil {
		bs.pubKeyChanged = make(map[string][]string)
	}
	key := string(address.Bytes())
	if bs.pubKeyChanged[key] == nil {
		bs.pubKeyChanged[key] = make([]string, 0)
	}
	bs.pubKeyChanged[key] = append(bs.pubKeyChanged[key], name)
}

func (bs *BTPStateImpl) setNetworkModified(nid int64) {
	if bs.networkModified == nil {
		bs.networkModified = make(map[int64]bool)
	}
	bs.networkModified[nid] = true
}

func (bs *BTPStateImpl) getPubKeysOfValidators(bc BTPContext, mod module.NetworkTypeModule) ([][]byte, bool) {
	var err error
	keys := make([][]byte, 0)
	validators := bc.GetValidatorState()
	for i := 0; i < validators.Len(); i++ {
		v, _ := validators.Get(i)
		if key, fromDSA := bc.GetPublicKey(v.Address(), mod.UID(), false); key != nil {
			if fromDSA {
				key, err = mod.NetworkTypeKeyFromDSAKey(key)
				if err != nil {
					continue
				}
			}
			keys = append(keys, key)
		}
	}
	return keys, len(keys) == validators.Len()
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
	var nt *networkType
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
		if err = scoredb.NewArrayDB(bc.Store(), ActiveNetworkTypeIDsKey).Put(ntid); err != nil {
			return
		}

		keys, allHasPubKey := bs.getPubKeysOfValidators(bc, mod)
		if allHasPubKey != true {
			err = scoreresult.InvalidParameterError.Errorf("All validators must have public key for %s", mod.UID())
			return
		}
		nt = NewNetworkType(networkTypeName, mod.NewProofContext(keys))
	} else {
		if nt, _ = bci.getNetworkType(ntid); nt == nil {
			err = scoreresult.InvalidParameterError.Errorf("There is network type for %d", ntid)
			return
		}
	}

	store := bc.Store()
	nid, varDB = bci.getNewNetworkID()
	if err = varDB.Set(nid); err != nil {
		return
	}

	nw := NewNetwork(ntid, name, owner, bc.BlockHeight(), true)
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	if err = nwDB.Set(nid, nw.Bytes()); err != nil {
		return
	}

	nt.AddOpenNetworkID(nid)
	ntDB := scoredb.NewDictDB(store, NetworkTypeByIDKey, 1)
	if err = ntDB.Set(ntid, nt.Bytes()); err != nil {
		return
	}

	bs.setNetworkModified(nid)
	return
}

func (bs *BTPStateImpl) CloseNetwork(bc BTPContext, nid int64) (int64, error) {
	store := bc.Store()
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	nwValue := nwDB.Get(nid)
	if nwValue == nil {
		return 0, scoreresult.InvalidParameterError.Errorf("There is no network for %d", nid)
	}
	nw := NewNetworkFromBytes(nwValue.Bytes())
	nw.SetOpen(false)
	if err := nwDB.Set(nid, nw.Bytes()); err != nil {
		return 0, err
	}

	ntDB := scoredb.NewDictDB(store, NetworkTypeByIDKey, 1)
	if ntValue := ntDB.Get(nw.NetworkTypeID()); ntValue == nil {
		return 0, scoreresult.InvalidParameterError.Errorf("There is no network type for %d", nw.NetworkTypeID())
	} else {
		nt := NewNetworkTypeFromBytes(ntValue.Bytes())
		if err := nt.RemoveOpenNetworkID(nid); err != nil {
			return 0, scoreresult.InvalidParameterError.Wrapf(err, "There is no open network %d in %d", nid, nw.NetworkTypeID())
		}
		if err := ntDB.Set(nw.NetworkTypeID(), nt.Bytes()); err != nil {
			return 0, err
		}
	}

	return nw.NetworkTypeID(), nil
}

func (bs *BTPStateImpl) HandleMessage(bc BTPContext, from module.Address, nid int64) error {
	store := bc.Store()
	nwDB := scoredb.NewDictDB(store, NetworkByIDKey, 1)
	nwValue := nwDB.Get(nid)
	if nwValue == nil {
		return scoreresult.InvalidParameterError.Errorf("There is no network for %d", nid)
	}
	nw := NewNetworkFromBytes(nwValue.Bytes())
	if !from.Equal(nw.Owner()) {
		return scoreresult.AccessDeniedError.Errorf("Only owner can send BTP message")
	}
	nw.IncreaseNextMessageSN()
	if err := nwDB.Set(nid, nw.Bytes()); err != nil {
		return err
	}
	bs.setNetworkModified(nid)

	return nil
}

func (bs *BTPStateImpl) IsNetworkTypeUID(name string) bool {
	return ntm.ForUID(name) != nil
}

func (bs *BTPStateImpl) IsDSAName(name string) bool {
	for _, mod := range ntm.Modules() {
		if mod.DSA() == name {
			return true
		}
	}
	return false
}

func (bs *BTPStateImpl) SetPublicKey(bc BTPContext, from module.Address, name string, pubKey []byte) error {
	var mod module.NetworkTypeModule
	uids := make([]string, 0)
	dsa := true
	if mod = ntm.ForUID(name); mod == nil {
		for _, mod = range ntm.Modules() {
			if mod.DSA() == name {
				uids = append(uids, mod.UID())
			}
		}
	} else {
		dsa = false
		uids = append(uids, name)
	}
	if len(uids) == 0 {
		return scoreresult.InvalidParameterError.Errorf("Invalid name %s", name)
	}
	dbase := scoredb.NewDictDB(bc.Store(), PubKeyByNameKey, 2)
	old := dbase.Get(from, name)
	if old != nil && bytes.Compare(old.Bytes(), pubKey) == 0 {
		return nil
	}

	// find public key changed network type
	for _, uid := range uids {
		if 0 != bc.GetNetworkTypeIDByName(uid) {
			old = dbase.Get(from, uid)
			if old == nil || (!dsa && bytes.Compare(pubKey, old.Bytes()) != 0) {
				bs.setPubKeyChanged(from, uid)
			}
		}
	}

	if err := dbase.Set(from, name, pubKey); err != nil {
		return err
	}
	return nil
}

func (bs *BTPStateImpl) CheckPublicKey(bc BTPContext, from module.Address) error {
	openedNetworkTypes, err := bc.GetNetworkTypeIDs()
	if err != nil {
		return err
	}
	for _, ntid := range openedNetworkTypes {
		ntView, err := bc.GetNetworkTypeView(ntid)
		if err != nil {
			return err
		}
		if key, _ := bc.GetPublicKey(from, ntView.UID(), false); key == nil {
			return errors.NotFoundError.Errorf("not found pubKey for %s", from)
		}
	}
	return nil
}

func (bs *BTPStateImpl) update(bc BTPContext) error {
	for ntid := range bs.proofContextChanged {
		if err := bs.updateNetworkType(bc, ntid); err != nil {
			return err
		}
	}
	for nid := range bs.networkModified {
		if err := bs.updateNetwork(bc, nid); err != nil {
			return err
		}
	}
	return nil
}

func (bs *BTPStateImpl) updateNetworkType(bc BTPContext, ntid int64) error {
	bci := bc.(*btpContext)
	if nt, ntDB := bci.getNetworkType(ntid); nt == nil {
		return errors.NotFoundError.Errorf("not found ntid=%d", ntid)
	} else {
		mod := ntm.ForUID(nt.UID())
		keys, _ := bs.getPubKeysOfValidators(bc, mod)
		proof := mod.NewProofContext(keys)

		nt.SetNextProofContext(proof.Bytes())
		nt.SetNextProofContextHash(proof.Hash())
		if err := ntDB.Set(ntid, nt.Bytes()); err != nil {
			return err
		}
		return nil
	}
}

func (bs *BTPStateImpl) updateNetwork(bc BTPContext, nid int64) error {
	bci := bc.(*btpContext)
	if nw, nwDB := bci.getNetwork(nid); nw == nil {
		return errors.NotFoundError.Errorf("not found nid=%d", nid)
	} else {
		pcChanged := false
		if _, ok := bs.proofContextChanged[nw.NetworkTypeID()]; ok {
			pcChanged = true
		} else {
			pcChanged = len(nw.LastNetworkSectionHash()) == 0
		}
		nw.SetNextProofContextChanged(pcChanged)
		return nwDB.Set(nid, nw.Bytes())
	}
}

func (bs *BTPStateImpl) applyBTPSection(bc BTPContext, btpSection module.BTPSection) error {
	for _, nts := range btpSection.NetworkTypeSections() {
		for nid := range bs.networkModified {
			ns, err := nts.NetworkSectionFor(nid)
			if err != nil {
				continue
			}
			if err = bs.applyNetwork(bc, ns); err != nil {
				return err
			}
		}
	}

	bs.digest = btpSection.Digest()
	bs.digestHash = bs.digest.Hash()
	return nil
}

func (bs *BTPStateImpl) applyNetwork(bc BTPContext, ns module.NetworkSection) error {
	bci := bc.(*btpContext)
	nid := ns.NetworkID()
	if nw, nwDB := bci.getNetwork(nid); nw == nil {
		return errors.NotFoundError.Errorf("not found nid=%d", nid)
	} else {
		nw.SetPrevNetworkSectionHash(nw.LastNetworkSectionHash())
		nw.SetLastNetworkSectionHash(ns.Hash())
		return nwDB.Set(nid, nw.Bytes())
	}
}

func (bs *BTPStateImpl) setValidatorChanged(bc BTPContext, names []string) error {
	bci := bc.(*btpContext)
	for _, name := range names {
		ntid, _ := bci.getNetworkTypeIdByName(name)
		nt, err := bc.GetNetworkTypeView(ntid)
		bs.setProofContextChanged(ntid)
		if err != nil {
			return err
		}
		for _, nid := range nt.OpenNetworkIDs() {
			bs.setNetworkModified(nid)
		}
	}
	return nil
}

func (bs *BTPStateImpl) compareValidators(v2 ValidatorState) bool {
	if len(bs.validators) != v2.Len() {
		return false
	}

	for i := 0; i < v2.Len(); i++ {
		v, _ := v2.Get(i)
		if _, ok := bs.validators[string(v.Address().Bytes())]; !ok {
			return false
		}
	}
	return true
}

func (bs *BTPStateImpl) handleValidatorChange(bc BTPContext) error {
	names := make([]string, 0)
	if bs.compareValidators(bc.GetValidatorState()) == false {
		// validator list changed
		ntids, err := bc.GetNetworkTypeIDs()
		if err != nil {
			return err
		}
		for _, ntid := range ntids {
			ntView, err := bc.GetNetworkTypeView(ntid)
			if err != nil {
				return err
			}
			names = append(names, ntView.UID())
		}
	} else {
		for key := range bs.validators {
			if name, ok := bs.pubKeyChanged[key]; ok {
				// public key changed
				names = append(names, name...)
			}
		}
	}
	if err := bs.setValidatorChanged(bc, names); err != nil {
		return err
	}
	return nil
}

func (bs *BTPStateImpl) BuildAndApplySection(bc BTPContext, btpMsgs *list.List) (module.BTPSection, error) {
	sb := btp.NewSectionBuilder(bc)

	// check validator change
	if err := bs.handleValidatorChange(bc); err != nil {
		return nil, err
	}

	for nid := range bs.networkModified {
		sb.EnsureSection(nid)
	}

	for i := btpMsgs.Front(); i != nil; i = i.Next() {
		e := i.Value.(*bTPMsg)
		sb.SendMessage(e.nid, e.message)
	}

	if err := bs.update(bc); err != nil {
		return nil, err
	}

	if section, err := sb.Build(); err != nil {
		return nil, err
	} else {
		if err = bs.applyBTPSection(bc, section); err != nil {
			return nil, err
		}
		return section, nil
	}
}

func NewBTPState(dbase db.Database, hash []byte) BTPState {
	state := new(BTPStateImpl)
	state.btpData = new(btpData)
	state.dbase = dbase
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

type networkType struct {
	uid                  string
	nextProofContextHash []byte
	nextProofContext     []byte
	openNetworkIDs       []int64
}

func (nt *networkType) UID() string {
	return nt.uid
}

func (nt *networkType) NextProofContextHash() []byte {
	return nt.nextProofContextHash
}

func (nt *networkType) NextProofContext() []byte {
	return nt.nextProofContext
}

func (nt *networkType) OpenNetworkIDs() []int64 {
	return nt.openNetworkIDs
}
func (nt *networkType) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["networkTypeName"] = nt.UID()
	if len(nt.NextProofContext()) == 0 {
		jso["nextProofContext"] = nil
	} else {
		jso["nextProofContext"] = base64.StdEncoding.EncodeToString(nt.nextProofContext)
	}
	nids := nt.OpenNetworkIDs()
	onids := make([]interface{}, len(nids))
	for i, nid := range nids {
		onids[i] = intconv.FormatInt(nid)
	}
	jso["openNetworkIDs"] = onids
	return jso
}

func (nt *networkType) SetNextProofContextHash(hash []byte) {
	nt.nextProofContextHash = hash
}

func (nt *networkType) SetNextProofContext(bs []byte) {
	nt.nextProofContext = bs
}

func (nt *networkType) AddOpenNetworkID(nid int64) {
	nt.openNetworkIDs = append(nt.openNetworkIDs, nid)
}

func (nt *networkType) RemoveOpenNetworkID(nid int64) error {
	for i, v := range nt.OpenNetworkIDs() {
		if v == nid {
			copy(nt.openNetworkIDs[i:], nt.openNetworkIDs[i+1:])
			nt.openNetworkIDs[len(nt.openNetworkIDs)-1] = 0
			nt.openNetworkIDs = nt.openNetworkIDs[:len(nt.openNetworkIDs)-1]
			return nil
		}
	}
	return errors.Errorf("There is no open network id %d", nid)
}

func (nt *networkType) Bytes() []byte {
	return codec.MustMarshalToBytes(nt)
}

func (nt *networkType) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&nt.uid,
		&nt.nextProofContextHash,
		&nt.nextProofContext,
		&nt.openNetworkIDs,
	)
}

func (nt *networkType) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		nt.uid,
		nt.nextProofContextHash,
		nt.nextProofContext,
		nt.openNetworkIDs,
	)
}

func NewNetworkType(uid string, proofContext module.BTPProofContext) *networkType {
	nt := new(networkType)
	nt.uid = uid
	if proofContext != nil {
		nt.nextProofContext = proofContext.Bytes()
		nt.nextProofContextHash = proofContext.Hash()
	}
	return nt
}

func NewNetworkTypeFromBytes(b []byte) *networkType {
	nt := new(networkType)
	codec.MustUnmarshalFromBytes(b, nt)
	return nt
}

type network struct {
	startHeight             int64
	name                    string
	owner                   *common.Address
	networkTypeID           int64
	open                    bool
	nextMessageSN           int64
	nextProofContextChanged bool
	prevNetworkSectionHash  []byte
	lastNetworkSectionHash  []byte
}

func (nw *network) StartHeight() int64 {
	return nw.startHeight
}

func (nw *network) Name() string {
	return nw.name
}

func (nw *network) Owner() module.Address {
	return nw.owner
}

func (nw *network) NetworkTypeID() int64 {
	return nw.networkTypeID
}

func (nw *network) Open() bool {
	return nw.open
}

func (nw *network) NextMessageSN() int64 {
	return nw.nextMessageSN
}

func (nw *network) NextProofContextChanged() bool {
	return nw.nextProofContextChanged
}

func (nw *network) PrevNetworkSectionHash() []byte {
	return nw.prevNetworkSectionHash
}

func (nw *network) LastNetworkSectionHash() []byte {
	return nw.lastNetworkSectionHash
}

func formatBool(yn bool) string {
	if yn {
		return "0x1"
	} else {
		return "0x0"
	}
}

func (nw *network) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["startHeight"] = intconv.FormatInt(nw.startHeight)
	jso["networkTypeID"] = intconv.FormatInt(nw.networkTypeID)
	jso["networkName"] = nw.name
	jso["open"] = formatBool(nw.open)
	jso["nextMessageSN"] = intconv.FormatInt(nw.nextMessageSN)
	jso["nextProofContextChanged"] = formatBool(nw.nextProofContextChanged)
	if len(nw.prevNetworkSectionHash) == 0 {
		jso["prevNSHash"] = nil
	} else {
		jso["prevNSHash"] = "0x" + hex.EncodeToString(nw.prevNetworkSectionHash)
	}
	if len(nw.lastNetworkSectionHash) == 0 {
		jso["lastNSHash"] = nil
	} else {
		jso["lastNSHash"] = "0x" + hex.EncodeToString(nw.lastNetworkSectionHash)
	}
	return jso
}

func (nw *network) SetOpen(yn bool) {
	nw.open = yn
}

func (nw *network) IncreaseNextMessageSN() {
	nw.nextMessageSN++
}

func (nw *network) SetNextProofContextChanged(yn bool) {
	nw.nextProofContextChanged = yn
}

func (nw *network) SetPrevNetworkSectionHash(hash []byte) {
	nw.prevNetworkSectionHash = hash
}

func (nw *network) SetLastNetworkSectionHash(hash []byte) {
	nw.lastNetworkSectionHash = hash
}

func (nw *network) Bytes() []byte {
	return codec.MustMarshalToBytes(nw)
}

func (nw *network) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&nw.startHeight,
		&nw.name,
		&nw.owner,
		&nw.networkTypeID,
		&nw.open,
		&nw.nextMessageSN,
		&nw.nextProofContextChanged,
		&nw.prevNetworkSectionHash,
		&nw.lastNetworkSectionHash,
	)
}

func (nw *network) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		nw.startHeight,
		nw.name,
		nw.owner,
		nw.networkTypeID,
		nw.open,
		nw.nextMessageSN,
		nw.nextProofContextChanged,
		nw.prevNetworkSectionHash,
		nw.lastNetworkSectionHash,
	)
}

func NewNetwork(ntid int64, name string, owner module.Address, startHeight int64, nextProofContextChanged bool) *network {
	return &network{
		networkTypeID:           ntid,
		name:                    name,
		owner:                   common.AddressToPtr(owner),
		open:                    true,
		startHeight:             startHeight,
		nextProofContextChanged: nextProofContextChanged,
	}
}

func NewNetworkFromBytes(b []byte) *network {
	nw := new(network)
	codec.MustUnmarshalFromBytes(b, nw)
	return nw
}
