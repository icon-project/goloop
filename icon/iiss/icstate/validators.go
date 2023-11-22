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
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
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
	nodeList   []*common.Address
	nextPssIdx int
	// The last block height when validatorList was updated
	// But it is exceptionally 0 if validatorList is updated at the end of the current term
	lastHeight int64

	nodeMap map[string]int
}

func (vd *validatorsData) init(prepSnapshots PRepSnapshots, ownerToNodeMapper OwnerToNodeMappable, size int) {
	size = icutils.Min(len(prepSnapshots), size)
	vd.nodeList = make([]*common.Address, size)
	vd.nodeMap = make(map[string]int)

	for i := 0; i < size; i++ {
		owner := prepSnapshots[i].Owner()
		node := ownerToNodeMapper.GetNodeByOwner(owner)
		if node == nil {
			node = owner
		}

		vd.nodeList[i] = common.AddressToPtr(node)
		vd.nodeMap[icutils.ToKey(node)] = i
	}
	vd.nextPssIdx = size
}

func (vd *validatorsData) equal(other *validatorsData) bool {
	if vd == other {
		return true
	}
	if vd.Len() != other.Len() {
		return false
	}
	if vd.nextPssIdx != other.nextPssIdx {
		return false
	}
	if vd.lastHeight != other.lastHeight {
		return false
	}
	for i, node := range vd.nodeList {
		if !node.Equal(other.nodeList[i]) {
			return false
		}
	}
	return true
}

func (vd *validatorsData) clone() validatorsData {
	size := len(vd.nodeList)
	nodeMap := make(map[string]int)
	nodeList := make([]*common.Address, size)

	for i, node := range vd.nodeList {
		nodeList[i] = node
		nodeMap[icutils.ToKey(node)] = i
	}
	return validatorsData{
		nodeList:   nodeList,
		nodeMap:    nodeMap,
		nextPssIdx: vd.nextPssIdx,
		lastHeight: vd.lastHeight,
	}
}

func (vd *validatorsData) set(other *validatorsData) {
	size := len(other.nodeList)
	nodeMap := make(map[string]int)
	nodeList := make([]*common.Address, size)

	for i, node := range other.nodeList {
		nodeList[i] = node
		nodeMap[icutils.ToKey(node)] = i
	}
	vd.nodeList = nodeList
	vd.nodeMap = nodeMap
	vd.nextPssIdx = other.nextPssIdx
	vd.lastHeight = other.lastHeight
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

func (vd *validatorsData) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f,
				"{nodeList:%+v nextPssIdx:%+v lastHeight:%+v}",
				vd.nodeList, vd.nextPssIdx, vd.lastHeight,
			)
		} else {
			_, _ = fmt.Fprintf(f, "{%v %v %v}", vd.nodeList, vd.nextPssIdx, vd.lastHeight)
		}
	case 's':
		_, _ = fmt.Fprint(f, vd.String())
	}
}

func (vd *validatorsData) String() string {
	return fmt.Sprintf("{%s %d %d}", vd.nodeList, vd.nextPssIdx, vd.lastHeight)
}

func newValidatorsData(nodes []module.Address) validatorsData {
	size := len(nodes)
	nodeList := make([]*common.Address, size)
	nodeMap := make(map[string]int)

	for i, node := range nodes {
		nodeList[i] = common.AddressToPtr(node)
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
	if err := decoder.DecodeAll(&vss.nodeList, &vss.nextPssIdx, &vss.lastHeight); err != nil {
		return err
	}

	vss.nodeMap = make(map[string]int)
	for i, node := range vss.nodeList {
		vss.nodeMap[icutils.ToKey(node)] = i
	}
	return nil
}

func (vss *ValidatorsSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(vss.nodeList, vss.nextPssIdx, vss.lastHeight)
}

func (vss *ValidatorsSnapshot) Equal(object icobject.Impl) bool {
	other, ok := object.(*ValidatorsSnapshot)
	if !ok {
		return false
	}
	if vss == other {
		return true
	}
	return vss.equal(&other.validatorsData)
}

// IsUpdated returns true if validatorList is updated at this block
func (vss *ValidatorsSnapshot) IsUpdated(blockHeight int64) bool {
	if blockHeight < vss.lastHeight {
		panic(errors.Errorf("Invalid blockHeight: bh=%d < lh=%d", blockHeight, vss.lastHeight))
	}
	return blockHeight == vss.lastHeight
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

func (vs *ValidatorsState) Set(blockHeight int64, i, nextPssIdx int, node module.Address) {
	if node == nil {
		vs.remove(i)
	} else {
		oldNode := vs.nodeList[i]
		if oldNode.Equal(node) {
			// No need to update
			return
		}
		vs.set(i, nextPssIdx, node)
	}

	// Record the blockHeight when ValidatorsState is updated
	vs.lastHeight = blockHeight
	vs.setDirty()
}

func (vs *ValidatorsState) set(i, nextPssIdx int, node module.Address) {
	old := vs.nodeList[i]
	if old.Equal(node) {
		return
	}

	delete(vs.nodeMap, icutils.ToKey(old))

	vs.nodeList[i] = common.AddressToPtr(node)
	vs.nodeMap[icutils.ToKey(node)] = i
	if nextPssIdx >= 0 {
		vs.nextPssIdx = nextPssIdx
	}
}

func (vs *ValidatorsState) remove(i int) {
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
	prepSnapshots PRepSnapshots, ownerToNodeMapper OwnerToNodeMappable, size int) *ValidatorsSnapshot {
	vss := &ValidatorsSnapshot{}
	vss.validatorsData.init(prepSnapshots, ownerToNodeMapper, size)
	return vss
}

// changeValidatorNodeAddress is called when a main prep wants to change its node address
func (s *State) changeValidatorNodeAddress(
	blockHeight int64, owner module.Address, oldNode module.Address, newNode module.Address) error {
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
	vs.Set(blockHeight, i, -1, newNode)
	return s.SetValidatorsSnapshot(vs.GetSnapshot())
}

func (s *State) replaceMainPRepByOwner(sc icmodule.StateContext, owner module.Address) error {
	node := s.GetNodeByOwner(owner)
	blockHeight := sc.BlockHeight()
	newMainPRepOwner, err := s.replaceMainPRepByNode(node, blockHeight)
	if err != nil {
		return err
	}
	if newMainPRepOwner != nil {
		err = s.OnMainPRepReplaced(sc, owner, newMainPRepOwner)
	}
	return err
}

// Do not modify PRepStatusState fields here
func (s *State) replaceMainPRepByNode(node module.Address, blockHeight int64) (module.Address, error) {
	vss := s.GetValidatorsSnapshot()
	if vss == nil {
		return nil, errors.InvalidStateError.Errorf("ValidatorsSnapshot not found: bh=%d", blockHeight)
	}

	i := vss.IndexOf(node)
	if i < 0 {
		return nil, errors.Errorf("Invalid validator: node=%s", node)
	}

	term := s.GetTermSnapshot()
	index := vss.NextPRepSnapshotIndex()
	newOwner, nextPssIdx := s.chooseNewMainPRep(term.prepSnapshots, index)
	if nextPssIdx < 0 {
		s.logData(blockHeight, node, i, term, vss, index)
		return nil, errors.Errorf("Failed to choose a new validator: oldNode=%s", node)
	}

	vs := NewValidatorsStateWithSnapshot(vss)
	vs.Set(blockHeight, i, nextPssIdx, s.GetNodeByOwner(newOwner))
	return newOwner, s.SetValidatorsSnapshot(vs.GetSnapshot())
}

func (s *State) logData(
	blockHeight int64, node module.Address, vssIdx int, term *TermSnapshot, vss *ValidatorsSnapshot, startIdx int) {
	s.logger.Errorf("Extra main prep error start =================================================")
	s.logger.Errorf("bh=%d node=%s vssIdx=%d startIdx=%d", blockHeight, node, vssIdx, startIdx)
	s.logger.Errorf("term=%s", term)
	s.logger.Errorf("vss=%s", vss)
	s.logger.Errorf("pss=%s", term.prepSnapshots)
	s.logger.Errorf("Extra main prep error end ===================================================")
}

// chooseNewMainPRep returns the owner address of a new validator from PRepSnapshots
// DO NOT change any fields of PRepStatus here
func (s *State) chooseNewMainPRep(prepSnapshots PRepSnapshots, startIdx int) (module.Address, int) {
	var ps *PRepStatusState
	var pss *PRepSnapshot

	size := len(prepSnapshots)
	for i := startIdx; i < size; i++ {
		pss = prepSnapshots[i]
		owner := pss.Owner()

		ps = s.GetPRepStatusByOwner(owner, false)
		if ps == nil {
			continue
		}

		switch ps.Grade() {
		case GradeMain:
			return nil, -1
		case GradeSub:
			return owner, i + 1
		}
	}
	// No SubPRep remains to replace old one
	return nil, size
}
