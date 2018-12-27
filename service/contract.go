package service

import (
	"bytes"
	"github.com/icon-project/goloop/common/merkle"
	"log"

	"github.com/icon-project/goloop/common/db"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
)

type contractStatus int

const (
	csInactive contractStatus = 1 << iota
	csActive
	csPending
	csRejected
	csBlacklist
	csDisable
)

type ContractSnapshot interface {
	CodeHash() []byte
	Code() ([]byte, error)
	EEType() string
	ContentType() string
	Params() []byte
	Status() contractStatus
	Equal(s ContractSnapshot) bool
}
type contractSnapshotImpl struct {
	bk           db.Bucket
	isNew        bool
	status       contractStatus
	contentType  string
	eeType       string
	deployTxHash []byte
	auditTxHash  []byte
	codeHash     []byte
	code         []byte
	params       []byte
}

func (c *contractSnapshotImpl) Equal(s ContractSnapshot) bool {
	if c2, ok := s.(*contractSnapshotImpl); ok {
		if c == c2 {
			return true
		}
		if c == nil || c2 == nil {
			return false
		}
		if c.status != c2.status {
			return false
		}
		if bytes.Equal(c.codeHash, c2.codeHash) == false {
			return false
		}
	} else {
		log.Panicf("Invalid object")
	}

	return true
}

func (c *contractSnapshotImpl) CodeHash() []byte {
	return c.codeHash
}

func (c *contractSnapshotImpl) Code() ([]byte, error) {
	if len(c.code) == 0 {
		code, err := c.bk.Get(c.codeHash)
		if err != nil {
			return nil, err
		}
		if len(code) == 0 {
			return nil, errors.New("Failed to find code by codeHash")
		}
		c.code = code
	}
	return c.code, nil
}

func (c *contractSnapshotImpl) EEType() string {
	return c.eeType
}

func (c *contractSnapshotImpl) ContentType() string {
	return c.contentType
}

func (c *contractSnapshotImpl) Params() []byte {
	return c.params
}

func (c *contractSnapshotImpl) Status() contractStatus {
	return c.status
}

func (c *contractSnapshotImpl) CodecEncodeSelf(e *ugorji.Encoder) {
	_ = e.Encode(c.status)
	_ = e.Encode(c.contentType)
	_ = e.Encode(c.eeType)
	_ = e.Encode(c.deployTxHash)
	_ = e.Encode(c.auditTxHash)
	_ = e.Encode(c.codeHash)
	_ = e.Encode(c.params)
}

func (c *contractSnapshotImpl) CodecDecodeSelf(d *ugorji.Decoder) {
	if err := d.Decode(&c.status); err != nil {
		log.Fatalf("Fail to decode status in account")
	}
	if err := d.Decode(&c.contentType); err != nil {
		log.Fatalf("Fail to decode contentType in account")
	}
	if err := d.Decode(&c.eeType); err != nil {
		log.Fatalf("Fail to decode eeType in account")
	}
	if err := d.Decode(&c.deployTxHash); err != nil {
		log.Fatalf("Fail to decode deployTxHash in account")
	}
	if err := d.Decode(&c.auditTxHash); err != nil {
		log.Fatalf("Fail to decode auditTxHash in account")
	}
	if err := d.Decode(&c.codeHash); err != nil {
		log.Fatalf("Fail to decode codeHash in account")
	}
	if err := d.Decode(&c.params); err != nil {
		log.Fatalf("Fail to decode params in account, err = %s", err)
	}
}

func (c *contractSnapshotImpl) flush() error {
	if c.isNew == false {
		return nil
	}
	code, err := c.bk.Get(c.codeHash)
	if err != nil {
		return err
	}
	if len(code) != 0 {
		return errors.New("Code already exists")
	}
	if err := c.bk.Set(c.codeHash, c.code); err != nil {
		return err
	}
	return nil
}

func (c *contractSnapshotImpl) OnData(bs []byte, builder merkle.Builder) error {
	c.code = bs
	return nil
}

func (c *contractSnapshotImpl) Resolve(builder merkle.Builder) error {
	code, err := c.bk.Get(c.codeHash)
	if err != nil {
		return err
	}
	if code == nil {
		builder.RequestData(db.BytesByHash, c.codeHash, c)
	} else {
		c.code = code
	}
	return nil
}

type Contract interface {
	ContractSnapshot
	SetStatus(status contractStatus)
}

type contractImpl struct {
	contractSnapshotImpl
}

func (c *contractImpl) SetStatus(status contractStatus) {
	c.status = status
}

func (c *contractImpl) getSnapshot() *contractSnapshotImpl {
	var snapshot contractSnapshotImpl
	snapshot = c.contractSnapshotImpl
	return &snapshot
}

func (c *contractImpl) reset(snapshot *contractSnapshotImpl) {
	c.contractSnapshotImpl = *snapshot
}

type contractROState struct {
	ContractSnapshot
}

func (c *contractROState) SetStatus(status contractStatus) {
	log.Panicf("contractROState().SetStatus() is invoked")
}

func newContractROState(snapshot ContractSnapshot) Contract {
	return &contractROState{snapshot}
}
