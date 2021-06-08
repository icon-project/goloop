/*
 * Copyright 2021 ICON Foundation
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

package icstate

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

var emptyValidatorsData = validatorsData{
	nodeMap: make(map[string]int),
}

var emptyValidatorsSnapshot = &ValidatorsSnapshot{
	validatorsData: emptyValidatorsData,
}

type validatorsData struct {
	nodeList   []module.Address
	nextPssIdx int

	nodeMap    map[string]int
	serialized []byte
	hash       []byte
}

func (vd *validatorsData) init(prepSnapshots Arrayable, ownerToNodeMapper OwnerToNodeMappable, size int) {
	size = icutils.Min(prepSnapshots.Len(), size)
	vd.nodeList = make([]module.Address, size)
	vd.nodeMap = make(map[string]int)

	for i := 0; i < size; i++ {
		owner := prepSnapshots.Get(i).(*PRepSnapshot).Owner()
		node := ownerToNodeMapper.GetNodeByOwner(owner)
		if node == nil {
			node = owner
		}

		vd.nodeList[i] = node
		vd.nodeMap[icutils.ToKey(node)] = i
	}
	vd.nextPssIdx = size
}

func (vd *validatorsData) Hash() []byte {
	if vd.hash == nil && len(vd.nodeList) > 0 {
		s := vd.serialize()
		vd.hash = crypto.SHA3Sum256(s)
	}
	return vd.hash
}

func (vd *validatorsData) serialize() []byte {
	if vd.serialized == nil && len(vd.nodeList) > 0 {
		vd.serialized, _ = codec.BC.MarshalToBytes(vd.nodeList)
	}
	return vd.serialized
}

func (vd *validatorsData) equal(other *validatorsData) bool {
	if vd.Len() != other.Len() {
		return false
	}
	if vd.nextPssIdx != other.nextPssIdx {
		return false
	}
	for i, node := range vd.nodeList {
		if node != other.nodeList[i] {
			return false
		}
	}
	return true
}

func (vd *validatorsData) clone() validatorsData {
	size := len(vd.nodeList)
	nodeMap := make(map[string]int)
	nodeList := make([]module.Address, size)

	for i, node := range vd.nodeList {
		nodeList[i] = node
		nodeMap[icutils.ToKey(node)] = i
	}
	return validatorsData{
		nodeList:   nodeList,
		nodeMap:    nodeMap,
		nextPssIdx: vd.nextPssIdx,
	}
}

func (vd *validatorsData) IndexOf(node module.Address) int {
	key := icutils.ToKey(node)
	idx, ok := vd.nodeMap[key]
	if !ok {
		return -1
	}
	return idx
}

func (vd *validatorsData) Len() int {
	return len(vd.nodeList)
}

func (vd *validatorsData) Get(i int) module.Address {
	if i < 0 && i >= vd.Len() {
		return nil
	}
	return vd.nodeList[i]
}

func (vd *validatorsData) NextPRepSnapshotIndex() int {
	return vd.nextPssIdx
}

func (vd *validatorsData) NewValidatorSet() []module.Validator {
	size := vd.Len()
	vSet := make([]module.Validator, size)
	for i, node := range vd.nodeList {
		vSet[i], _ = state.ValidatorFromAddress(node)
	}
	return vSet
}

func newValidatorsData(nodes []module.Address) validatorsData {
	size := len(nodes)
	nodeList := make([]module.Address, size)
	nodeMap := make(map[string]int)

	for i, node := range nodes {
		nodeList[i] = node
		nodeMap[icutils.ToKey(node)] = i
	}

	return validatorsData{
		nodeList:   nodeList,
		nodeMap:    nodeMap,
		nextPssIdx: size,
	}
}

// ===========================================================================

type ValidatorsSnapshot struct {
	icobject.NoDatabase
	validatorsData
}

func (vss *ValidatorsSnapshot) Version() int {
	return 0
}

func (vss *ValidatorsSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	if err := decoder.DecodeListOf(&vss.nodeList, &vss.nextPssIdx); err != nil {
		return err
	}

	vss.nodeMap = make(map[string]int)
	for i, node := range vss.nodeList {
		vss.nodeMap[icutils.ToKey(node)] = i
	}
	return nil
}

func (vss *ValidatorsSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(vss.nodeList, vss.nextPssIdx)
}

func (vss *ValidatorsSnapshot) Equal(object icobject.Impl) bool {
	other, ok := object.(*ValidatorsSnapshot)
	if !ok {
		return false
	}
	return vss.equal(&other.validatorsData)
}

// =======================================================

type ValidatorsState struct {
	snapshot *ValidatorsSnapshot
	validatorsData
}

func (vs *ValidatorsState) setDirty() {
	if vs.snapshot != nil {
		vs.snapshot = nil
	}
}

func (vs *ValidatorsState) IsDirty() bool {
	return vs.snapshot == nil
}

func (vs *ValidatorsState) Set(i, nextPssIdx int, node module.Address) {
	old := vs.nodeList[i]
	if old.Equal(node) {
		return
	}

	delete(vs.nodeMap, icutils.ToKey(old))

	vs.nodeList[i] = node
	vs.nodeMap[icutils.ToKey(node)] = i
	if nextPssIdx >= 0 {
		vs.nextPssIdx = nextPssIdx
	}

	vs.setDirty()
}

func (vs *ValidatorsState) Remove(i int) {
	size := len(vs.nodeList)
	if i < 0 || i >= size {
		return
	}

	nodeList := vs.nodeList

	node := nodeList[i]
	delete(vs.nodeMap, icutils.ToKey(node))

	for j := i + 1; j < size; j++ {
		node = nodeList[j]
		nodeList[j-1] = node
		vs.nodeMap[icutils.ToKey(node)] = j - 1
	}

	vs.nodeList = nodeList[:size-1]
	vs.setDirty()
}

func (vs *ValidatorsState) GetSnapshot() *ValidatorsSnapshot {
	if vs.snapshot == nil {
		vs.snapshot = &ValidatorsSnapshot{
			validatorsData: vs.validatorsData.clone(),
		}
	}
	return vs.snapshot
}

func (vs *ValidatorsState) Reset(vss *ValidatorsSnapshot) {
	if vs.snapshot != nil && vs.snapshot.Equal(vss) {
		return
	}
	vs.snapshot = vss
	vs.validatorsData = vss.validatorsData.clone()
}

func newValidatorsWithTag(_ icobject.Tag) *ValidatorsSnapshot {
	return new(ValidatorsSnapshot)
}

func NewValidatorsStateWithSnapshot(vss *ValidatorsSnapshot) *ValidatorsState {
	vs := new(ValidatorsState)
	if vss == nil {
		vss = emptyValidatorsSnapshot
	}
	vs.Reset(vss)
	return vs
}

func NewValidatorsSnapshotWithPRepSnapshot(
	prepSnapshot Arrayable, ownerToNodeMapper OwnerToNodeMappable, size int) *ValidatorsSnapshot {
	vd := validatorsData{}
	vd.init(prepSnapshot, ownerToNodeMapper, size)

	return &ValidatorsSnapshot{
		validatorsData: vd,
	}
}

// changeValidatorNodeAddress is called when a main prep wants to change its node address
func (s *State) changeValidatorNodeAddress(
	owner module.Address, oldNode module.Address, newNode module.Address) error {
	if owner == nil || oldNode == nil || newNode == nil {
		return errors.Errorf(
			"Invalid argument: owner=%s oldNode=%s newNode=%s",
			owner, oldNode, newNode,
		)
	}

	// If old node is equal to new node, no need to change validators
	if oldNode.Equal(newNode) {
		return nil
	}

	ownerByOldNode := s.nodeOwnerCache.Get(oldNode)
	if !owner.Equal(ownerByOldNode) {
		return errors.Errorf("Owner mismatch: %s != %s", ownerByOldNode, owner)
	}

	vss := s.GetValidatorsSnapshot()
	if vss == nil {
		// No validators to change
		return nil
	}

	i := vss.IndexOf(oldNode)
	if i < 0 {
		return errors.Errorf("Invalid validator: node=%s", oldNode)
	}

	vs := NewValidatorsStateWithSnapshot(vss)
	vs.Set(i, -1, newNode)
	return s.SetValidatorsSnapshot(vs.GetSnapshot())
}

func (s *State) replaceValidatorByOwner(owner module.Address) error {
	node := s.GetNodeByOwner(owner)
	return s.replaceValidatorByNode(node)
}

func (s *State) replaceValidatorByNode(node module.Address) error {
	vss := s.GetValidatorsSnapshot()
	i := vss.IndexOf(node)
	if i < 0 {
		return errors.Errorf("Invalid validator: node=%s", node)
	}

	term := s.GetTerm()
	newOwner, nextPssIdx, _ := s.chooseNewValidator(term.prepSnapshots, vss.NextPRepSnapshotIndex())
	newNode := s.GetNodeByOwner(newOwner)

	vs := NewValidatorsStateWithSnapshot(vss)
	if newNode != nil {
		vs.Set(i, nextPssIdx, newNode)
	} else {
		vs.Remove(i)
	}
	return s.SetValidatorsSnapshot(vs.GetSnapshot())
}

// chooseNewValidator returns the owner address of a new validator from PRepSnapshots
// changing its grade from Sub to Main
func (s *State) chooseNewValidator(prepSnapshots Arrayable, startIdx int) (module.Address, int, error) {
	var ps *PRepStatus
	var pss *PRepSnapshot

	size := prepSnapshots.Len()
	for i := startIdx; i < size; i++ {
		pss = prepSnapshots.Get(i).(*PRepSnapshot)
		owner := pss.Owner()

		ps, _ = s.GetPRepStatusByOwner(owner, false)
		if ps == nil {
			continue
		}

		switch ps.Grade() {
		case Main:
			return nil, size, errors.Errorf("Critical problem in PRep grade management")
		case Sub:
			ps.SetGrade(Main)
			return owner, i + 1, nil
		}
	}
	// No SubPRep remains to replace old one
	return nil, size, nil
}
