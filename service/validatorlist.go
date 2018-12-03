package service

import (
	"fmt"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	"sync"
)

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

func ValidatorListFromHash(database db.Database, h []byte) (module.ValidatorList, error) {
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

func ValidatorListFromSlice(database db.Database, vl []module.Validator) (module.ValidatorList, error) {
	bk, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	nvl := &validatorList{
		lock:       sync.Mutex{},
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
