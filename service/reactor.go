package service

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

var (
	serviceReactorCodec = codec.MP
)

type serviceReactor struct {
	membership module.Membership
	txPool     *transactionPool
}

const (
	reactorName           = "serviceReactor"
	PROPAGATE_TRANSACTION = protocolInfo(0x1005)
)

var (
	subProtocols = []module.ProtocolInfo{PROPAGATE_TRANSACTION}
)

func (r *serviceReactor) OnReceive(subProtocol module.ProtocolInfo, buf []byte, peerId module.PeerID) (bool, error) {
	switch subProtocol {
	case PROPAGATE_TRANSACTION:
		var tx transaction
		//serviceReactorCodec.Unmarshal(bytes.NewBuffer(buf), &tx)
		//if tx.Verify() != nil {
		//	log.Errorf("Failed to unmarshal.")
		//	return false, nil
		//}
		// TODO below is temp for test. have to implement serialization with MP
		ntx, err := NewTransactionFromJSON(buf)
		log.Println("OnReceive err = ", err)
		if err != nil {
			return false, err
		}
		tx = *ntx.(*transaction)
		if result, err := r.txPool.add(&tx); result == false {
			log.Fatalf("Failed to add tx. tx = %v, err = %s\n", tx, err)
		}
		return true, nil
	}
	return false, nil
}

func (r *serviceReactor) propagateTransaction(pi module.ProtocolInfo, tx *transaction) error {
	// serialize transaction
	//buf := bytes.NewBuffer(nil)
	//if err := serviceReactorCodec.Marshal(buf, tx); err != nil {
	//	log.Errorf("Failed to marshal transaction. tx : %v, err : %s", tx, err)
	//	return err
	//}
	//r.membership.Broadcast(pi, buf.Bytes(), module.BROADCAST_ALL)
	// TODO: have to serialize with MP. below is temp code for test
	r.membership.Multicast(PROPAGATE_TRANSACTION, tx.Bytes(), module.ROLE_VALIDATOR)
	return nil
}

func (r *serviceReactor) OnError() {
}

type protocolInfo uint16

func (pi protocolInfo) ID() byte {
	return byte(pi >> 8)
}

func (pi protocolInfo) Version() byte {
	return byte(pi)
}

func (pi protocolInfo) Copy(b []byte) {
	binary.BigEndian.PutUint16(b[:2], uint16(pi))
}

func (pi protocolInfo) String() string {
	return fmt.Sprintf("{ID:SERVICE:%#02x,Ver:%#02x}", pi.ID(), pi.Version())
}

func (pi protocolInfo) Uint16() uint16 {
	return uint16(pi)
}
