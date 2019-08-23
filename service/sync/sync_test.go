package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/pkg/errors"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
	"github.com/icon-project/goloop/test"
)

type tReactorItem struct {
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

type tPacket struct {
	pi module.ProtocolInfo
	b  []byte
	id module.PeerID
}

type tNetworkManager struct {
	test.NetworkManagerBase
	id           module.PeerID
	reactorItems []*tReactorItem
	peers        []*tNetworkManager
	drop         bool
	recvBuf      []*tPacket
}

type tProtocolHandler struct {
	nm *tNetworkManager
	ri *tReactorItem
}

func newTNetworkManager(id module.PeerID) *tNetworkManager {
	return &tNetworkManager{id: id}
}

func (nm *tNetworkManager) GetPeers() []module.PeerID {
	res := make([]module.PeerID, len(nm.peers))
	for i := range nm.peers {
		res[i] = nm.peers[i].id
	}
	return res
}

func (nm *tNetworkManager) RegisterReactor(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *tNetworkManager) RegisterReactorForStreams(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
	//return registerReactorForStreams(nm, name, reactor, piList, priority, &common.GoTimeClock{})
}

func (nm *tNetworkManager) join(nm2 *tNetworkManager) {
	nm.peers = append(nm.peers, nm2)
	nm2.peers = append(nm2.peers, nm)
	for _, r := range nm.reactorItems {
		r.reactor.OnJoin(nm2.id)
	}
	for _, r := range nm2.reactorItems {
		r.reactor.OnJoin(nm.id)
	}
}

func (nm *tNetworkManager) onReceiveUnicast(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	nm.recvBuf = append(nm.recvBuf, &tPacket{pi, b, from})
}

func (nm *tNetworkManager) processRecvBuf() {
	for _, p := range nm.recvBuf {
		for _, r := range nm.reactorItems {
			r.reactor.OnReceive(p.pi, p.b, p.id)
		}
	}
	nm.recvBuf = nil
}

func (ph *tProtocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.id.Equal(id) {
			for _, r := range p.reactorItems {
				go r.reactor.OnReceive(pi, b, ph.nm.id)
			}
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func createAPeerID() module.PeerID {
	return network.NewPeerIDFromAddress(wallet.New().Address())
}

type tReceiveEvent struct {
	PI module.ProtocolInfo
	B  []byte
	ID module.PeerID
}

/*type tReceiveStreamMessageEvent struct {
	PI module.ProtocolInfo
	B  []byte
	ID module.PeerID
	SM streamMessage
}
*/
type tFailureEvent struct {
	Err error
	PI  module.ProtocolInfo
	B   []byte
}

type tJoinEvent struct {
	ID module.PeerID
}

type tLeaveEvent struct {
	ID module.PeerID
}

type tReactor struct {
	useStreamMessageEvent bool
	ch                    chan interface{}
}

//func newTReactor() *tReactor {
//	return &tReactor{ch: make(chan interface{}, 5)}
//}
//
//func (r *tReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
//	if r.useStreamMessageEvent {
//		sm := &streamMessage{}
//		codec.UnmarshalFromBytes(b, sm)
//		r.ch <- tReceiveStreamMessageEvent{pi, b, id, *sm}
//	} else {
//		r.ch <- tReceiveEvent{pi, b, id}
//	}
//	return false, nil
//}
//
//func (r *tReactor) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
//	r.ch <- tFailureEvent{err, pi, b}
//}
//
//func (r *tReactor) OnJoin(id module.PeerID) {
//	r.ch <- tJoinEvent{id}
//}
//
//func (r *tReactor) OnLeave(id module.PeerID) {
//	r.ch <- tLeaveEvent{id}
//}
//
//const (
//	pi0 protocolInfo = iota
//	pi1
//)
//
//var pis = []module.ProtocolInfo{pi0, pi1}
//
//type streamTestSetUp struct {
//	nm *tNetworkManager
//	r  *tReactor
//	ph module.ProtocolHandler
//
//	// for non stream
//	nm2 *tNetworkManager
//	r2  *tReactor
//	ph2 module.ProtocolHandler
//
//	clock    *common.TestClock
//	payloads [][]byte
//	tick     time.Duration
//}
//
//func newStreamTestSetUp(t *testing.T) *streamTestSetUp {
//	s := &streamTestSetUp{}
//	s.clock = &common.TestClock{}
//	s.nm = newTNetworkManager(createAPeerID())
//	s.nm2 = newTNetworkManager(createAPeerID())
//	s.nm.join(s.nm2)
//	s.r = newTReactor()
//	var err error
//	s.ph, err = s.nm.RegisterReactorForStreams("reactorA", s.r, pis, 1)
//	assert.Nil(t, err)
//	s.r2 = newTReactor()
//	s.ph2, err = s.nm2.RegisterReactor("reactorA", s.r2, pis, 1)
//	assert.Nil(t, err)
//	s.r2.useStreamMessageEvent = true
//
//	const NUM_PAYLOADS = 10
//	for i := 0; i < NUM_PAYLOADS; i++ {
//		s.payloads = append(s.payloads, []byte{byte(i + 1)})
//	}
//	s.tick = configPeerAckTimeout / 10
//	return s
//}

type testValidator struct {
	addr module.Address
}

func (tv *testValidator) Address() module.Address {
	return tv.addr
}

func (tv *testValidator) PublicKey() []byte {
	return tv.Address().Bytes()
}

func (tv *testValidator) Bytes() []byte {
	b, _ := c.MarshalToBytes(tv)
	return b
}

func TestSync_SimpleAccountSync(t *testing.T) {
	db1 := db.NewMapDB()
	db2 := db.NewMapDB()
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	_ = NewSyncManager(db1, nm, log.New())
	syncm2 := NewSyncManager(db2, nm2, log.New())

	nm.join(nm2)
	ws := state.NewWorldState(db1, nil, nil)
	ac := ws.GetAccountState([]byte("ABC"))
	ac.SetValue([]byte("ABC"), []byte("XYZ"))
	vs := ws.GetValidatorState()

	tvList := []module.Validator{
		&testValidator{addr: wallet.New().Address()},
		&testValidator{addr: wallet.New().Address()},
	}
	vs.Set(tvList)

	ac2 := ws.GetAccountState([]byte("XYZ"))
	ac2.SetValue([]byte("XYZ"), []byte("XYZ2"))

	acHash := ws.GetSnapshot().StateHash()
	log.Printf("acHash : %#x\n", acHash)
	ws.GetSnapshot().Flush()
	vh := ws.GetValidatorState().GetSnapshot().Hash()

	//r := syncm2.ForceSync(acHash, nil, nil, vlh)
	syncer1 := syncm2.NewSyncer(acHash, nil, nil, vh)
	r := syncer1.ForceSync()

	as := r.Wss.GetAccountSnapshot([]byte("ABC"))
	v, err := as.GetValue([]byte("ABC"))
	//for it := r.wss.Iterator(); it.Has(); it.Next() {
	//	o, k, _ := it.Get()
	//	log.Printf("iterator : o (%v), key(%#x)\n", o, k)
	//}
	//
	//v, err := r.Accounts.Get(crypto.SHA3Sum256([]byte("ABC")))
	if err != nil {
		t.Fatalf("err = %v\n", err)
	}

	log.Printf("v = %v\n", v)
}

func TestSync_AccountSync(t *testing.T) {
	var testItems [100]byte
	for i := range testItems {
		testItems[i] = byte(i)
	}

	const cPeers int = 16
	const cSyncPeers int = 3
	var databases [cPeers]db.Database
	var nms [cPeers]*tNetworkManager
	var syncM [cPeers]*Manager
	for i := 0; i < cPeers; i++ {
		databases[i] = db.NewMapDB()
		nms[i] = newTNetworkManager(createAPeerID())
		syncM[i] = NewSyncManager(databases[i], nms[i], log.New())
	}

	for i := 0; i < cPeers; i++ {
		for j := i; j < cPeers-1; j++ {
			nms[i].join(nms[j+1])
		}
	}

	var wss [cPeers]state.WorldState
	var prevHash []byte
	for i := 0; i < cPeers-cSyncPeers; i++ {
		wss[i] = state.NewWorldState(databases[i], nil, nil)
		for j := 0; j < 100; j++ {
			v := []byte{testItems[j]}
			ac := wss[i].GetAccountState(v)
			ac.SetValue(v, v)
		}
		ss := wss[i].GetSnapshot()
		ss.Flush()

		if i == 0 {
			prevHash = ss.StateHash()
		} else {
			if bytes.Compare(ss.StateHash(), prevHash) != 0 {
				t.Fatalf("Wrong hash\n")
			}
		}
	}

	finish := make(chan *Result)
	var results [cSyncPeers]*Result
	for i := 0; i < cSyncPeers; i++ {
		go func(index int) {
			r := syncM[cPeers-cSyncPeers+index].
				NewSyncer(prevHash, nil, nil, nil).
				ForceSync()
			finish <- r
		}(i)
	}
	finishCnt := 0

	for finishCnt != cSyncPeers {
		results[finishCnt] = <-finish
		finishCnt++
	}
	log.Printf("FINISH\n")
}

func TestSync_ReceiptsSync(t *testing.T) {
	db1 := db.NewMapDB()
	db2 := db.NewMapDB()

	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	_ = NewSyncManager(db1, nm, log.New())
	syncm2 := NewSyncManager(db2, nm2, log.New())

	nm.join(nm2)

	receiptsNum := 100
	patchReceipts := make([]txresult.Receipt, receiptsNum)
	normalReceipts := make([]txresult.Receipt, receiptsNum)

	for _, re := range [][]txresult.Receipt{patchReceipts, normalReceipts} {
		for i := 0; i < receiptsNum; i++ {
			addr := common.NewAddressFromString("cx0000000000000000000000000000000000000001")
			r := txresult.NewReceipt(addr)
			r.SetResult(module.StatusSuccess, big.NewInt(100*int64(i)), big.NewInt(1000), nil)
			r.SetCumulativeStepUsed(big.NewInt(100 * int64(i)))
			jso, err := r.ToJSON(jsonrpc.APIVersionLast)
			if err != nil {
				t.Errorf("Fail on ToJSON err=%+v", err)
			}
			jb, err := json.MarshalIndent(jso, "", "    ")

			fmt.Printf("JSON: %s\n", jb)

			r2, err := txresult.NewReceiptFromJSON(jb, jsonrpc.APIVersionLast)
			if err != nil {
				t.Errorf("Fail on Making Receipt from JSON err=%+v", err)
				return
			}
			re[i] = r2
		}
	}
	patchReceiptsList := txresult.NewReceiptListFromSlice(db1, patchReceipts)
	pHash := patchReceiptsList.Hash()
	patchReceiptsList.Flush()
	normalReceiptsList := txresult.NewReceiptListFromSlice(db1, normalReceipts)
	nHash := normalReceiptsList.Hash()
	normalReceiptsList.Flush()

	syncm2.NewSyncer(nil, pHash, nHash, nil).ForceSync()

	patchReceiptsList = txresult.NewReceiptListFromSlice(db2, patchReceipts)
	for it := patchReceiptsList.Iterator(); it.Has(); it.Next() {
		v, _ := it.Get()
		log.Printf("v = %v\n", v)
	}
}
