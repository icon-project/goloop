// Reactor for protocol v2

package sync2

import (
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ReactorV2 struct {
	mutex    sync.Mutex
	log      log.Logger
	database db.Database
	ph       module.ProtocolHandler

	version   byte
	server    *server
	readyPool *peerPool
}

func (r *ReactorV2) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.log.Debugf("OnReceive() pi(%d), peer id(%v)\n", pi, id)

	switch pi {
	case protoV2Request:
		go r.onRequest(b, id)
	case protoV2Response:
		go r.onResponse(b, id)
	}

	return false, nil
}

func (r *ReactorV2) onRequest(msg []byte, id module.PeerID) {
	res := r.server.requestV2(msg, id)

	b, err := c.MarshalToBytes(res)
	if err != nil {
		r.log.Warnf("Failed to marshal for responseData(%v)\n", res)
		return
	}
	r.log.Tracef("responseData ReqID(%d), Status(%d), peer(%s)\n", res.ReqID, res.Status, id)
	if err = r.ph.Unicast(protoV2Response, b, id); err != nil {
		r.log.Info("Failed to send data peerID(%s)\n", id)
	}
}

func (r *ReactorV2) processMsg(msg []byte, id module.PeerID) (*responseData, error) {
	r.log.Infof("processMsg() msg(%#x), peer id(%v)\n", msg, id)
	data := new(responseData)
	_, err := c.UnmarshalFromBytes(msg, data)

	if err != nil {
		r.log.Infof(
			"Failed onReceive. err(%v), receivedReqID(%d)\n", err, data.ReqID)
		return nil, errors.New("parse responseData failed")
	}
	return data, nil
}

func (r *ReactorV2) onResponse(msg []byte, id module.PeerID) {
	r.log.Infof("onResponse() peer id(%v)", id)
	d, err := r.processMsg(msg, id)
	if err != nil {
		return
	}

	peer := r.readyPool.getPeer(id)
	peer.OnData(d.ReqID, d.Data)
}

func (r *ReactorV2) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.log.Tracef("OnFailure() err(%+v), pi(%s)\n", err, pi)
}

// peer joined using protocol v2
func (r *ReactorV2) OnJoin(id module.PeerID) {
	r.log.Debugf("OnJoin() peer id(%v), version(%d)\n", id, r.version)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var dataSender DataSender = r
	peer := newPeer(id, dataSender, r.log)
	r.readyPool.push(peer)
}

// peer left using protocol v2
func (r *ReactorV2) OnLeave(id module.PeerID) {
	r.log.Debugf("OnLeave() peer id(%v)\n", id)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.readyPool.remove(id)
}

func (r *ReactorV2) ExistReadyPeer() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.readyPool.size() > 0
}

func (r *ReactorV2) GetVersion() byte {
	return r.version
}

func (r *ReactorV2) GetPeers() []*peer {
	return r.readyPool.peerList()
}

func (r *ReactorV2) RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	msg := &requestData{reqID, reqData}
	b, _ := c.MarshalToBytes(msg)

	return r.ph.Unicast(protoV2Request, b, peer)
}

func newReactorV2(database db.Database, logger log.Logger, version byte) *ReactorV2 {
	server := newServer(database, logger)

	reactor := &ReactorV2{
		log:       logger,
		database:  database,
		version:   version,
		server:    server,
		readyPool: newPeerPool(),
	}

	return reactor
}
