package state

import (
	"fmt"
	"log"
	"sync"

	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type ValidatorSnapshot module.ValidatorList

type ValidatorState interface {
	Hash() []byte
	Bytes() []byte
	IndexOf(module.Address) int
	Len() int
	Get(i int) (module.Validator, bool)
	Set([]module.Validator) error
	Add(v module.Validator) error
	Remove(v module.Validator) bool
	GetSnapshot() ValidatorSnapshot
	Reset(ValidatorSnapshot)
}

type validatorList struct {
	lock       sync.Mutex
	bucket     db.Bucket
	validators []*validator
	serialized []byte
	hash       []byte
	dirty      bool
	addrMap    map[string]int
}

func (vl *validatorList) serializeInLock() []byte {
	if vl.serialized == nil && len(vl.validators) > 0 {
		vl.serialized, _ = codec.MP.MarshalToBytes(vl.validators)
	}
	return vl.serialized
}

func (vl *validatorList) hashInLock() []byte {
	if vl.hash == nil && len(vl.validators) > 0 {
		s := vl.serializeInLock()
		vl.hash = crypto.SHA3Sum256(s)
	}
	return vl.hash
}

func (vl *validatorList) Hash() []byte {
	vl.lock.Lock()
	defer vl.lock.Unlock()
	return vl.hashInLock()
}

func (vl *validatorList) Bytes() []byte {
	vl.lock.Lock()
	defer vl.lock.Unlock()
	return vl.serializeInLock()
}

func (vl *validatorList) Len() int {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return len(vl.validators)
}

func (vl *validatorList) Get(i int) (module.Validator, bool) {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	if i < 0 || i >= len(vl.validators) {
		return nil, false
	}
	return vl.validators[i], true
}

func (vl *validatorList) IndexOf(addr module.Address) int {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return vl.indexOfInLock(addr)
}

func (vl *validatorList) indexOfInLock(addr module.Address) int {
	if vl.addrMap == nil {
		vl.addrMap = make(map[string]int)
		for i, v := range vl.validators {
			vl.addrMap[string(v.Address().Bytes())] = i
		}
	}
	if idx, ok := vl.addrMap[string(addr.Bytes())]; ok {
		return idx
	}
	return -1
}

func (vl *validatorList) String() string {
	return fmt.Sprintf("validatorList[%+v]", vl.validators)
}

func (vl *validatorList) OnData(bs []byte, bd merkle.Builder) error {
	vl.serialized = bs
	if _, err := codec.MP.UnmarshalFromBytes(bs, &vl.validators); err != nil {
		return err
	}
	return nil
}

type validatorSnapshot struct {
	*validatorList
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
}

func (vs *validatorState) GetSnapshot() ValidatorSnapshot {
	vs.lock.Lock()
	defer vs.lock.Unlock()

	vl := new(validatorList)
	vl.lock = sync.Mutex{}
	vl.bucket = vs.bucket
	vl.validators = make([]*validator, len(vs.validators))
	copy(vl.validators, vs.validators)
	vl.dirty = vs.dirty
	vl.addrMap = vs.addrMap
	return &validatorSnapshot{vl}
}

func (vs *validatorState) Reset(vss ValidatorSnapshot) {
	snapshot, ok := vss.(*validatorSnapshot)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", vss)
	}
	vs.bucket = snapshot.bucket
	vs.validators = make([]*validator, len(snapshot.validators))
	copy(vs.validators, snapshot.validators)
	vs.dirty = snapshot.dirty
	vs.serialized = snapshot.serialized
	vs.hash = snapshot.hash
	vs.addrMap = snapshot.addrMap
}

func (vs *validatorState) Set(vl []module.Validator) error {
	vs.lock.Lock()
	defer vs.lock.Unlock()

	vList := make([]*validator, len(vl))
	for i, v := range vl {
		if vo, ok := v.(*validator); ok {
			vList[i] = vo
		} else {
			return errors.New("NotCompatibleValidator")
		}
	}
	vs.validators = vList
	vs.markChange()
	return nil
}

func (vs *validatorState) Add(v module.Validator) error {
	vs.lock.Lock()
	defer vs.lock.Unlock()

	if vs.indexOfInLock(v.Address()) < 0 {
		var vo *validator
		var ok bool
		if vo, ok = v.(*validator); !ok {
			var vi module.Validator
			var err error
			if vi, err = ValidatorFromPublicKey(v.PublicKey()); err != nil {
				if vi, err = ValidatorFromAddress(v.Address()); err != nil {
					return err
				}
			}
			vo = vi.(*validator)
		}

		vs.validators = append(vs.validators, vo)
		vs.markChange()
	}
	return nil
}

func (vs *validatorState) markChange() {
	vs.dirty = true
	vs.hash = nil
	vs.serialized = nil
	vs.addrMap = nil
}

func (vs *validatorState) Remove(v module.Validator) bool {
	vs.lock.Lock()
	defer vs.lock.Unlock()

	i := vs.indexOfInLock(v.Address())
	if i < 0 {
		return false
	}
	vs.validators = append(vs.validators[:i], vs.validators[i+1:]...)
	vs.markChange()
	return true
}

func ValidatorStateFromHash(database db.Database, h []byte) (ValidatorState, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vs := &validatorState{
		&validatorList{
			bucket: bk,
			dirty:  false,
		},
	}
	if len(h) > 0 {
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, errors.New("InvalidHashValue")
		}
		_, err = codec.MP.UnmarshalFromBytes(value, &vs.validators)
		if err != nil {
			return nil, err
		}
		vs.hash = h
		vs.serialized = value
	}
	return vs, nil
}
func ValidatorSnapshotFromHash(database db.Database, h []byte) (ValidatorSnapshot, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vss := &validatorSnapshot{
		&validatorList{
			bucket: bk,
			dirty:  false,
		},
	}
	if len(h) > 0 {
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, errors.New("InvalidHashValue")
		}
		_, err = codec.MP.UnmarshalFromBytes(value, &vss.validators)
		if err != nil {
			return nil, err
		}
		vss.hash = h
		vss.serialized = value
	}
	return vss, nil
}

func NewValidatorStateWithBuilder(builder merkle.Builder, h []byte) (ValidatorState, error) {
	bk, err := builder.Database().GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vs := &validatorState{
		&validatorList{
			bucket: bk,
		},
	}
	if len(h) > 0 {
		vs.hash = h
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if value == nil {
			builder.RequestData(db.BytesByHash, h, vs)
			vs.dirty = true
		} else {
			if _, err := codec.UnmarshalFromBytes(value, &vs.validators); err != nil {
				return nil, err
			}
			vs.serialized = value
		}
	}
	return vs, nil
}

func ValidatorSnapshotFromSlice(database db.Database, vl []module.Validator) (ValidatorSnapshot, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vs := &validatorSnapshot{
		&validatorList{
			bucket:     bk,
			validators: nil,
			serialized: nil,
			hash:       nil,
			dirty:      true,
		},
	}

	vList := make([]*validator, len(vl))
	for i, v := range vl {
		if vo, ok := v.(*validator); ok {
			vList[i] = vo
		} else {
			return nil, errors.New("NotCompatibleValidator")
		}
	}
	vs.validators = vList

	return vs, nil
}

func ValidatorStateFromSnapshot(vss ValidatorSnapshot) ValidatorState {
	snapshot, ok := vss.(*validatorSnapshot)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", vss)
	}
	vs := &validatorState{
		&validatorList{
			bucket:     snapshot.bucket,
			serialized: snapshot.serialized,
			hash:       snapshot.hash,
			dirty:      snapshot.dirty,
			addrMap:    snapshot.addrMap,
		},
	}
	vs.validators = make([]*validator, len(snapshot.validators))
	copy(vs.validators, snapshot.validators)
	return vs
}
