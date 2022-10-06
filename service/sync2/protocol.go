package sync2

import (
	"fmt"

	"github.com/icon-project/goloop/module"
)

// protocol message codes
const (
	protoHasNode module.ProtocolInfo = iota
	protoResult
	protoRequestNodeData
	protoNodeData
)

var protocol = []module.ProtocolInfo{
	protoHasNode,
	protoResult,
	protoRequestNodeData,
	protoNodeData,
}

type errCode int

const (
	NoError errCode = iota
	ErrTimeExpired
	ErrNoData
)

func (e errCode) String() string {
	switch e {
	case NoError:
		return "NoError"
	case ErrTimeExpired:
		return "ErrTimeExpired"
	case ErrNoData:
		return "ErrNoData"
	default:
		return fmt.Sprintf("Unknown(%d)", e)
	}
}

type hasNode struct {
	ReqID         uint32
	StateHash     []byte
	ValidatorHash []byte
	PatchHash     []byte
	NormalHash    []byte
}

func (r *hasNode) String() string {
	return fmt.Sprintf("ReqID(%d), StateHash(%#x), ValidatorHash(%#x), patchHash(%#x), NormalHash(%#x)",
		r.ReqID, r.StateHash, r.ValidatorHash, r.PatchHash, r.NormalHash)
}

type result struct {
	ReqID  uint32
	Status errCode
}

func (r *result) String() string {
	return fmt.Sprintf("ReqID(%d), Status(%d)",
		r.ReqID, r.Status)
}

type requestNodeData struct {
	ReqID  uint32
	Type   syncType
	Hashes [][]byte
}

func (r *requestNodeData) String() string {
	return fmt.Sprintf("ReqID(%d), Hashes(%#x)",
		r.ReqID, r.Hashes)
}

type nodeData struct {
	ReqID  uint32
	Status errCode
	Type   syncType
	Data   [][]byte
}

func (r *nodeData) String() string {
	return fmt.Sprintf("ReqID(%d), Status(%d), Data(%#x)",
		r.ReqID, r.Status, r.Data)
}
