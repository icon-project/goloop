package sync2

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
)

var (
	value1 = []byte("My Test value")
	value2 = []byte("My Sample value")
)

type mockReactor struct {
	logger log.Logger

	version   byte
	readyPool *peerPool
}

func (r *mockReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) {
	r.logger.Debugf("mockReactor(%v) OnReceive() peer=%v", r.version, id)

	go func() {
		d := new(responseData)
		_, err := c.UnmarshalFromBytes(b, d)
		if err != nil {
			r.logger.Errorf("invaild data error=%+v", err)
			return
		}

		p := r.readyPool.getPeer(id)
		p.OnData(d.ReqID, NoError, d.Data)
	}()
}

func (r *mockReactor) OnJoin(id module.PeerID) {
	r.logger.Debugf("mockReactor(%v) OnJoin() peer=%v", r.version, id)

	var dataSender DataSender = r
	peer := newPeer(id, dataSender, r.logger)
	r.readyPool.push(peer)
}

func (r *mockReactor) GetVersion() byte {
	return r.version
}

func (r *mockReactor) WatchPeers(w PeerWatcher) []*peer {
	return r.readyPool.peerList()
}

func (r *mockReactor) UnwatchPeers(watcher PeerWatcher) bool {
	/* do nothing */
	return true
}

func (r *mockReactor) RequestData(id module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	r.logger.Debugf("mockReactor(%v) RequestData() reqID=%d", r.version, reqID)

	peer := r.readyPool.getPeer(id)
	dummyPeer := peer
	var dummyData []BucketIDAndBytes

	r.logger.Debugf("mockReactor(%v) RequestData() reqData=%v", r.version, reqData)
	bkID := reqData[0].BkID
	key := string(reqData[0].Bytes)
	hasher := bkID.Hasher()

	switch key {
	case string(hasher.Hash(value1)):
		dummyData = []BucketIDAndBytes{
			{bkID, value1},
		}
	case string(hasher.Hash(value2)):
		dummyData = []BucketIDAndBytes{
			{bkID, value2},
		}
	}

	r.logger.Debugf("mockReactor(%v) request dummy data=%v", r.version, dummyData)
	// create dummy response
	resDummy := &responseData{
		ReqID:  reqID,
		Status: NoError,
		Data:   dummyData,
	}
	b, _ := c.MarshalToBytes(resDummy)

	r.logger.Debugf("mockReactor(%v) request dummy peer=%v", r.version, dummyPeer.id)

	r.OnReceive(protoV2Request, b, dummyPeer.id)

	return nil
}

type requestor struct {
	logger log.Logger
	id     db.BucketID
}

func (req *requestor) OnData(v []byte, builder merkle.Builder) error {
	req.logger.Debugf("requestor1 bucket id=%v, value=%#x", req.id, v)

	req2 := &requestor2{
		logger: req.logger,
		id:     req.id,
	}

	hasher := req.id.Hasher()

	key2 := hasher.Hash(value2)
	builder.RequestData(req.id, key2, req2)
	return nil
}

type requestor2 struct {
	logger log.Logger
	id     db.BucketID
}

func (req *requestor2) OnData(v []byte, builder merkle.Builder) error {
	req.logger.Debugf("requestor2 bucket id=%v, value=%#x", req.id, v)
	return nil
}

func TestSyncProcessorState(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	var builder merkle.Builder

	reactor1 := &mockReactor{
		logger:    log1,
		version:   protoV1,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		logger:    log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	reactors := []SyncReactor{reactor1, reactor2}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	req1 := &requestor{
		logger: log1,
		id:     db.BytesByHash,
	}

	// given test hashes on srcdb
	key1 := crypto.SHA3Sum256(value1)
	err := DBSet(srcdb, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	key2 := crypto.SHA3Sum256(value2)
	err = DBSet(srcdb, db.BytesByHash, key2, value2)
	assert.NoError(t, err)

	// when create syncProcessor
	builder = merkle.NewBuilder(dstdb)
	builder.RequestData(db.BytesByHash, key1, req1)
	sproc := newSyncProcessor(builder, reactors, log1, false)

	// then unresolved count is 1
	expected1 := 1
	actual1 := sproc.UnresolvedCount()
	assert.EqualValuesf(t, expected1, actual1, "UnresolveCount expected=%v, actual=%v", expected1, actual1)

	// when start sync
	err = sproc.DoSync()
	assert.NoError(t, err)
	builder.Flush(true)

	// then
	bk, _ := dstdb.GetBucket(db.BytesByHash)
	expected2 := value1
	actual2, _ := bk.Get(key1)
	assert.EqualValuesf(t, expected2, actual2, "Sync Result expected=%v, actual=%v", expected2, actual2)

	expected3 := value2
	actual3, _ := bk.Get(key2)
	assert.EqualValuesf(t, expected3, actual3, "Sync Result expected=%v, actual=%v", expected3, actual3)
}

const TestHasher db.BucketID = "T"

type testHasher struct {
}

func (h testHasher) Name() string {
	return "testhash"
}

func (h testHasher) Hash(v []byte) []byte {
	// use different hash algorithm
	// return crypto.SHA3Sum256(v)
	return crypto.SHASum256(v)
}

func init() {
	db.RegisterHasher(TestHasher, testHasher{})
}

func TestSyncProcessorBTPData(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	var builder merkle.Builder

	reactor1 := &mockReactor{
		logger:    log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		logger:    log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	syncer1 := &syncer{
		logger:   log1,
		database: srcdb,
		reactors: []SyncReactor{reactor1, reactor2},
		plt:      dummyExBuilder,
	}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	req1 := &requestor{
		logger: log1,
		id:     TestHasher,
	}

	// given test hashes on srcdb
	key1 := crypto.SHASum256(value1)
	err := DBSet(srcdb, TestHasher, key1, value1)
	assert.NoError(t, err)

	key2 := crypto.SHASum256(value2)
	err = DBSet(srcdb, TestHasher, key2, value2)
	assert.NoError(t, err)

	// when create syncProcessor
	builder = merkle.NewBuilder(dstdb)
	builder.RequestData(TestHasher, key1, req1)
	sproc := newSyncProcessor(builder, syncer1.reactors, syncer1.logger, false)

	// then unresolved count is 1
	expected1 := 1
	actual1 := sproc.builder.UnresolvedCount()
	assert.EqualValuesf(t, expected1, actual1, "UnresolveCount expected=%v, actual=%v", expected1, actual1)

	// when start sync
	err = sproc.DoSync()
	assert.NoError(t, err)
	builder.Flush(true)

	// then
	bk, _ := dstdb.GetBucket(TestHasher)
	expected2 := value1
	actual2, _ := bk.Get(key1)
	assert.EqualValuesf(t, expected2, actual2, "Sync Result expected=%v, actual=%v", expected2, actual2)

	expected3 := value2
	actual3, _ := bk.Get(key2)
	assert.EqualValuesf(t, expected3, actual3, "Sync Result expected=%v, actual=%v", expected3, actual3)
}

func TestSyncProcessorDataSyncer(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	reactor1 := &mockReactor{
		logger:    log1,
		version:   protoV1,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		logger:    log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	syncer1 := &syncer{
		logger:   log1,
		database: srcdb,
		reactors: []SyncReactor{reactor1, reactor2},
		plt:      dummyExBuilder,
	}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	// given
	var wg sync.WaitGroup

	// create data syncProcessor
	builder := merkle.NewBuilder(dstdb)
	sproc := newSyncProcessor(builder, syncer1.reactors, syncer1.logger, true)

	wg.Add(1)
	sproc.Start(func(err error) {
		expectedError := errors.ErrInterrupted.Error()
		assert.Errorf(t, err, expectedError, "sync finished expected=%v, actual=%v", expectedError, err)
		wg.Done()
	})

	// when add request to data syncer
	key1 := crypto.SHA3Sum256(value1)
	sproc.AddRequest(db.BytesByHash, key1)

	// waiting finish request data sync
	var try int
	for {
		if sproc.UnresolvedCount() == 0 {
			break
		} else if try >= 100 {
			t.Logf("data syncer failed. tried(%v)", try)
			break
		}
		time.Sleep(100 * time.Millisecond)
		try += 1
	}
	builder.Flush(true)

	// then
	bk, _ := dstdb.GetBucket(db.BytesByHash)
	expected1 := value1
	actual1, _ := bk.Get(key1)
	assert.EqualValuesf(t, expected1, actual1, "Sync Result expected=%v, actual=%v", expected1, actual1)

	// when stop data syncer
	time.AfterFunc(10*time.Millisecond, sproc.Stop)

	wg.Wait()
}

func TestSyncProcessorStartAsync(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	nm1 := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	log1 := logger.WithFields(log.Fields{log.FieldKeyWallet: nm1.id.String()[2:]})

	reactor1 := &mockReactor{
		logger:    log1,
		version:   protoV1,
		readyPool: newPeerPool(),
	}
	reactor2 := &mockReactor{
		logger:    log1,
		version:   protoV2,
		readyPool: newPeerPool(),
	}

	syncer1 := &syncer{
		logger:   log1,
		database: srcdb,
		reactors: []SyncReactor{reactor1, reactor2},
		plt:      dummyExBuilder,
	}

	reactor1.OnJoin(nm1.id)
	reactor2.OnJoin(nm2.id)

	req1 := &requestor{
		logger: log1,
		id:     db.BytesByHash,
	}

	// given
	key1 := crypto.SHA3Sum256(value1)
	err := DBSet(srcdb, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	var doneErr error

	doneCb := func(err error) {
		t.Logf("done called by %v", err)
		doneErr = err
		wg.Done()
	}

	// when async start finished by done
	builder := merkle.NewBuilder(dstdb)
	builder.RequestData(db.BytesByHash, key1, req1)
	sproc := newSyncProcessor(builder, syncer1.reactors, syncer1.logger, false)

	wg.Add(1)
	sproc.Start(doneCb)
	wg.Wait()

	// then done with nil
	assert.NoErrorf(t, doneErr, "done error expected=NoError, actual=%v", doneErr)

	// given
	dstdb2 := db.NewMapDB()

	// when async start finished by external stop call
	builder2 := merkle.NewBuilder(dstdb2)
	builder2.RequestData(db.BytesByHash, key1, req1)
	sproc2 := newSyncProcessor(builder2, syncer1.reactors, syncer1.logger, false)

	wg.Add(1)
	sproc2.Start(doneCb)

	sproc2.Stop()
	wg.Wait()

	// then done with ErrInterrupted
	expectedError := errors.ErrInterrupted
	assert.EqualErrorf(t, doneErr, expectedError.Error(), "done error expected=%v, actual=%v", errors.ErrInterrupted, doneErr)
}

type mockRequester struct {
}

func (r *mockRequester) OnData(v []byte, builder merkle.Builder) error {
	return nil
}

func TestSyncProcessorRequestData(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	srcdb := db.NewMapDB()
	dstdb := db.NewMapDB()
	builder := merkle.NewBuilder(dstdb)

	sp1 := &syncProcessor{
		logger:      logger,
		builder:     builder,
		reactors:    []SyncReactor{},
		readyPool:   newPeerPool(),
		sentPool:    newPeerPool(),
		checkedPool: newPeerPool(),
	}

	logger.Debugf("reqIter=%+v", sp1.reqIter)

	key1 := crypto.SHA3Sum256(value1)
	err := DBSet(srcdb, db.BytesByHash, key1, value1)
	assert.NoError(t, err)

	key2 := crypto.SHA3Sum256(value2)
	err = DBSet(srcdb, db.BytesByHash, key2, value2)
	assert.NoError(t, err)

	dataSender := &mockReactor{}

	getDataSize := func() int {
		dataSize := sp1.readyPool.size() * configPackSize
		logger.Debugf("dataSize=%v", dataSize)
		if dataSize > configRoundLimit {
			dataSize = configRoundLimit
		}
		return dataSize
	}

	addPeers := func(size int) {
		for i := 0; i < size; i++ {
			id := createAPeerID()
			p := newPeer(id, dataSender, logger)
			sp1.readyPool.push(p)
		}
	}

	removePeers := func(size int) {
		for i := 0; i < size; i++ {
			sp1.readyPool.pop()
		}
	}

	addRequestData := func(start, end uint32) {
		t.Logf("start : %v, end : %v", start, end)
		bs := make([]byte, 4)
		for i := start; i <= end; i++ {
			newValue := make([]byte, len(value1))
			copy(newValue, value1)

			binary.LittleEndian.PutUint32(bs, i)
			newValue = append(newValue, bs...)

			newKey := crypto.SHA3Sum256(newValue)
			DBSet(srcdb, db.BytesByHash, newKey, newValue)
			builder.RequestData(db.BytesByHash, newKey, &mockRequester{})
		}
	}

	onData := func(packs [][]BucketIDAndBytes) {
		t.Logf("packSize : %v", len(packs))
		for _, pack := range packs {
			for _, bnb := range pack {
				value, _ := DBGet(srcdb, db.BytesByHash, bnb.Bytes)
				builder.OnData(db.BytesByHash, value)
			}
		}
	}

	reqDataSize := func(packs [][]BucketIDAndBytes) int {
		var size int
		for _, pack := range packs {
			size += len(pack)
		}

		return size
	}

	// given peerSize = 10, packSize = configPackSize(50), roundLimit = configRoundLimit(500)
	logger.Debugf("configPackSize=%d, configRoundLimit=%d", configPackSize, configRoundLimit)
	peerSize := 10
	addPeers(peerSize)

	// when request data == 2, packs == 1, peerSize == 10
	peers := sp1.readyPool.size()
	logger.Debugf("readyPool=%d", peers)

	builder.RequestData(db.BytesByHash, key1, &mockRequester{})
	builder.RequestData(db.BytesByHash, key2, &mockRequester{})
	logger.Debugf("unresolve count=%v", builder.UnresolvedCount())

	packs := sp1.getPacks()

	// then request data == 2
	expected1 := 1
	actual1 := len(packs)
	logger.Debugf("pack size : expected=%v, actual=%v", expected1, actual1)
	assert.EqualValuesf(t, expected1, actual1, "pack size expected=%v, actual=%v", expected1, actual1)

	expected2 := 2
	actual2 := reqDataSize(packs)
	logger.Debugf("request data size : expected=%v, actual=%v", expected2, actual2)
	assert.EqualValuesf(t, expected2, actual2, "request data size expected=%v, actual=%v", expected2, actual2)

	// when received data 1 -> request data == 151, packs == 4, peerSize == 10
	requestDataSize := uint32(151 - builder.UnresolvedCount())
	logger.Debugf("value2=%#x", value2)
	addRequestData(1, requestDataSize)
	logger.Debugf("unresolve count=%v", builder.UnresolvedCount())

	packs = sp1.getPacks()

	// then request data == 151
	expected3 := int(math.Ceil(float64(151) / float64(configPackSize)))
	actual3 := len(packs)
	logger.Debugf("pack size : expected=%v, actual=%v", expected3, actual3)
	assert.EqualValuesf(t, expected3, actual3, "pack size expected=%v, actual=%v", expected3, actual3)

	expected4 := 151
	actual4 := reqDataSize(packs)
	logger.Debugf("request data size : expected=%v, actual=%v", expected4, actual4)
	assert.EqualValuesf(t, expected4, actual4, "request data size expected=%v, actual=%v", expected4, actual4)

	// when received data 1 -> request data == 515, packs == 11, peerSize == 5
	start := requestDataSize + 1
	end := uint32(515) - 2
	addRequestData(start, end)
	removePeers(6)
	logger.Debugf("unresolve count=%v, readyPool=%d", builder.UnresolvedCount(), sp1.readyPool.size())

	packs = sp1.getPacks()

	// then request data == 200, packs == 4, peerSize == 4
	dataSize := getDataSize()
	expected5 := int(math.Ceil(float64(dataSize) / float64(configPackSize)))
	actual5 := len(packs)
	logger.Debugf("pack size : expected=%v, actual=%v", expected5, actual5)
	assert.EqualValuesf(t, expected5, actual5, "pack size expected=%v, actual=%v", expected5, actual5)

	expected6 := dataSize
	actual6 := reqDataSize(packs)
	logger.Debugf("request data size : expected=%v, actual=%v", expected6, actual6)
	assert.EqualValuesf(t, expected6, actual6, "request data size expected=%v, actual=%v", expected6, actual6)

	onData(packs)

	// when request data == 315, packs == 10, peerSize == 5
	addPeers(1)
	logger.Debugf("unresolve count=%v, readyPool=%d", builder.UnresolvedCount(), sp1.readyPool.size())

	for sp1.reqCount > 0 {
		logger.Debugf("reqCount=%v", sp1.reqCount)
		remainCount := configRoundLimit - sp1.reqCount
		logger.Debugf("remainCount=%v", remainCount)

		packs = sp1.getPacks()

		if remainCount >= sp1.readyPool.size()*configPackSize { // readyPoolSize * configPackSize
			// then request data == 250, packs == 5
			dataSize := getDataSize()
			expected7 := int(math.Ceil(float64(dataSize) / float64(configPackSize)))
			actual7 := len(packs)
			logger.Debugf("pack size : expected=%v, actual=%v", expected7, actual7)
			assert.EqualValuesf(t, expected7, actual7, "pack size expected=%v, actual=%v", expected7, actual7)

			expected8 := dataSize
			actual8 := reqDataSize(packs)
			logger.Debugf("request data size : expected=%v, actual=%v", expected8, actual8)
			assert.EqualValuesf(t, expected8, actual8, "request data size expected=%v, actual=%v", expected8, actual8)
		} else {
			// then request data == 50, packs == 1
			dataSize := remainCount
			expected9 := int(math.Ceil(float64(dataSize) / float64(configPackSize)))
			actual9 := len(packs)
			logger.Debugf("pack size : expected=%v, actual=%v", expected9, actual9)
			assert.EqualValuesf(t, expected9, actual9, "pack size expected=%v, actual=%v", expected9, actual9)

			expected10 := remainCount
			actual10 := reqDataSize(packs)
			logger.Debugf("request data size : expected=%v, actual=%v", expected10, actual10)
			assert.EqualValuesf(t, expected10, actual10, "request data size expected=%v, actual=%v", expected10, actual10)
		}

		onData(packs)
	}

	// given request data == 50, peerSize == 1
	start = end + 1
	end = start + 34
	addRequestData(start, end)
	removePeers(4)

	logger.Debugf("unresolve count=%v, readyPool=%d", builder.UnresolvedCount(), sp1.readyPool.size())
	logger.Debugf("reqCount=%v", sp1.reqCount)

	// when request data == 50, packs == 1, peerSize == 1
	packs = sp1.getPacks()

	// then request data == 50, packs == 1
	expected11 := 1
	actual11 := len(packs)
	logger.Debugf("pack size : expected=%v, actual=%v", expected11, actual11)
	assert.EqualValuesf(t, expected11, actual11, "pack size expected=%v, actual=%v", expected11, actual11)

	expected12 := 50
	actual12 := reqDataSize(packs)
	logger.Debugf("request data size : expected=%v, actual=%v", expected12, actual12)
	assert.EqualValuesf(t, expected12, actual12, "request data size expected=%v, actual=%v", expected12, actual12)

	// given add 4 peers, builder reached end of requests
	addPeers(4)
	logger.Debugf("unresolve count=%v, readyPool=%d", builder.UnresolvedCount(), sp1.readyPool.size())

	// when request data == 50, packs == 1, peerSize == 5
	packs = sp1.getPacks()

	// then request data == 50, packs == 1, packs must not empty
	expected13 := 1
	actual13 := len(packs)
	logger.Debugf("pack size : expected=%v, actual=%v", expected11, actual11)
	assert.EqualValuesf(t, expected13, actual13, "pack size expected=%v, actual=%v", expected13, actual13)

	expected14 := 50
	actual14 := reqDataSize(packs)
	logger.Debugf("request data size : expected=%v, actual=%v", expected14, actual14)
	assert.EqualValuesf(t, expected12, actual12, "request data size expected=%v, actual=%v", expected14, actual14)
}
