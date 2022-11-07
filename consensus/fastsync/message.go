package fastsync

import (
	"io"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

// TODO: close message
const (
	ProtoBlockRequest module.ProtocolInfo = iota << 8
	ProtoBlockMetadata
	ProtoBlockData
	ProtoCancelAllBlockRequests
)

var protocols = []module.ProtocolInfo{
	ProtoBlockRequest,
	ProtoBlockMetadata,
	ProtoBlockData,
	ProtoCancelAllBlockRequests,
}

type BlockRequestV1 struct {
	RequestID uint32
	Height    int64
}

type BlockRequestV2 struct {
	RequestID   uint32
	Height      int64
	ProofOption int32
}

type BlockRequest = BlockRequestV2

func (m *BlockRequest) RLPEncodeSelf(e codec.Encoder) error {
	var err error
	if m.ProofOption == 0 {
		err = e.EncodeListOf(m.RequestID, m.Height)
	} else {
		err = e.EncodeListOf(m.RequestID, m.Height, m.ProofOption)
	}
	return err
}

func (m *BlockRequest) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	cnt, err := d2.DecodeMulti(&m.RequestID, &m.Height, &m.ProofOption)
	if cnt == 2 && err == io.EOF {
		m.ProofOption = 0
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

type BlockMetadata struct {
	RequestID   uint32
	BlockLength int32 // -1 if fails
	Proof       []byte
}

type BlockData struct {
	RequestID uint32
	Data      []byte
}

type CancelAllBlockRequests struct {
}
