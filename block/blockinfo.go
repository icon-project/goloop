package block

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

func newBlockInfo(height int64, timestamp int64) *blockInfo {
	return &blockInfo{
		height:    height,
		timestamp: timestamp,
	}
}
