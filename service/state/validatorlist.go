package state

import (
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type ValidatorList interface {
	module.ValidatorList
	Copy() ValidatorList
	Add(v module.Validator) error
	Remove(v module.Validator) bool
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

func (vl *validatorList) Flush() error {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	if vl.dirty {
		data := vl.serializeInLock()
		key := vl.hashInLock()
		return vl.bucket.Set(key, data)
	}
	return nil
}

func (vl *validatorList) Len() int {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	return len(vl.validators)
}

func (vl *validatorList) Copy() ValidatorList {
	nvl := new(validatorList)
	nvl.lock = sync.Mutex{}
	nvl.bucket = vl.bucket
	copy(nvl.validators, vl.validators)
	copy(nvl.serialized, vl.serialized)
	copy(nvl.hash, vl.hash)
	nvl.dirty = vl.dirty
	// Don't copy because the current map may change to be useless.
	nvl.addrMap = nil
	return nvl
}
func (vl *validatorList) Add(v module.Validator) error {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	if vl.indexOfInLock(v.Address()) < 0 {
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

		vl.validators = append(vl.validators, vo)
		vl.dirty = true
		vl.addrMap = nil
	}
	return nil
}

func (vl *validatorList) Remove(v module.Validator) bool {
	vl.lock.Lock()
	defer vl.lock.Unlock()

	i := vl.indexOfInLock(v.Address())
	if i < 0 {
		return false
	}
	vl.validators = append(vl.validators[:i], vl.validators[i+1:]...)
	vl.dirty = true
	vl.addrMap = nil
	return true
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
	return fmt.Sprintf("ValidatorList[%+v]", vl.validators)
}

func (vl *validatorList) OnData(bs []byte, bd merkle.Builder) error {
	vl.serialized = bs
	if _, err := codec.MP.UnmarshalFromBytes(bs, &vl.validators); err != nil {
		return err
	}
	return nil
}

func ValidatorListFromHash(database db.Database, h []byte) (ValidatorList, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vl := &validatorList{
		bucket: bk,
		dirty:  false,
	}
	if len(h) > 0 {
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, errors.New("InvalidHashValue")
		}
		_, err = codec.MP.UnmarshalFromBytes(value, &vl.validators)
		if err != nil {
			return nil, err
		}
		vl.hash = h
		vl.serialized = value
	}
	return vl, nil
}

func NewValidatorListWithBuilder(builder merkle.Builder, h []byte) (ValidatorList, error) {
	bk, err := builder.Database().GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	vl := &validatorList{
		bucket: bk,
	}
	if len(h) > 0 {
		vl.hash = h
		value, err := bk.Get(h)
		if err != nil {
			return nil, err
		}
		if value == nil {
			builder.RequestData(db.BytesByHash, h, vl)
			vl.dirty = true
		} else {
			if _, err := codec.UnmarshalFromBytes(value, &vl.validators); err != nil {
				return nil, err
			}
			vl.serialized = value
		}
	}
	return vl, nil
}

func ValidatorListFromSlice(database db.Database, vl []module.Validator) (ValidatorList, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	nvl := &validatorList{
		bucket:     bk,
		validators: nil,
		serialized: nil,
		hash:       nil,
		dirty:      true,
	}

	vList := make([]*validator, len(vl))
	for i, v := range vl {
		if vo, ok := v.(*validator); ok {
			vList[i] = vo
		} else {
			return nil, errors.New("NotCompatibleValidator")
		}
	}
	nvl.validators = vList

	return nvl, nil
}
