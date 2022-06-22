package sync2

import (
	"fmt"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

// protocol message codes
const (
	protoV2Request module.ProtocolInfo = iota
	protoV2Response
)

var protocolv2 = []module.ProtocolInfo{
	protoV2Request,
	protoV2Response,
}

type BucketIDAndBytes struct {
	BkID  db.BucketID
	Bytes []byte
}

func (b BucketIDAndBytes) String() string {
	return fmt.Sprintf("{BkID:%s, Bytes:%#x}", b.BkID, b.Bytes)
}

type requestData struct {
	ReqID uint32
	Data  []BucketIDAndBytes
}

func (r *requestData) String() string {
	return fmt.Sprintf("ReqID(%d), Data(%+v)", r.ReqID, r.Data)
}

type responseData struct {
	ReqID  uint32
	Status errCode
	Data   []BucketIDAndBytes
}

func (r *responseData) String() string {
	return fmt.Sprintf("ReqID(%d), Status(%d), Data(%+v)",
		r.ReqID, r.Status, r.Data)
}
