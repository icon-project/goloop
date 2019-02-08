package service

import (
	"log"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/module"
)

type transactionReactor struct {
	membership module.ProtocolHandler
	normalPool *transactionPool
	patchPool  *transactionPool
}

const (
	reactorName                  = "transactionReactor"
	protocolPropagateTransaction = protocolInfo(0x1001)
)

var (
	sReactorCodec = codec.MP
	subProtocols  = []module.ProtocolInfo{protocolPropagateTransaction}
)

func (r *transactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case protocolPropagateTransaction:
		var tx transaction
		if _, err := sReactorCodec.UnmarshalFromBytes(buf, &tx); err != nil {
			log.Printf("Failed to unmarshal transaction. buf=%x, err=%+v\n", buf, err)
			return false, err
		}

		if err := tx.Verify(); err != nil {
			log.Printf("Failed to verify tx. err=%+v\n", err)
			return false, err
		}
		if tx.Group() == module.TransactionGroupPatch {
			if err := r.patchPool.add(&tx); err != nil {
				return false, err
			}
		} else {
			if err := r.normalPool.add(&tx); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (r *transactionReactor) propagateTransaction(pi module.ProtocolInfo, tx *transaction) error {
	buf, err := sReactorCodec.MarshalToBytes(tx)
	if err != nil {
		log.Printf("Failed to marshal transaction. tx=%v, err=%+v\n", tx, err)
	}

	if r != nil {
		r.membership.Multicast(protocolPropagateTransaction, buf, module.ROLE_VALIDATOR)
	}
	return nil
}

func (r *transactionReactor) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	// Nothing to do now.
}

func (r *transactionReactor) OnJoin(id module.PeerID) {
	// Nothing to do now.
}

func (r *transactionReactor) OnLeave(id module.PeerID) {
	// Nothing to do now.
}

func newTransactionReactor(nm module.NetworkManager, patch *transactionPool, normal *transactionPool) *transactionReactor {
	ra := &transactionReactor{
		patchPool:  patch,
		normalPool: normal,
	}
	ra.membership,_ = nm.RegisterReactor(reactorName, ra, subProtocols, 2)
	return ra
}
