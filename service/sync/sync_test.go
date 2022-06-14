package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
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
	module.NetworkManager
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

func (nm *tNetworkManager) RegisterReactor(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *tNetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
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

func DBSet(database db.Database, id db.BucketID, k, v []byte) error {
	bk, err := database.GetBucket(id)
	if err != nil {
		return err
	}
	return bk.Set(k, v)
}

func DBGet(database db.Database, id db.BucketID, k []byte) ([]byte, error) {
	bk, err := database.GetBucket(id)
	if err != nil {
		return nil, err
	}
	return bk.Get(k)
}

func TestSync_SimpleAccountSync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)
	db1 := db.NewMapDB()
	db2 := db.NewMapDB()
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm.id.String()[2:]})
	log2 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm2.id.String()[2:]})
	syncm1 := NewSyncManager(db1, nm, dummyExBuilder, log1)
	syncm2 := NewSyncManager(db2, nm2, dummyExBuilder, log2)
	syncm1.Start()
	syncm2.Start()

	nm.join(nm2)
	ws := state.NewWorldState(db1, nil, nil, nil, nil)
	ac := ws.GetAccountState([]byte("ABC"))
	ac.SetValue([]byte("ABC"), []byte("XYZ"))
	vs := ws.GetValidatorState()

	tvList := []module.Validator{
		&testValidator{addr: wallet.New().Address()},
		&testValidator{addr: wallet.New().Address()},
	}
	vs.Set(tvList)

	value1 := []byte("My Test Is")
	key1 := crypto.SHA3Sum256(value1)
	err := DBSet(db1, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	err = syncm2.AddRequest(db.BytesByHash, key1)
	assert.NoError(t, err)

	ac2 := ws.GetAccountState([]byte("XYZ"))
	ac2.SetValue([]byte("XYZ"), []byte("XYZ2"))

	acHash := ws.GetSnapshot().StateHash()
	logger.Printf("acHash : %#x\n", acHash)
	ws.GetSnapshot().Flush()
	vh := ws.GetValidatorState().GetSnapshot().Hash()

	syncer1 := syncm2.NewSyncer(acHash, nil, nil, vh, nil)
	r, _ := syncer1.ForceSync()

	logger.Printf("END\n")
	as := r.Wss.GetAccountSnapshot([]byte("ABC"))
	v, err := as.GetValue([]byte("ABC"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("XYZ"), v)

	time.Sleep(DataRequestRoundInterval + DataRequestNodeInterval)

	value2, err := DBGet(db2, db.BytesByHash, key1)
	assert.NoError(t, err)
	assert.Equal(t, value1, value2)
}

func TestSync_DataSync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	const cPeers int = 16
	var databases [cPeers]db.Database
	var nms [cPeers]*tNetworkManager
	var syncM [cPeers]*Manager
	for i := 0; i < cPeers; i++ {
		databases[i] = db.NewMapDB()
		nms[i] = newTNetworkManager(createAPeerID())
		syncM[i] = NewSyncManager(databases[i], nms[i], dummyExBuilder, logger)
		syncM[i].Start()
	}

	for i := 0; i < cPeers; i++ {
		for j := i; j < cPeers-1; j++ {
			nms[i].join(nms[j+1])
		}
	}

	var keys [][]byte
	var values [][]byte
	for i := 0; i < cPeers; i++ {
		value := []byte(fmt.Sprintf("TEST Data %d", i))
		key := crypto.SHA3Sum256(value)
		err := DBSet(databases[i], db.BytesByHash, key, value)
		assert.NoError(t, err)
		keys = append(keys, key)
		values = append(values, value)
	}

	for i := 0; i < cPeers; i++ {
		for j := 0; j < cPeers; j++ {
			err := syncM[i].AddRequest(db.BytesByHash, keys[j])
			assert.NoError(t, err)
		}
	}

	var checkedServers int
	for try := 0; try < 6 && checkedServers < cPeers; try++ {
		checkedServers = 0
		for i := 0; i < cPeers; i++ {
			checkedEntries := 0
			for j := 0; j < cPeers; j++ {
				value, err := DBGet(databases[i], db.BytesByHash, keys[j])
				assert.NoError(t, err)
				if bytes.Equal(value, values[j]) {
					checkedEntries += 1
				}
			}
			if checkedEntries == cPeers {
				checkedServers += 1
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
	assert.Equal(t, cPeers, checkedServers)
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
		syncM[i].Start()
	}

	for i := 0; i < cPeers; i++ {
		for j := i; j < cPeers-1; j++ {
			nms[i].join(nms[j+1])
		}
	}

	var wss [cPeers]state.WorldState
	var prevHash []byte
	for i := 0; i < cPeers-cSyncPeers; i++ {
		wss[i] = state.NewWorldState(databases[i], nil, nil, nil, nil)
		for j := 0; j < 100; j++ {
			v := []byte{testItems[j]}
			ac := wss[i].GetAccountState(v)
			ac.SetValue(v, v)
			k := crypto.SHA3Sum256(v)
			err := DBSet(databases[i], db.BytesByHash, k, v)
			assert.NoError(t, err)
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
			addr := common.MustNewAddressFromString("cx0000000000000000000000000000000000000001")
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
