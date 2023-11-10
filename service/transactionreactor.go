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
	membership module.AsyncProtocolHandler
	tm         *TransactionManager
	log        log.Logger
	ts         *TransactionShare
}

func (r *TransactionReactor) handleTransactionInBackground(buf []byte, peerId module.PeerID, propagate bool) (bool, error){
	onResult, err := r.membership.HandleInBackground()
	if err != network.ErrInProgress {
		return false, err
	}
	go func() {
		tx, err := transaction.NewTransaction(buf)
		if err != nil {
			r.log.Warnf("InvalidPacket from=%s", peerId.String())
			r.log.Debugf("Failed to unmarshal transaction. buf=%x, err=%+v", buf, err)
			onResult(false, err)
			return
		}

		if err := r.tm.Add(tx, false, false); err != nil {
			onResult(false, err)
			return
		}
		if propagate {
			if err := r.PropagateTransaction(tx); err != nil {
				if !network.NotAvailableError.Equals(err) {
					r.log.Debugf("Fail to propagate transaction err=%+v", err)
				}
			}
		}
		onResult(true, nil)
		return
	}()
	return false, network.ErrInProgress
}

func (r *TransactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case protoPropagateTransaction:
		return r.handleTransactionInBackground(buf, peerId, false)
	case protoResponseTransaction:
		return r.handleTransactionInBackground(buf, peerId, true)
	case protoRequestTransaction:
		return r.ts.HandleRequestTransaction(buf, peerId)
	}
	return false, nil
}

func (r *TransactionReactor) PropagateTransaction(tx transaction.Transaction) error {
	if r != nil && r.membership != nil {
		return r.membership.Multicast(protoPropagateTransaction, tx.Bytes(), module.RoleValidator)
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
	ph, _ := r.nm.RegisterReactor(ReactorName, module.ProtoTransaction, r, subProtocols, ReactorPriority, module.NotRegisteredProtocolPolicyClose)
	r.membership = ph.(module.AsyncProtocolHandler)
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
