package service

import (
	"log"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

type TransactionReactor struct {
	nm         module.NetworkManager
	membership module.ProtocolHandler
	normalPool *TransactionPool
	patchPool  *TransactionPool
}

const (
	ReactorName                  = "TransactionReactor"
	ProtocolPropagateTransaction = protocolInfo(0x1001)
)

var (
	subProtocols = []module.ProtocolInfo{ProtocolPropagateTransaction}
)

func (r *TransactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case ProtocolPropagateTransaction:
		tx, err := transaction.NewTransaction(buf)
		if err != nil {
			log.Printf("Failed to unmarshal transaction. buf=%x, err=%+v\n", buf, err)
			return false, err
		}

		if err := tx.Verify(); err != nil {
			log.Printf("Failed to verify tx. err=%+v\n", err)
			return false, err
		}
		if tx.Group() == module.TransactionGroupPatch {
			if err := r.patchPool.Add(tx); err != nil {
				return false, err
			}
		} else {
			if err := r.normalPool.Add(tx); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (r *TransactionReactor) PropagateTransaction(pi module.ProtocolInfo, tx transaction.Transaction) error {
	if r != nil && r.membership != nil {
		r.membership.Multicast(ProtocolPropagateTransaction, tx.Bytes(), module.ROLE_VALIDATOR)
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
	r.membership, _ = r.nm.RegisterReactor(ReactorName, r, subProtocols, 2)
}

func (r *TransactionReactor) Stop() {
	_ = r.nm.UnregisterReactor(r)
}

func NewTransactionReactor(nm module.NetworkManager, patch *TransactionPool, normal *TransactionPool) *TransactionReactor {
	ra := &TransactionReactor{
		patchPool:  patch,
		normalPool: normal,
		nm:         nm,
	}
	return ra
}
