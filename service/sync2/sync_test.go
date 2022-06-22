package sync2

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/state"
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
				go r.reactor.OnReceive(pi, b, ph.nm.id)
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

func TestSync_AccountSync(t *testing.T) {
	logger := log.New()
	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})
	log2 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm2.id.String()[2:]})

	manager1 := NewSyncManager(srcdb, nm1, dummyExBuilder, log1)
	t.Log(manager1)

	manager2 := NewSyncManager(dstdb, nm2, dummyExBuilder, log2)
	nm1.join(nm2)

	// given
	ws := state.NewWorldState(srcdb, nil, nil, nil)
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

	// when start sync
	builders2 := manager2.GetSyncBuilders(acHash, nil, nil, nil, nil)

	for _, builder := range builders2 {
		t.Logf("builder unresolved : %d\n", builder.UnresolvedCount())
	}
	syncer := manager2.GetSyncer()
	result, err := syncer.SyncWithBuilders(builders2, nil)
	if err != nil {
		t.Errorf("Sync Processor failed : err(%v)", err)
	} else {
		t.Logf("wss : %x, prl : %x, nrl : %x\n", result.Wss.StateHash(), result.PatchReceipts.Hash(), result.NormalReceipts.Hash())
	}

	// start data syncer
	manager2.Start()

	err = manager2.AddRequest(db.BytesByHash, key1)
	assert.NoError(t, err)

	// then
	as := result.Wss.GetAccountSnapshot([]byte("ABC"))
	v, err := as.GetValue([]byte("ABC"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("XYZ"), v)

	time.Sleep(DataRequestRoundInterval + DataRequestNodeInterval)

	expected1 := value1
	actual1, err := DBGet(dstdb, db.BytesByHash, key1)
	assert.NoError(t, err)
	assert.Equal(t, expected1, actual1)

	manager2.Term()
}
