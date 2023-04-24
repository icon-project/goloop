package state

import (
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
)

type ValidatorSnapshot module.ValidatorList

type ValidatorState interface {
	IndexOf(module.Address) int
	Len() int
	Get(i int) (module.Validator, bool)
	Set([]module.Validator) error
	Add(v module.Validator) error
	Remove(v module.Validator) bool
	Replace(ov, nv module.Validator) error
	SetAt(i int, v module.Validator) error
	GetSnapshot() ValidatorSnapshot
	Reset(ValidatorSnapshot)
}

type validatorList struct {
	lock       sync.Mutex
	bucket     db.Bucket
	validators []*validator
	addrMap    map[string]int
}

func (vl *validatorList) Len() int {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return len(vl.validators)
}

func (vl *validatorList) Get(i int) (module.Validator, bool) {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return vl.getInLock(i)
}

func (vl *validatorList) getInLock(i int) (module.Validator, bool) {
	if i < 0 || i >= len(vl.validators) {
		return nil, false
	}
	return vl.validators[i], true
}

func (vl *validatorList) IsSame(validators []*validator) bool {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	if len(validators) != len(vl.validators) {
		return false
	}
	for i, v := range vl.validators {
		if !validators[i].Equal(v) {
			return false
		}
	}
	return true
}

func (vl *validatorList) String() string {
	vl.lock.Lock()
	defer vl.lock.Unlock()
	return fmt.Sprintf("validatorList[%+v]", vl.validators)
}

func (vl *validatorList) IndexOf(addr module.Address) int {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return vl.indexOfInLock(addr, true)
}

func makeAddrMapFrom(vl []*validator, strict bool) (map[string]int, error) {
	m := make(map[string]int)
	for i, v := range vl {
		m[string(v.Address().Bytes())] = i
	}
	if strict && len(m) != len(vl) {
		return nil, errors.IllegalArgumentError.Errorf(
			"DuplicateValidator(duplicates=%d)", len(vl)-len(m))
	}
	return m, nil
}

func (vl *validatorList) indexOfInLock(addr module.Address, mapCreate bool) int {
	if vl.addrMap == nil {
		if !mapCreate {
			for i, v := range vl.validators {
				if v.Address().Equal(addr) {
					return i
				}
			}
			return -1
		}
		vl.addrMap, _ = makeAddrMapFrom(vl.validators, false)
	}
	if idx, ok := vl.addrMap[string(addr.Bytes())]; ok {
		return idx
	}
	return -1
}

func (vl *validatorList) clone() *validatorList {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return &validatorList {
		bucket: vl.bucket,
		validators: append([]*validator(nil), vl.validators...),
	}
}

type validatorSnapshot struct {
	*validatorList
	dirty      bool
	serialized []byte
	hash       []byte
}

func (vss *validatorSnapshot) hashInLock() []byte {
	if vss.hash == nil && len(vss.validators) > 0 {
		s := vss.serializeInLock()
		vss.hash = crypto.SHA3Sum256(s)
	}
	return vss.hash
}

func (vss *validatorSnapshot) Hash() []byte {
	vss.lock.Lock()
	defer vss.lock.Unlock()
	return vss.hashInLock()
}

func (vss *validatorSnapshot) serializeInLock() []byte {
	if vss.serialized == nil && len(vss.validators) > 0 {
		vss.serialized, _ = codec.BC.MarshalToBytes(vss.validators)
	}
	return vss.serialized
}

func (vss *validatorSnapshot) OnData(bs []byte, bd merkle.Builder) error {
	vss.serialized = bs
	if _, err := codec.BC.UnmarshalFromBytes(bs, &vss.validators); err != nil {
		return err
	}
	return nil
}

func (vss *validatorSnapshot) Bytes() []byte {
	vss.lock.Lock()
	defer vss.lock.Unlock()
	return vss.serializeInLock()
}

func (vss *validatorSnapshot) Flush() error {
	vss.lock.Lock()
	defer vss.lock.Unlock()

	if vss.dirty {
		data := vss.serializeInLock()
		key := vss.hashInLock()
		return vss.bucket.Set(key, data)
	}
	return nil
}

type validatorState struct {
	*validatorList
	wLock    sync.Mutex
	snapshot *validatorSnapshot
}

func (vs *validatorState) GetSnapshot() ValidatorSnapshot {
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	if vs.snapshot != nil {
		return vs.snapshot
	}

	vss := new(validatorSnapshot)
	vss.validatorList = vs.validatorList
	vss.dirty = true
	vs.snapshot = vss
	return vss
}

func (vs *validatorState) Reset(vss ValidatorSnapshot) {
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	snapshot, ok := vss.(*validatorSnapshot)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", vss)
	}
	if vs.snapshot == snapshot {
		return
	}
	if vs.bucket != snapshot.bucket {
		log.Panicf("It tries to Reset with invalid snapshot bucket=%+v", snapshot.bucket)
	}
	vs.validatorList = snapshot.validatorList
	vs.snapshot = snapshot
}

func ToSliceOfValidatorPtr(vl []module.Validator) ([]*validator, error) {
	vList := make([]*validator, len(vl))
	for i, v := range vl {
		if vo, err := validatorFromValidator(v); err != nil {
			return nil, err
		} else {
			vList[i] = vo
		}
	}
	return vList, nil
}

func (vs *validatorState) becomeChangeableInLock() {
	vs.updateValidatorsInLock(nil, nil)
}

func (vs *validatorState) updateValidatorsInLock(validators []*validator, addrMap map[string]int) {
	if vs.snapshot != nil {
		if validators != nil {
			vs.validatorList = &validatorList{
				bucket:     vs.bucket,
				validators: validators,
			}
		} else {
			vs.validatorList = vs.clone()
		}
		vs.snapshot = nil
	} else if validators != nil {
		vs.validators = validators
		vs.addrMap = addrMap
	}
}

func (vs *validatorState) Set(vl []module.Validator) error {
	validators, err := ToSliceOfValidatorPtr(vl)
	if err != nil {
		return err
	}
	if vs.IsSame(validators) {
		return nil
	}
	addrMap, err := makeAddrMapFrom(validators, true);
	if err != nil {
		return err
	}
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	vs.updateValidatorsInLock(validators, addrMap)
	return nil
}

func (vs *validatorState) Add(v module.Validator) error {
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	if idx := vs.IndexOf(v.Address()); idx >= 0 {
		return nil
	}
	vo, err := validatorFromValidator(v)
	if err != nil {
		return err
	}
	vs.becomeChangeableInLock()
	vs.validators = append(vs.validators, vo)
	if vs.addrMap != nil {
		vs.addrMap[string(v.Address().Bytes())] = len(vs.validators) - 1
	}
	return nil
}

func (vs *validatorState) Remove(v module.Validator) bool {
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	i := vs.IndexOf(v.Address())
	if i < 0 {
		return false
	}
	vs.becomeChangeableInLock()
	vs.validators = append(vs.validators[:i], vs.validators[i+1:]...)
	vs.addrMap = nil
	return true
}

func (vs *validatorState) Replace(ov, nv module.Validator) error {
	oAddr := ov.Address()
	nAddr := nv.Address()
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	i := vs.IndexOf(oAddr)
	if i < 0 {
		return errors.NotFoundError.New("ValidatorNotFound")
	}
	if oAddr.Equal(nAddr) {
		return nil
	}
	if idx := vs.IndexOf(nAddr); idx >= 0 {
		return  errors.IllegalArgumentError.New("ValidatorInUse")
	}
	return vs.setAtInLock(i, ov, nv)
}

func (vs *validatorState) SetAt(i int, v module.Validator) error {
	vs.wLock.Lock()
	defer vs.wLock.Unlock()

	ov, ok := vs.Get(i)
	if !ok {
		return errors.IllegalArgumentError.New("IndexOutOfRange")
	}

	oAddr := ov.Address()
	nAddr := v.Address()
	if oAddr.Equal(nAddr) {
		// No need to change
		return nil
	}

	if idx := vs.IndexOf(nAddr); idx >= 0 {
		return errors.IllegalArgumentError.New("ValidatorInUse")
	}

	return vs.setAtInLock(i, ov, v)
}

// setAtInLock() assumes that all arguments are valid
func (vs *validatorState) setAtInLock(i int, ov, nv module.Validator) error {
	vo, err := validatorFromValidator(nv)
	if err != nil {
		return err
	}
	vs.becomeChangeableInLock()
	vs.validators[i] = vo
	if vs.addrMap != nil {
		delete(vs.addrMap, string(ov.Address().Bytes()))
		vs.addrMap[string(nv.Address().Bytes())] = i
	}
	return nil
}

func ValidatorStateFromHash(database db.Database, h []byte) (ValidatorState, error) {
	vss, err := ValidatorSnapshotFromHash(database, h)
	if err != nil {
		return nil, err
	}
	return ValidatorStateFromSnapshot(vss), nil
}

func ValidatorSnapshotFromHash(database db.Database, h []byte) (ValidatorSnapshot, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	return validatorSnapshotFromHash(bk, h)
}

func validatorSnapshotFromHash(bk db.Bucket, h []byte) (*validatorSnapshot, error) {
	vss := &validatorSnapshot{
		validatorList: &validatorList{
			bucket: bk,
		},
	}
	if len(h) > 0 {
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, errors.NotFoundError.New("InvalidHashValue")
		}
		_, err = codec.BC.UnmarshalFromBytes(value, &vss.validators)
		if err != nil {
			return nil, err
		}
		vss.hash = h
		vss.serialized = value
	}
	return vss, nil
}

func NewValidatorSnapshotWithBuilder(builder merkle.Builder, h []byte) (ValidatorSnapshot, error) {
	bk, err := builder.Database().GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vss := &validatorSnapshot{
		validatorList: &validatorList{
			bucket: bk,
		},
	}
	if len(h) > 0 {
		vss.hash = h
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if value == nil {
			builder.RequestData(db.BytesByHash, h, vss)
			vss.dirty = true
		} else {
			if _, err := codec.UnmarshalFromBytes(value, &vss.validators); err != nil {
				return nil, err
			}
			vss.serialized = value
		}
	}
	return vss, nil
}

func ValidatorSnapshotFromSlice(database db.Database, vl []module.Validator) (ValidatorSnapshot, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vss := &validatorSnapshot{
		validatorList: &validatorList{
			bucket: bk,
		},
		dirty: true,
	}
	vList := make([]*validator, len(vl))
	for i, v := range vl {
		if vo, ok := v.(*validator); ok {
			vList[i] = vo
		} else {
			return nil, errors.ErrIllegalArgument
		}
	}
	vss.validators = vList

	return vss, nil
}

func ValidatorStateFromSnapshot(vss ValidatorSnapshot) ValidatorState {
	snapshot, ok := vss.(*validatorSnapshot)
	if !ok {
		log.Panicf("InvalidValidatorSnapshot(hash=<%x>)", vss.Hash())
	}
	vs := &validatorState{
		validatorList: snapshot.validatorList,
		snapshot:      snapshot,
	}
	return vs
}
