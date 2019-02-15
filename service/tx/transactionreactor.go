package tx

import (
	"log"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/module"
)

type TransactionReactor struct {
	membership module.ProtocolHandler
	normalPool *TransactionPool
	patchPool  *TransactionPool
}

const (
	ReactorName                  = "TransactionReactor"
	ProtocolPropagateTransaction = protocolInfo(0x1001)
)

var (
	sReactorCodec = codec.MP
	subProtocols  = []module.ProtocolInfo{ProtocolPropagateTransaction}
)

func (r *TransactionReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case ProtocolPropagateTransaction:
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
			if err := r.patchPool.Add(&tx); err != nil {
				return false, err
			}
		} else {
			if err := r.normalPool.Add(&tx); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (r *TransactionReactor) PropagateTransaction(pi module.ProtocolInfo, tx Transaction) error {
	buf, err := sReactorCodec.MarshalToBytes(tx)
	if err != nil {
		log.Printf("Failed to marshal transaction. tx=%v, err=%+v\n", tx, err)
	}

	if r != nil {
		r.membership.Multicast(ProtocolPropagateTransaction, buf, module.ROLE_VALIDATOR)
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

func NewTransactionReactor(nm module.NetworkManager, patch *TransactionPool, normal *TransactionPool) *TransactionReactor {
	ra := &TransactionReactor{
		patchPool:  patch,
		normalPool: normal,
	}
	ra.membership, _ = nm.RegisterReactor(ReactorName, ra, subProtocols, 2)
	return ra
}
