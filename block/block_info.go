package block

import (
	"time"

	"github.com/icon-project/goloop/module"
)

type blockInfo struct {
	height    int64
	timestamp int64
}

func (bi blockInfo) Height() int64 {
	return bi.height
}

func (bi blockInfo) Timestamp() int64 {
	return bi.timestamp
}

func newBlockInfo(height int64, timestamp time.Time) *blockInfo {
	return &blockInfo{
		height:    height,
		timestamp: unixMicroFromTime(timestamp),
	}
}

func newBlockInfoFromBlock(block module.Block) *blockInfo {
	return &blockInfo{
		height:    block.Height(),
		timestamp: unixMicroFromTime(block.Timestamp()),
	}
}
