package service

import (
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

type TransactionReactor struct {
	nm         module.NetworkManager
	membership module.ProtocolHandler
	tm         *TransactionManager
	tsc        *TxTimestampChecker
}

const (
	ReactorName                  = "transaction"
	ProtocolPropagateTransaction = module.ProtocolInfo(0x1001)
	ReactorPriority              = 4
)

var (
	subProtocols = []module.ProtocolInfo{ProtocolPropagateTransaction}
)

func (r *TransactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case ProtocolPropagateTransaction:
		tx, err := transaction.NewTransaction(buf)
		if err != nil {
			log.Tracef("Failed to unmarshal transaction. buf=%x, err=%+v\n", buf, err)
			return false, err
		}

		if err := r.tm.Add(tx, false); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *TransactionReactor) PropagateTransaction(pi module.ProtocolInfo, tx transaction.Transaction) error {
	if r != nil && r.membership != nil {
		return r.membership.Multicast(ProtocolPropagateTransaction, tx.Bytes(), module.ROLE_VALIDATOR)
	}
	return nil
}

func (r *TransactionReactor) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	// Nothing to do now.
}

func (r *TransactionReactor) OnJoin(id module.PeerID) {
	// Nothing to do now.
}

func (r *TransactionReactor) OnLeave(id module.PeerID) {
	// Nothing to do now.
}

func (r *TransactionReactor) Start() {
	r.membership, _ = r.nm.RegisterReactor(ReactorName, module.ProtoTransaction, r, subProtocols, ReactorPriority)
}

func (r *TransactionReactor) Stop() {
	_ = r.nm.UnregisterReactor(r)
}

func NewTransactionReactor(nm module.NetworkManager, tm *TransactionManager) *TransactionReactor {
	ra := &TransactionReactor{
		tm: tm,
		nm: nm,
	}
	return ra
}
