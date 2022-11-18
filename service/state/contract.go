package state

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

type ContractStatus int

const (
	CSInactive ContractStatus = 1 << iota
	CSActive
	CSPending
	CSRejected
)

func (cs ContractStatus) String() string {
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

const (
	CTAppZip    = "application/zip"
	CTAppJava   = "application/java"
	CTAppSystem = "application/x.score.system"
)

type ContractSnapshot interface {
	CodeID() []byte
	CodeHash() []byte
	Code() ([]byte, error)
	EEType() EEType
	ContentType() string
	DeployTxHash() []byte
	AuditTxHash() []byte
	Params() []byte
	Status() ContractStatus
	Equal(s ContractSnapshot) bool
}

type contract struct {
	bk           db.Bucket
	needFlush    bool
	state        ContractStatus
	contentType  string
	eeType       EEType
	deployTxHash []byte
	auditTxHash  []byte
	codeHash     []byte
	code         []byte
	params       []byte
	markDirty    func()
}

func (c *contract) Equal(s ContractSnapshot) bool {
	if c2, ok := s.(*contract); ok {
		if c == c2 {
			return true
		}
		if c == nil || c2 == nil {
			return false
		}
		return c.state == c2.state &&
			bytes.Equal(c.deployTxHash, c2.deployTxHash) &&
			bytes.Equal(c.auditTxHash, c2.auditTxHash) &&
			bytes.Equal(c.codeHash, c2.codeHash)
	} else {
		log.Panicf("Invalid object")
	}

	return true
}

func (c *contract) CodeHash() []byte {
	if c == nil {
		return nil
	}
	return c.codeHash
}

func (c *contract) Code() ([]byte, error) {
	if c.code == nil {
		if len(c.codeHash) == 0 {
			return nil, nil
		}
		code, err := c.bk.Get(c.codeHash)
		if err != nil {
			return nil, err
		}
		if code == nil {
			return nil, errors.NotFoundError.Errorf(
				"FAIL to find code by codeHash(%x)", c.codeHash)
		}
		c.code = code
	}
	return c.code, nil
}

func (c *contract) EEType() EEType {
	return c.eeType
}

func (c *contract) ContentType() string {
	return c.contentType
}

func (c *contract) DeployTxHash() []byte {
	return c.deployTxHash
}

func (c *contract) CodeID() []byte {
	if len(c.deployTxHash) > 0 {
		return c.deployTxHash
	} else {
		return crypto.SHA3Sum256(codec.BC.MustMarshalToBytes(c))
	}
}

func (c *contract) AuditTxHash() []byte {
	return c.auditTxHash
}

func (c *contract) Params() []byte {
	return c.params
}

func (c *contract) Status() ContractStatus {
	return c.state
}

func (c *contract) RLPEncodeSelf(e codec.Encoder) error {
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

func (c *contract) RLPDecodeSelf(d codec.Decoder) error {
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

func (c *contract) flush() error {
	if c.needFlush == false {
		return nil
	}
	if err := c.bk.Set(c.codeHash, c.code); err != nil {
		return err
	}
	c.needFlush = false
	return nil
}

func (c *contract) OnData(bs []byte, builder merkle.Builder) error {
	c.code = bs
	return nil
}

func (c *contract) Resolve(builder merkle.Builder) error {
	if len(c.codeHash) > 0 {
		code, err := c.bk.Get(c.codeHash)
		if err != nil {
			return err
		}
		if code == nil {
			builder.RequestData(db.BytesByHash, c.codeHash, c)
		} else {
			c.code = code
		}
	}
	return nil
}

func (c *contract) ResetDB(dbase db.Database) error {
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

func (c *contract) String() string {
	return fmt.Sprintf("Contract{hash=%#x ee=%s deploy=%#x audit=%#x}",
		c.codeHash, c.eeType, c.deployTxHash, c.auditTxHash)
}

type ContractState interface {
	ContractSnapshot
	SetCode([]byte) error
}

func (c *contract) SetCode(code []byte) error {
	if c.markDirty == nil {
		panic("SetCodeOnSnapshot")
	}
	if len(code) == 0 {
		c.code = nil
		c.codeHash = nil
		c.markDirty()
		return nil
	}
	codeHash := crypto.SHA3Sum256(code)
	if bytes.Equal(codeHash, c.codeHash) {
		return nil
	}
	c.code = code
	c.codeHash = codeHash
	c.needFlush = true
	c.markDirty()
	return nil
}

func (c *contract) getSnapshot() *contract {
	return c.cloneWithMarkDirty(nil)
}

func (c *contract) cloneWithMarkDirty(markDirty func()) *contract {
	if c == nil {
		return nil
	}
	nc := new(contract)
	*nc = *c
	nc.markDirty = markDirty
	return nc
}

type contractROState struct {
	ContractSnapshot
}

func (c *contractROState) SetCode(code []byte) error {
	log.Panicf("contractROState().SetCode() is invoked")
	return errors.InvalidStateError.New("ReadOnlyContract")
}

func newContractROState(snapshot ContractSnapshot) ContractState {
	if snapshot == nil {
		return nil
	}
	return &contractROState{snapshot}
}

func newContractState(snapshot *contract, markDirty func()) *contract {
	return snapshot.cloneWithMarkDirty(markDirty)
}
