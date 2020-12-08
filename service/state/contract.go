package state

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

type ContractState int

const (
	CSInactive ContractState = 1 << iota
	CSActive
	CSPending
	CSRejected
)

func (cs ContractState) String() string {
	var status string
	switch cs {
	case CSInactive:
		status = "inactive"
	case CSActive:
		status = "active"
	case CSPending:
		status = "pending"
	case CSRejected:
		status = "rejected"
	default:
		status = fmt.Sprintf("Unknown(state=%d)", cs)
	}
	return status
}

const ASActive = 0
const (
	ASDisabled = 1 << iota
	ASBlocked
)

const (
	CTAppZip    = "application/zip"
	CTAppJava   = "application/java"
	CTAppSystem = "application/x.score.system"
)

type ContractSnapshot interface {
	CodeID() []byte
	CodeHash() []byte
	Code() ([]byte, error)
	SetCode([]byte) error
	EEType() EEType
	ContentType() string
	DeployTxHash() []byte
	AuditTxHash() []byte
	Params() []byte
	Status() ContractState
	Equal(s ContractSnapshot) bool
}

type contractSnapshotImpl struct {
	bk           db.Bucket
	isNew        bool
	state        ContractState
	contentType  string
	eeType       EEType
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
		if c.state != c2.state {
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
	if c == nil {
		return nil
	}
	return c.codeHash
}

func (c *contractSnapshotImpl) Code() ([]byte, error) {
	if len(c.code) == 0 {
		if len(c.codeHash) == 0 {
			return nil, nil
		}
		code, err := c.bk.Get(c.codeHash)
		if err != nil {
			return nil, err
		}
		if len(code) == 0 {
			return nil, errors.NotFoundError.Errorf(
				"FAIL to find code by codeHash(%x)", c.codeHash)
		}
		c.code = code
	}
	return c.code, nil
}

func (c *contractSnapshotImpl) SetCode(code []byte) error {
	if len(code) == 0 {
		c.code = nil
		c.codeHash = nil
		return nil
	}
	c.code = code
	codeHash := sha3.Sum256(code)
	c.codeHash = codeHash[:]
	return nil
}

func (c *contractSnapshotImpl) EEType() EEType {
	return c.eeType
}

func (c *contractSnapshotImpl) ContentType() string {
	return c.contentType
}

func (c *contractSnapshotImpl) DeployTxHash() []byte {
	return c.deployTxHash
}

func (c *contractSnapshotImpl) CodeID() []byte {
	if len(c.deployTxHash) > 0 {
		return c.deployTxHash
	} else {
		return crypto.SHA3Sum256(codec.BC.MustMarshalToBytes(c))
	}
}

func (c *contractSnapshotImpl) AuditTxHash() []byte {
	return c.auditTxHash
}

func (c *contractSnapshotImpl) Params() []byte {
	return c.params
}

func (c *contractSnapshotImpl) Status() ContractState {
	return c.state
}

const (
	contractSnapshotImplEntries = 7
)

func (c *contractSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		c.state,
		c.contentType,
		c.eeType,
		c.deployTxHash,
		c.auditTxHash,
		c.codeHash,
		c.params,
	)
}

func (c *contractSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(
		&c.state,
		&c.contentType,
		&c.eeType,
		&c.deployTxHash,
		&c.auditTxHash,
		&c.codeHash,
		&c.params,
	)
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
		return nil
	}
	if err := c.bk.Set(c.codeHash, c.code); err != nil {
		return err
	}
	c.isNew = false
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

func (c *contractSnapshotImpl) ResetDB(dbase db.Database) error {
	if c == nil {
		return nil
	}
	if bk, err := dbase.GetBucket(db.BytesByHash); err != nil {
		return errors.CriticalIOError.Wrap(err, "FailToGetBucket")
	} else {
		c.bk = bk
		return nil
	}
}

type Contract interface {
	ContractSnapshot
	SetStatus(state ContractState)
}

type contractImpl struct {
	contractSnapshotImpl
}

func (c *contractImpl) SetStatus(state ContractState) {
	c.state = state
}

func (c *contractImpl) getSnapshot() *contractSnapshotImpl {
	if c == nil {
		return nil
	}
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

func (c *contractROState) SetStatus(state ContractState) {
	log.Panicf("contractROState().SetStatus() is invoked")
}

func newContractROState(snapshot ContractSnapshot) Contract {
	if snapshot == nil {
		return nil
	}
	return &contractROState{snapshot}
}
