package service

import (
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	ReactorName     = "transaction"
	ReactorPriority = 4
)

const (
	protoPropagateTransaction = module.ProtocolInfo(0x1001)
	protoRequestTransaction   = module.ProtocolInfo(0x1100)
	protoResponseTransaction  = module.ProtocolInfo(0x1200)
)

var (
	subProtocols = []module.ProtocolInfo{
		protoPropagateTransaction,
		protoRequestTransaction,
		protoResponseTransaction,
	}
)

type TransactionReactor struct {
	nm         module.NetworkManager
	membership module.ProtocolHandler
	tm         *TransactionManager
	log        log.Logger
	ts         *TransactionShare
}

func (r *TransactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case protoPropagateTransaction:
		tx, err := transaction.NewTransaction(buf)
		if err != nil {
			r.log.Warnf("InvalidPacket(PropagateTransaction) from=%s", peerId.String())
			r.log.Debugf("Failed to unmarshal transaction. buf=%x, err=%+v", buf, err)
			return false, err
		}

		if err := r.tm.Add(tx, false, false); err != nil {
			return false, err
		}
		return true, nil
	case protoResponseTransaction:
		tx, err := transaction.NewTransaction(buf)
		if err != nil {
			r.log.Warnf("InvalidPacket(ResponseTransaction) from=%s", peerId.String())
			r.log.Debugf("Failed to unmarshal transaction. buf=%x, err=%+v", buf, err)
			return false, err
		}

		if err := r.tm.Add(tx, false, false); err != nil {
			r.log.Debugf("Fail to add transaction id=%#x from=%s err=%+v",
				tx.ID(), peerId.String(), err)
			return false, err
		}
		if err := r.PropagateTransaction(tx); err != nil {
			if !network.NotAvailableError.Equals(err) {
				r.log.Debugf("Fail to propagate transaction err=%+v", err)
			}
		}
		return false, nil
	case protoRequestTransaction:
		return r.ts.HandleRequestTransaction(buf, peerId)
	}
	return false, nil
}

func (r *TransactionReactor) PropagateTransaction(tx transaction.Transaction) error {
	if r != nil && r.membership != nil {
		return r.membership.Multicast(protoPropagateTransaction, tx.Bytes(), module.ROLE_VALIDATOR)
	}
	return nil
}

func (r *TransactionReactor) OnJoin(id module.PeerID) {
	r.ts.HandleJoin(id)
}

func (r *TransactionReactor) OnLeave(id module.PeerID) {
	r.ts.HandleLeave(id)
}

func (r *TransactionReactor) Start(wallet module.Wallet) {
	r.membership, _ = r.nm.RegisterReactor(ReactorName, module.ProtoTransaction, r, subProtocols, ReactorPriority, module.NotRegisteredProtocolPolicyClose)
	r.ts.Start(r.membership, wallet)
	r.tm.SetPoolCapacityMonitor(r.ts)
}

func (r *TransactionReactor) Stop() {
	r.ts.Stop()
	_ = r.nm.UnregisterReactor(r)
}

func NewTransactionReactor(nm module.NetworkManager, tm *TransactionManager) *TransactionReactor {
	ra := &TransactionReactor{
		tm:  tm,
		nm:  nm,
		log: tm.Logger(),
		ts:  NewTransactionShare(tm),
	}
	return ra
}
