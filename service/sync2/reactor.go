// Reactor for protocol v1

package sync2

import (
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

// syncType for protocol v1
type syncType int

const (
	syncWorldState syncType = 1 << iota
	syncPatchReceipts
	syncNormalReceipts
	syncExtensionState
	syncTypeReserved
)

type ReactorV1 struct {
	mutex    sync.Mutex
	log      log.Logger
	database db.Database
	ph       module.ProtocolHandler

	version   byte
	server    *server
	readyPool *peerPool
}

func (r *ReactorV1) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.log.Debugf("OnReceive() pi(%d), peer id(%v)\n", pi, id)

	switch pi {
	case protoHasNode:
		go r.onHasNode(b, id)
	case protoResult:
		// do nothing
	case protoRequestNodeData:
		go r.onRequestNodeData(b, id)
	case protoNodeData:
		go r.onResponseNodeData(b, id)
	}
	return false, nil
}

func (r *ReactorV1) onHasNode(msg []byte, id module.PeerID) {
	res := r.server.hasNode(msg, id)

	if b, err := c.MarshalToBytes(res); err != nil {
		r.log.Warnf("Failed to marshal result error(%+v)\n", err)
	} else if err = r.ph.Unicast(protoResult, b, id); err != nil {
		r.log.Infof("Failed to send result error(%+v)\n", err)
	}
}

func (r *ReactorV1) onRequestNodeData(msg []byte, id module.PeerID) {
	r.log.Debugf("OnRequestNodeData() peer id(%v)\n", id)
	res := r.server.requestNode(msg, id)

	b, err := c.MarshalToBytes(res)
	if err != nil {
		r.log.Warnf("Failed to marshal for nodeData(%v)\n", res)
		return
	}
	r.log.Tracef("responseNode ReqID(%d), Status(%d), Type(%d) to peer(%s)\n", res.ReqID, res.Status, res.Type, id)
	if err = r.ph.Unicast(protoNodeData, b, id); err != nil {
		r.log.Info("Failed to send data peerID(%s)\n", id)
	}
}

func (r *ReactorV1) processMsg(msg []byte, id module.PeerID) (*nodeData, error) {
	data := new(nodeData)
	_, err := c.UnmarshalFromBytes(msg, data)

	if err != nil {
		r.log.Infof(
			"Failed onReceive. err(%v), receivedReqID(%d)\n", err, data.ReqID)
		return nil, errors.New("parse nodeData failed")
	}

	return data, nil
}

func (r *ReactorV1) onResponseNodeData(msg []byte, id module.PeerID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	d, err := r.processMsg(msg, id)
	if err != nil {
		return
	}
	var data []BucketIDAndBytes
	for _, b := range d.Data {
		data = append(data, BucketIDAndBytes{BkID: db.BytesByHash, Bytes: b})
	}
	peer := r.readyPool.getPeer(id)
	peer.OnData(d.ReqID, data)
}

func (r *ReactorV1) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.log.Tracef("OnFailure() err(%+v), pi(%s)\n", err, pi)
}

// peer joined using protocol v1
func (r *ReactorV1) OnJoin(id module.PeerID) {
	r.log.Tracef("OnJoin() peer id(%v), version(%d)\n", id, r.version)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var dataSender DataSender = r
	peer := newPeer(id, dataSender, r.log)
	r.readyPool.push(peer)
}

// peer left using protocol v1
func (r *ReactorV1) OnLeave(id module.PeerID) {
	r.log.Tracef("OnLeave() peer id(%v)\n", id)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.readyPool.remove(id)
}

func (r *ReactorV1) ExistReadyPeer() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.readyPool.size() > 0
}

func (r *ReactorV1) GetVersion() byte {
	return r.version
}

func (r *ReactorV1) GetPeers() []*peer {
	return r.readyPool.peerList()
}

func (r *ReactorV1) RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	var keys [][]byte
	for _, data := range reqData {
		if data.BkID != db.BytesByHash && data.BkID != db.MerkleTrie {
			return errors.IllegalArgumentError.Errorf("InvalidBucketID(bkid=%s)", data.BkID)
		}
		keys = append(keys, data.Bytes)
	}
	msg := &requestNodeData{reqID, syncWorldState, keys}
	b, _ := c.MarshalToBytes(msg)

	r.log.Debugf("RequestData() peer id(%v)", peer)
	return r.ph.Unicast(protoRequestNodeData, b, peer)
}

func newReactorV1(database db.Database, logger log.Logger, version byte) *ReactorV1 {

	server := newServer(database, logger)

	reactor := &ReactorV1{
		log:       logger,
		database:  database,
		version:   version,
		server:    server,
		readyPool: newPeerPool(),
	}

	return reactor
}
