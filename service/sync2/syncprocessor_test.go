package sync2

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

var (
	value1 = []byte("My Test Is")
	value2 = []byte("My Test Is2")
)

type mockReactor struct {
	log log.Logger

	version   byte
	readyPool *peerPool
}

func (r *mockReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) {
	r.log.Debugf("mockReactor(%v) OnReceive() peer id(%v)\n", r.version, id)

	d := new(responseData)
	_, err := c.UnmarshalFromBytes(b, d)
	if err != nil {
		r.log.Errorf("invaild data %v", err)
		return
	}

	p := r.readyPool.getPeer(id)
	p.OnData(d.ReqID, d.Data)
}

func (r *mockReactor) OnJoin(id module.PeerID) {
	r.log.Debugf("mockReactor(%v) OnJoin() peer id(%v)\n", r.version, id)

	var dataSender DataSender = r
	peer := newPeer(id, dataSender, r.log)
	r.readyPool.push(peer)
}

func (r *mockReactor) ExistReadyPeer() bool {
	return r.readyPool.size() > 0
}

func (r *mockReactor) GetVersion() byte {
	return r.version
}

func (r *mockReactor) WatchPeers(w PeerWatcher) []*peer {
	return r.readyPool.peerList()
}

func (r *mockReactor) RequestData(id module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	r.log.Debugf("mockReactor(%v) RequestData() reqID(%d)", r.version, reqID)

	peer := r.readyPool.getPeer(id)
	dummyPeer := peer
	var dummyData []BucketIDAndBytes

	r.log.Debugf("mockReactor(%v) RequestData() reqData(%v)", r.version, reqData)
	key := string(reqData[0].Bytes)

	switch key {
	case string(crypto.SHA3Sum256(value1)):
		dummyData = []BucketIDAndBytes{
			{db.BytesByHash, value1},
		}
	case string(crypto.SHA3Sum256(value2)):
		dummyData = []BucketIDAndBytes{
			{db.BytesByHash, value2},
		}
	}

	r.log.Debugf("mockReactor(%v) request dummy data(%v)\n", r.version, dummyData)
	// create dummy response
	resDummy := &responseData{
		ReqID:  reqID,
		Status: NoError,
		Data:   dummyData,
	}
	b, _ := c.MarshalToBytes(resDummy)

	r.log.Debugf("mockReactor(%v) request dummy peer id(%v)\n", r.version, dummyPeer.id)

	f := func() {
		r.OnReceive(protoV2Request, b, dummyPeer.id)
	}
	go time.AfterFunc(100*time.Millisecond, f)

	return nil
}

type requestor struct {
	log log.Logger
	id  db.BucketID
}

func (req *requestor) OnData(v []byte, builder merkle.Builder) error {
	req.log.Debugf("requestor1 bucket id : %v, value : %x\n", req.id, v)

	req2 := &requestor2{
		log: req.log,
		id:  db.BytesByHash,
	}

	key2 := crypto.SHA3Sum256(value2)
	builder.RequestData(db.BytesByHash, key2, req2)
	return nil
}

type requestor2 struct {
	log log.Logger
	id  db.BucketID
}

func (req *requestor2) OnData(v []byte, builder merkle.Builder) error {
	req.log.Debugf("requestor2 bucket id : %v, value : %x\n", req.id, v)
	return nil
}

func TestSyncProcessorStateBasic(t *testing.T) {
	logger := log.New()
	srcdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	syncer := &syncer{
		log:      log1,
		database: srcdb,
		reactors: []SyncReactor{},
		plt:      dummyExBuilder,
	}

	// given no data

	// when create syncProcessor
	builder := merkle.NewBuilder(srcdb)
	sp := newSyncProcessor(builder, syncer.reactors, syncer.log, false)

	// then unresolve count is 0
	syncp := sp.(*syncProcessor)
	expected1 := 0
	actual1 := syncp.builder.UnresolvedCount()
	assert.EqualValuesf(t, expected1, actual1, "UnresolveCount expected : %v, actual : %v", expected1, actual1)

	// when start sync
	sp.StartSync()

	// then last expect state is done
	expected2 := DoneState
	actual2 := syncp.state
	assert.EqualValuesf(t, expected2, actual2, "Last state expected : %v, acutal : %v", expected2, actual2)
}

func TestSyncProcessorStateAdvance(t *testing.T) {
	logger := log.New()
	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	var builder merkle.Builder

	reactor1 := &mockReactor{
		log:       log1,
		version:   protoV1,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		log:       log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	syncer1 := &syncer{
		log:      log1,
		database: srcdb,
		reactors: []SyncReactor{reactor1, reactor2},
		plt:      dummyExBuilder,
	}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	req1 := &requestor{
		log: log1,
		id:  db.BytesByHash,
	}

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

	key1 := crypto.SHA3Sum256(value1)
	err := DBSet(srcdb, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	key2 := crypto.SHA3Sum256(value2)
	err = DBSet(srcdb, db.BytesByHash, key2, value2)
	assert.NoError(t, err)

	// when create syncProcessor
	builder = merkle.NewBuilder(dstdb)
	builder.RequestData(db.BytesByHash, key1, req1)
	sp1 := newSyncProcessor(builder, syncer1.reactors, syncer1.log, false)

	// then unresolved count is 1
	syncp := sp1.(*syncProcessor)
	expected1 := 1
	actual1 := syncp.builder.UnresolvedCount()
	assert.EqualValuesf(t, expected1, actual1, "UnresolveCount expected : %v, actual : %v", expected1, actual1)

	// when start sync
	err = sp1.StartSync()
	assert.NoError(t, err)
	builder.Flush(true)

	// then
	bk, _ := dstdb.GetBucket(db.BytesByHash)
	expected2 := value1
	actual2, _ := bk.Get(key1)
	assert.EqualValuesf(t, expected2, actual2, "Sync Result expected : %v, actual : %v", expected2, actual2)

	expected3 := value2
	actual3, _ := bk.Get(key2)
	assert.EqualValuesf(t, expected3, actual3, "Sync Result expected : %v, actual : %v", expected3, actual3)
}

func TestSyncProcessorDataSyncer(t *testing.T) {
	logger := log.New()
	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	var builder merkle.Builder

	reactor1 := &mockReactor{
		log:       log1,
		version:   protoV1,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		log:       log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	syncer1 := &syncer{
		log:      log1,
		database: srcdb,
		reactors: []SyncReactor{reactor1, reactor2},
		plt:      dummyExBuilder,
	}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	// given
	var wg sync.WaitGroup

	// create data syncProcessor
	builder = merkle.NewBuilder(dstdb)
	sp1 := newSyncProcessor(builder, syncer1.reactors, syncer1.log, true)

	wg.Add(1)
	go func() {
		err := sp1.StartSync()
		if err == errors.ErrInterrupted {
			wg.Done()
		}
	}()

	log1.Debugf("sleep 1 seconds..")
	time.Sleep(time.Second)

	// when add request to data syncer
	key1 := crypto.SHA3Sum256(value1)
	sp1.AddRequest(db.BytesByHash, key1)

	// waiting finish request data sync
	log1.Debugf("sleep 1 seconds..")
	time.Sleep(1 * time.Second)
	builder.Flush(true)

	// then
	bk, _ := dstdb.GetBucket(db.BytesByHash)
	expected1 := value1
	actual1, _ := bk.Get(key1)
	assert.EqualValuesf(t, expected1, actual1, "Sync Result expected : %v, actual : %v", expected1, actual1)

	// when stop data syncer
	log1.Debugf("sleep 1 seconds..")
	time.AfterFunc(time.Second, sp1.Stop)

	wg.Wait()
}

func ExampleSyncProcessor() {
}
