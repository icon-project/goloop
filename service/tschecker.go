package service

import (
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	ConfigTXTimestampThresholdDefault = int64(5 * time.Minute / time.Microsecond)
	ConfigPatchTimestampThreshold     = int64(1 * time.Minute / time.Microsecond)
)

func CheckTxTimestamp(min, max int64, tx transaction.Transaction) error {
	ts := tx.Timestamp()
	if ts <= min {
		return ExpiredTransactionError.Errorf("Expired(min-%s)",
			time.Duration(min-ts)*time.Microsecond)
	} else if ts > max {
		return FutureTransactionError.Errorf("FutureTx(max+%s)",
			time.Duration(ts-max)*time.Microsecond)
	}
	return nil
}

type TxTimestampChecker struct {
	threshold int64
}

func (c *TxTimestampChecker) CheckWithCurrent(min int64, tx transaction.Transaction) error {
	return CheckTxTimestamp(min, (time.Now().UnixNano()/1000)+c.Threshold(), tx)
}

func (c *TxTimestampChecker) SetThreshold(d time.Duration) {
	log.Debugf("SetThreshold:%s", d)
	atomic.StoreInt64(&c.threshold, DurationToTimestamp(d))
}

func (c *TxTimestampChecker) Threshold() int64 {
	return atomic.LoadInt64(&c.threshold)
}

func (c *TxTimestampChecker) TransactionThreshold(group module.TransactionGroup) int64 {
	if group == module.TransactionGroupNormal {
		return c.Threshold()
	} else {
		return ConfigPatchTimestampThreshold
	}
}

func TimestampToDuration(t int64) time.Duration {
	return time.Duration(t) * time.Microsecond
}

func DurationToTimestamp(d time.Duration) int64 {
	return int64(d / time.Microsecond)
}

func NewTimestampChecker() *TxTimestampChecker {
	return &TxTimestampChecker{
		threshold: ConfigTXTimestampThresholdDefault,
	}
}

func TransactionTimestampThreshold(wc state.WorldContext, g module.TransactionGroup) int64 {
	if g == module.TransactionGroupNormal {
		th := wc.TransactionTimestampThreshold()
		if th == 0 {
			th = ConfigTXTimestampThresholdDefault
		}
		return th
	} else {
		return ConfigPatchTimestampThreshold
	}
}

type TimestampRange interface {
	CheckTx(tx transaction.Transaction) error
}

type timestampRange struct {
	min, max int64
}

func (r *timestampRange) CheckTx(tx transaction.Transaction) error {
	return CheckTxTimestamp(r.min, r.max, tx)
}

func NewTimestampRange(bts int64, th int64) TimestampRange {
	return &timestampRange{
		min: bts - th,
		max: bts + th,
	}
}

type dummyTimestampRange struct{}

func (d dummyTimestampRange) CheckTx(tx transaction.Transaction) error {
	return nil
}

func NewDummyTimeStampRange() TimestampRange {
	return dummyTimestampRange{}
}

func NewTxTimestampRangeFor(c state.WorldContext, g module.TransactionGroup) TimestampRange {
	th := TransactionTimestampThreshold(c, g)
	bts := c.BlockTimeStamp()
	return &timestampRange{
		min: bts - th,
		max: bts + th,
	}
}
