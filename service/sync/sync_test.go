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
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
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

func (nm *tNetworkManager) RegisterReactor(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *tNetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
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

type dummyExtensionBuilderType struct{}

func (d dummyExtensionBuilderType) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	return nil
}

var dummyExBuilder Platform = dummyExtensionBuilderType{}

func TestSync_SimpleAccountSync(t *testing.T) {
	db1 := db.NewMapDB()
	db2 := db.NewMapDB()
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	_ = NewSyncManager(db1, nm, dummyExBuilder, log.New())
	syncm2 := NewSyncManager(db2, nm2, dummyExBuilder, log.New())

	nm.join(nm2)
	ws := state.NewWorldState(db1, nil, nil, nil)
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

	syncer1 := syncm2.NewSyncer(acHash, nil, nil, vh, nil)
	r := syncer1.ForceSync()

	log.Printf("END\n")
	as := r.Wss.GetAccountSnapshot([]byte("ABC"))
	v, err := as.GetValue([]byte("ABC"))
	if err != nil {
		t.Fatalf("err = %v\n", err)
	}

	log.Printf("v = %v\n", v)
	log.Printf("END OF TestSync_SimpleAccountSync\n")
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
		syncM[i] = NewSyncManager(databases[i], nms[i], dummyExBuilder, log.New())
	}

	for i := 0; i < cPeers; i++ {
		for j := i; j < cPeers-1; j++ {
			nms[i].join(nms[j+1])
		}
	}

	var wss [cPeers]state.WorldState
	var prevHash []byte
	for i := 0; i < cPeers-cSyncPeers; i++ {
		wss[i] = state.NewWorldState(databases[i], nil, nil, nil)
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

	for i := 0; i < cSyncPeers; i++ {
		func(index int) {
			syncM[cPeers-cSyncPeers+index].
				NewSyncer(prevHash, nil, nil, nil, nil).
				ForceSync()
			log.Printf("Finish (%d)\n", index)
		}(i)
	}
	log.Printf("FINISH\n")
}

var receiptRevisions = []module.Revision{0, module.UseMPTOnEvents}

func TestSync_ReceiptsSync(t *testing.T) {
	for _, rev := range receiptRevisions {
		t.Run(fmt.Sprint("Revision:", rev), func(t *testing.T) {
			testReceiptSyncByRev(t, rev)
		})
	}
}

func testReceiptSyncByRev(t *testing.T, rev module.Revision) {
	db1 := db.NewMapDB()
	db2 := db.NewMapDB()

	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	_ = NewSyncManager(db1, nm, dummyExBuilder, log.New())
	syncm2 := NewSyncManager(db2, nm2, dummyExBuilder, log.New())

	nm.join(nm2)

	receiptsNum := 2
	patchReceipts := make([]txresult.Receipt, receiptsNum)
	normalReceipts := make([]txresult.Receipt, receiptsNum)

	for j, re := range [][]txresult.Receipt{patchReceipts, normalReceipts} {
		for i := 0; i < receiptsNum; i++ {
			addr := common.NewAddressFromString("cx0000000000000000000000000000000000000001")
			r := txresult.NewReceipt(db1, rev, addr)
			r.SetResult(module.StatusSuccess, big.NewInt(100*int64(i+j)), big.NewInt(1000), nil)
			r.SetCumulativeStepUsed(big.NewInt(100 * int64(i)))
			jso, err := r.ToJSON(module.JSONVersionLast)
			if err != nil {
				t.Errorf("Fail on ToJSON err=%+v", err)
			}
			jb, err := json.MarshalIndent(jso, "", "    ")

			//fmt.Printf("JSON: %s\n", jb)

			r2, err := txresult.NewReceiptFromJSON(db1, rev, jb)
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

	syncer := syncm2.NewSyncer(nil, pHash, nHash, nil, nil)
	syncer.ForceSync()
	syncer.Finalize()

	patchReceiptsListByHash := txresult.NewReceiptListFromHash(db2, pHash)
	log.Printf("pHash = %v, patchReceiptsListByHash = %v\n", pHash, patchReceiptsListByHash)

	i := 0
	for it := patchReceiptsListByHash.Iterator(); it.Has(); it.Next() {
		v, err := it.Get()
		if err != nil {
			log.Errorf("err = %s\n", err)
		}
		log.Printf("i = %d, p(%v)\n", i, patchReceipts[i].Bytes())
		log.Printf("v = %v\n", v)
		if bytes.Compare(patchReceipts[i].Bytes(), v.Bytes()) != 0 {
			t.Errorf("Diff pr %v, v %v\n", patchReceipts[i].Bytes(), v.Bytes())
		}
		i++
	}

	normalReceiptsListByHash := txresult.NewReceiptListFromHash(db2, nHash)
	i = 0
	for it := normalReceiptsListByHash.Iterator(); it.Has(); it.Next() {
		v, _ := it.Get()
		if bytes.Compare(normalReceipts[i].Bytes(), v.Bytes()) != 0 {
			t.Errorf("Diff pr %v, v %v\n", normalReceipts[i].Bytes(), v.Bytes())
		}
		i++
	}
}
