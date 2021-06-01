package fastsync

import "github.com/icon-project/goloop/module"

// TODO: close message
const (
	protoBlockRequest module.ProtocolInfo = iota << 8
	protoBlockMetadata
	protoBlockData
	protoCancelAllBlockRequests
)

var protocols = []module.ProtocolInfo{
	protoBlockRequest,
	protoBlockMetadata,
	protoBlockData,
	protoCancelAllBlockRequests,
}

type BlockRequest struct {
	RequestID uint32
	Height    int64
}

type BlockMetadata struct {
	RequestID   uint32
	BlockLength int32 // -1 if fails
	VoteList    []byte
}

type BlockData struct {
	RequestID uint32
	Data      []byte
}

type CancelAllBlockRequests struct {
}
