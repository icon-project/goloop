package state

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/service/scoreapi"
)

type apiInfoStore struct {
	bk    db.Bucket
	dirty bool
	info  *scoreapi.Info
	hash  []byte
	bytes []byte
}

func (s *apiInfoStore) getHash() []byte {
	if s.hash == nil && s.info != nil {
		bs, err := codec.BC.MarshalToBytes(s.info)
		if err != nil {
			log.Panicf("Fail to encode api info info=%+v err=%+v",
				s.info, err)
		}
		s.bytes = bs
		s.hash = crypto.SHA3Sum256(bs)
	}
	return s.hash
}

func (s *apiInfoStore) Get() (*scoreapi.Info, error) {
	if s.bk != nil {
		if len(s.hash) > 0 && s.info == nil {
			bs, err := s.bk.Get(s.hash)
			if err != nil {
				return nil, errors.CriticalIOError.Wrapf(err, "FailToGetAPIInfo(hash=%x)", s.hash)
			}
			_, err = codec.UnmarshalFromBytes(bs, &s.info)
			if err != nil {
				return nil, errors.CriticalFormatError.Wrapf(err, "InvalidAPIInfo(hash=%x)", s.hash)
			}
			s.bytes = bs
		}
	}
	return s.info, nil
}

func (s *apiInfoStore) Set(info *scoreapi.Info) {
	s.hash = nil
	s.bytes = nil
	s.info = info
	s.dirty = true
}

func (s *apiInfoStore) Equal(s2 *apiInfoStore) bool {
	if s.bk == nil && s2.bk == nil {
		return s.info.Equal(s2.info)
	}
	if s.bk == nil || s2.bk == nil {
		return false
	}
	return bytes.Equal(s.getHash(), s2.getHash())
}

func (s *apiInfoStore) Flush() error {
	if s.dirty && s.bk != nil {
		h := s.getHash()
		err := s.bk.Set(h, s.bytes)
		if err != nil {
			return errors.CriticalIOError.Wrap(err, "FailToSetAPIInfo")
		}
	}
	return nil
}

func (s *apiInfoStore) ResetDB(b db.Database) error {
	if bk, err := b.GetBucket(db.BytesByHash); err != nil {
		return err
	} else {
		s.bk = bk
		return nil
	}
}

func (s *apiInfoStore) RLPEncodeSelf(e codec.Encoder) error {
	if s.bk == nil {
		return e.Encode(s.info)
	} else {
		hv := s.getHash()
		return e.Encode(hv)
	}
}

func (s *apiInfoStore) RLPDecodeSelf(d codec.Decoder) error {
	if s.bk == nil {
		return d.Decode(&s.info)
	} else {
		return d.Decode(&s.hash)
	}
}

func (s *apiInfoStore) Resolve(bd merkle.Builder) error {
	if s.bk == nil {
		return nil
	}
	if len(s.hash) > 0 {
		value, err := s.bk.Get(s.hash)
		if err != nil {
			return err
		}
		if value == nil {
			bd.RequestData(db.BytesByHash, s.hash, s)
			return nil
		}
		if _, err = codec.UnmarshalFromBytes(value, &s.info); err != nil {
			return err
		}
		s.bytes = value
	}
	return nil
}

func (s *apiInfoStore) OnData(value []byte, builder merkle.Builder) error {
	_, err := codec.UnmarshalFromBytes(value, &s.info)
	if err != nil {
		return errors.CriticalFormatError.Wrapf(err, "InvalidAPIInfo(hash=%x)", s.hash)
	}
	s.bytes = value
	return nil
}
