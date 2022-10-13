package sync2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
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
	pi       module.ProtocolInfo
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
	joinReactors []*tReactorItem
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
		pi:       pi,
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
		pi:       pi,
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

	var nmPiVer, nm2PiVer, piVer byte

	for _, r := range nm.reactorItems {
		ver := r.pi.Version()
		if nmPiVer < ver {
			nmPiVer = ver
		}
	}

	for _, r := range nm2.reactorItems {
		ver := r.pi.Version()
		if nm2PiVer < ver {
			nm2PiVer = ver
		}
	}

	if nmPiVer == nm2PiVer {
		piVer = nmPiVer
	} else {
		piVer = func(x, y byte) byte {
			if x < y {
				return x
			}
			return y
		}(nmPiVer, nm2PiVer)
	}

	for _, r := range nm.reactorItems {
		if piVer == r.pi.Version() {
			nm.joinReactors = append(nm.joinReactors, r)
			r.reactor.OnJoin(nm2.id)
		}
	}
	for _, r := range nm2.reactorItems {
		if piVer == r.pi.Version() {
			nm2.joinReactors = append(nm2.joinReactors, r)
			r.reactor.OnJoin(nm.id)
		}
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
			for _, r := range p.joinReactors {
				runtime.Gosched()
				r.reactor.OnReceive(pi, b, ph.nm.id)
			}
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func (ph *tProtocolHandler) GetPeers() []module.PeerID {
	return ph.nm.GetPeers()
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

func TestSyncSimpleAccountSync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})
	log2 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm2.id.String()[2:]})

	NewSyncManager(srcdb, nm1, dummyExBuilder, log1)

	manager2 := NewSyncManager(dstdb, nm2, dummyExBuilder, log2)
	nm1.join(nm2)

	// given
	ws := state.NewWorldState(srcdb, nil, nil, nil, nil)
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
	err := DBSet(srcdb, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	acHash := ws.GetSnapshot().StateHash()
	ws.GetSnapshot().Flush()
	logger.Printf("account hash : (%x)\n", acHash)

	syncer2 := manager2.NewSyncer(acHash, nil, nil, nil, nil, nil, true)
	result, err := syncer2.ForceSync()
	assert.NoError(t, err)

	// then
	as := result.Wss.GetAccountSnapshot([]byte("ABC"))
	v, err := as.GetValue([]byte("ABC"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("XYZ"), v)

	// when start data syncer
	manager2.Start()

	err = manager2.AddRequest(db.BytesByHash, key1)
	assert.NoError(t, err)

	var try int
	for {
		if manager2.UnresolvedRequestCount() == 0 {
			break
		} else if try >= 10 {
			t.Logf("datasyncer sync failed. tried(%v)", try)
			break
		}
		time.Sleep(100 * time.Millisecond)
		try += 1
	}

	// then
	expected1 := value1
	actual1, err := DBGet(dstdb, db.BytesByHash, key1)
	assert.NoError(t, err)
	assert.Equal(t, expected1, actual1)

	manager2.Term()
}

func TestSyncDataSync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	// given 16 peers
	const cPeers int = 16
	var databases [cPeers]db.Database
	var nms [cPeers]*tNetworkManager
	var syncM [cPeers]*Manager
	var slog [cPeers]log.Logger
	for i := 0; i < cPeers; i++ {
		databases[i] = db.NewMapDB()
		nms[i] = newTNetworkManager(createAPeerID())
		slog[i] = logger.WithFields(log.Fields{log.FieldKeyWallet: nms[i].id.String()[2:]})
		syncM[i] = NewSyncManager(databases[i], nms[i], dummyExBuilder, slog[i])
		syncM[i].Start()
	}

	for i := 0; i < cPeers; i++ {
		for j := i; j < cPeers-1; j++ {
			nms[i].join(nms[j+1])
		}
	}

	// peers have different data
	var keys [][]byte
	var values [][]byte
	var reqSize int = 16
	for i := 0; i < cPeers; i++ {
		value := []byte(fmt.Sprintf("TEST Data %d", i))
		key := crypto.SHA3Sum256(value)
		err := DBSet(databases[i], db.BytesByHash, key, value)
		assert.NoError(t, err)
		keys = append(keys, key)
		values = append(values, value)
	}

	// when addRequest all data to peers
	rkeys := append([][]byte(nil), keys...)

	for i := 0; i < cPeers; i++ {
		rand.Shuffle(len(rkeys), func(i, j int) { rkeys[i], rkeys[j] = rkeys[j], rkeys[i] })
		for j := 0; j < reqSize; j++ {
			err := syncM[i].AddRequest(db.BytesByHash, rkeys[j])
			assert.NoError(t, err)
		}
	}

	var wg sync.WaitGroup
	waitFinish := func(mgr *Manager, idx int) {
		var try int = 0
		for {
			if mgr.UnresolvedRequestCount() == 0 {
				break
			} else if try >= 50 {
				t.Logf("syncM[%d] sync failed. try count(%d)", idx, try)
				break
			}

			time.Sleep(100 * time.Millisecond)
			try += 1
		}
		wg.Done()
	}

	for i := 0; i < cPeers; i++ {
		wg.Add(1)
		go waitFinish(syncM[i], i)
	}
	wg.Wait()

	// then all data synced
	var checkedServers int = 0
	for i := 0; i < cPeers; i++ {
		syncM[i].Term()
		checkedEntries := 0
		for j := 0; j < reqSize; j++ {
			value, err := DBGet(databases[i], db.BytesByHash, keys[j])
			assert.NoError(t, err)
			if bytes.Equal(value, values[j]) {
				checkedEntries += 1
			}
		}
		if checkedEntries == reqSize {
			checkedServers += 1
		}
	}
	assert.Equal(t, cPeers, checkedServers)
}

func TestSyncAccountSync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	var testItems [1000]byte
	for i := range testItems {
		testItems[i] = byte(i)
	}

	const cPeers int = 16
	const cSyncPeers int = 3
	var databases [cPeers]db.Database
	var nms [cPeers]*tNetworkManager
	var syncM [cPeers]*Manager
	var slog [cPeers]log.Logger
	for i := 0; i < cPeers; i++ {
		databases[i] = db.NewMapDB()
		nms[i] = newTNetworkManager(createAPeerID())
		slog[i] = logger.WithFields(log.Fields{log.FieldKeyWallet: nms[i].id.String()[2:]})
		syncM[i] = NewSyncManager(databases[i], nms[i], dummyExBuilder, slog[i])
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
		for j := 0; j < len(testItems); j++ {
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
			if !bytes.Equal(ss.StateHash(), prevHash) {
				t.Fatalf("Wrong hash\n")
			}
		}
	}

	for i := 0; i < cSyncPeers; i++ {
		testName := "ForceSync_" + strconv.Itoa(i)
		t.Run(testName, func(t *testing.T) {
			syncM[cPeers-cSyncPeers+i].
				NewSyncer(prevHash, nil, nil, nil, nil, nil, false).
				ForceSync()
			t.Logf("Finish (%d)\n", i)
		})
		time.Sleep(time.Millisecond)
	}

	t.Logf("FINISH\n")
}

var receiptRevisions = []module.Revision{0, module.UseMPTOnEvents}

func TestSyncReceiptsSync(t *testing.T) {
	for _, rev := range receiptRevisions {
		t.Run(fmt.Sprint("Revision:", rev), func(t *testing.T) {
			testReceiptSyncByRev(t, rev)
		})
	}
}

func testReceiptSyncByRev(t *testing.T, rev module.Revision) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	db1 := db.NewMapDB()
	db2 := db.NewMapDB()
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm.id.String()[2:]})
	log2 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm2.id.String()[2:]})
	NewSyncManager(db1, nm, dummyExBuilder, log1)
	syncm2 := NewSyncManager(db2, nm2, dummyExBuilder, log2)

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
			jb, _ := json.MarshalIndent(jso, "", "    ")

			//t.Logf("JSON: %s\n", jb)

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

	syncer := syncm2.NewSyncer(nil, pHash, nHash, nil, nil, nil, false)
	syncer.ForceSync()
	syncer.Finalize()

	patchReceiptsListByHash := txresult.NewReceiptListFromHash(db2, pHash)
	t.Logf("pHash = %v, patchReceiptsListByHash = %v\n", pHash, patchReceiptsListByHash)

	i := 0
	for it := patchReceiptsListByHash.Iterator(); it.Has(); it.Next() {
		v, err := it.Get()
		if err != nil {
			log.Errorf("err = %s\n", err)
		}
		t.Logf("i = %d, p(%v)\n", i, patchReceipts[i].Bytes())
		t.Logf("v = %v\n", v)
		if !bytes.Equal(patchReceipts[i].Bytes(), v.Bytes()) {
			t.Errorf("Diff pr %v, v %v\n", patchReceipts[i].Bytes(), v.Bytes())
		}
		i++
	}

	normalReceiptsListByHash := txresult.NewReceiptListFromHash(db2, nHash)
	i = 0
	for it := normalReceiptsListByHash.Iterator(); it.Has(); it.Next() {
		v, _ := it.Get()
		if !bytes.Equal(normalReceipts[i].Bytes(), v.Bytes()) {
			t.Errorf("Diff pr %v, v %v\n", normalReceipts[i].Bytes(), v.Bytes())
		}
		i++
	}
}
