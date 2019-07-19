package service

import (
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type TxTimestampChecker struct {
	threshold int64
}

func (c *TxTimestampChecker) CheckWithCurrent(tx transaction.Transaction) error {
	return c.CheckWith(tx, time.Now().UnixNano()/1000)
}

func (c *TxTimestampChecker) CheckWith(tx transaction.Transaction, base int64) error {
	th := atomic.LoadInt64(&c.threshold)
	if th == 0 {
		th = transaction.ConfigTXTimestampThresholdDefault
	}
	diff := tx.Timestamp() - base
	if diff <= -th {
		log.Infof("Diff=%s Threshold=%s", TimestampToDuration(diff), TimestampToDuration(th))
		return ExpiredTransactionError.Errorf("ExpiredTx(diff=%s)", TimestampToDuration(diff))
	} else if diff > th {
		return InvalidTransactionError.Errorf("FutureTx(diff=%s)", TimestampToDuration(diff))
	}
	return nil
}

func (c *TxTimestampChecker) SetThreshold(d time.Duration) {
	log.Debugf("SetThreshold:%s", d)
	atomic.StoreInt64(&c.threshold, DurationToTimestamp(d))
}

func (c *TxTimestampChecker) Threshold() int64 {
	return atomic.LoadInt64(&c.threshold)
}

func TimestampToDuration(t int64) time.Duration {
	return time.Duration(t) * time.Microsecond
}

func DurationToTimestamp(d time.Duration) int64 {
	return int64(d / time.Microsecond)
}

func NewTimestampChecker() *TxTimestampChecker {
	return &TxTimestampChecker{
		threshold: transaction.ConfigTXTimestampThresholdDefault,
	}
}

func CheckTxTimestamp(c state.WorldContext, tx transaction.Transaction) error {
	th := c.TransactionTimestampThreshold()
	if th == 0 {
		th = transaction.ConfigTXTimestampThresholdDefault
	}
	diff := tx.Timestamp() - c.BlockTimeStamp()
	if diff <= -th {
		log.Infof("Diff=%s Threshold=%s", TimestampToDuration(diff), TimestampToDuration(th))
		return ExpiredTransactionError.Errorf("Expired(diff=%s)", time.Duration(diff*1000))
	} else if diff > th {
		return InvalidTransactionError.Errorf("FutureTx(diff=%s)", time.Duration(diff*1000))
	}
	return nil
}
