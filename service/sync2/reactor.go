// Reactor for protocol v1

package sync2

import (
	"sync"

	"github.com/icon-project/goloop/common"
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

type ReactorCommon struct {
	mutex  sync.Mutex
	logger log.Logger
	ph     module.ProtocolHandler

	version   byte
	readyPool *peerPool
	watchers  []PeerWatcher
	sender    DataSender
}

func (r *ReactorCommon) OnJoin(id module.PeerID) {
	r.logger.Tracef("OnJoin() peer=%v, version=%d", id, r.version)
	locker := common.LockForAutoCall(&r.mutex)
	defer locker.Unlock()

	if r.readyPool.has(id) {
		return
	}
	p := newPeer(id, r.sender, r.logger)
	r.readyPool.push(p)

	watchers := r.watchers
	locker.CallAfterUnlock(func() {
		for _, watcher := range watchers {
			watcher.OnPeerJoin(p)
		}
	})
}

func (r *ReactorCommon) OnLeave(id module.PeerID) {
	r.logger.Tracef("OnLeave() peer=%v", id)
	locker := common.LockForAutoCall(&r.mutex)
	defer locker.Unlock()

	p := r.readyPool.remove(id)
	watchers := r.watchers

	locker.CallAfterUnlock(func() {
		for _, w := range watchers {
			w.OnPeerLeave(p)
		}
	})
}

func (r *ReactorCommon) GetVersion() byte {
	return r.version
}

func (r *ReactorCommon) WatchPeers(watcher PeerWatcher) []*peer {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.watchers = append(r.watchers, watcher)
	return r.readyPool.peerList()
}

func (r *ReactorCommon) UnwatchPeers(watcher PeerWatcher) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, w := range r.watchers {
		if w == watcher {
			if len(r.watchers) > 1 {
				watchers := make([]PeerWatcher, len(r.watchers)-1)
				copy(watchers, r.watchers[:i])
				copy(watchers[i:], r.watchers[i+1:])
				r.watchers = watchers
			} else {
				r.watchers = nil
			}
			return true
		}
	}
	return false
}

type ReactorV1 struct {
	ReactorCommon
	merkleTrie  db.Bucket
	bytesByHash db.Bucket
}

func (r *ReactorV1) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.logger.Tracef("OnReceive() pi=%d, peer=%v", pi, id)

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

func (r *ReactorV1) hasNode(msg []byte, id module.PeerID) *result {
	hr := new(hasNode)
	if _, err := c.UnmarshalFromBytes(msg, &hr); err != nil {
		r.logger.Tracef("Failed to unmarshal data len(msg)=%d", len(msg))
		return nil
	}

	status := NoError
	for _, hash := range [][]byte{hr.StateHash, hr.PatchHash, hr.NormalHash} {
		if len(hash) == 0 {
			continue
		}
		if v, err := r.merkleTrie.Get(hash); err != nil || v == nil {
			r.logger.Tracef("hasNode NoData v=%v, hash=%#x, err=%v", v, hash, err)
			status = ErrNoData
			break
		}
	}

	if hr.ValidatorHash != nil {
		if v, err := r.bytesByHash.Get(hr.ValidatorHash); err != nil || v == nil {
			r.logger.Tracef("hasNode NoData v=%v, hash=%#x, err=%v", v, hr.ValidatorHash, err)
			status = ErrNoData
		}
	}

	res := &result{hr.ReqID, status}
	r.logger.Tracef("responseResult=%v to peer=%v", res, id)

	return res
}

func (r *ReactorV1) _resolveNode(hashes [][]byte) (errCode, [][]byte) {
	r.logger.Tracef("_resolveNode() len(hashes)=%d", len(hashes))
	values := make([][]byte, 0, len(hashes))
	for _, hash := range hashes {
		var err error
		var v []byte
		for _, bucket := range []db.Bucket{r.merkleTrie, r.bytesByHash} {
			if v, err = bucket.Get(hash); err == nil && v != nil {
				values = append(values, v)
				break
			}
		}
	}
	r.logger.Debugf("_resolveNode() len(values)=%d", len(values))
	if len(values) == 0 {
		return ErrNoData, nil
	}
	return NoError, values
}

func (r *ReactorV1) requestNode(msg []byte, id module.PeerID) *nodeData {
	req := new(requestNodeData)
	if _, err := c.UnmarshalFromBytes(msg, &req); err != nil {
		r.logger.Info("Failed to unmarshal len(msg)=%d, error=%+v", len(msg), err)
		return nil
	}

	r.logger.Tracef("requestNode() request data reqID=%d, dataLen=%d", req.ReqID, len(req.Hashes))
	status, values := r._resolveNode(req.Hashes)
	r.logger.Tracef("requestNode() response data dataLen=%d, status=%d, peer=%s", len(values), status, id)
	res := &nodeData{req.ReqID, status, req.Type, values}

	return res
}

func (r *ReactorV1) onHasNode(msg []byte, id module.PeerID) {
	res := r.hasNode(msg, id)

	if b, err := c.MarshalToBytes(res); err != nil {
		r.logger.Warnf("Failed to marshal result error=%+v", err)
	} else if err = r.ph.Unicast(protoResult, b, id); err != nil {
		r.logger.Infof("Failed to send result error=%+v", err)
	}
}

func (r *ReactorV1) onRequestNodeData(msg []byte, id module.PeerID) {
	r.logger.Tracef("OnRequestNodeData() peer=%v", id)
	res := r.requestNode(msg, id)

	b, err := c.MarshalToBytes(res)
	if err != nil {
		r.logger.Warnf("Failed to marshal for nodeData=%v", res)
		return
	}
	r.logger.Tracef("responseNode ReqID=%d, Status=%d, Type=%d to peer=%v", res.ReqID, res.Status, res.Type, id)
	if err = r.ph.Unicast(protoNodeData, b, id); err != nil {
		r.logger.Info("Failed to send data peerID=%v", id)
	}
}

func (r *ReactorV1) processMsg(msg []byte, id module.PeerID) (*nodeData, error) {
	data := new(nodeData)
	_, err := c.UnmarshalFromBytes(msg, data)

	if err != nil {
		r.logger.Infof("Failed onReceive. receivedReqID=%d, err=%+v", data.ReqID, err)
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
	if d.Status == NoError {
		data = make([]BucketIDAndBytes, 0, len(d.Data))
		for _, b := range d.Data {
			data = append(data, BucketIDAndBytes{BkID: db.BytesByHash, Bytes: b})
		}
	}
	peer := r.readyPool.getPeer(id)
	if err := peer.OnData(d.ReqID, d.Status, data); err != nil {
		r.logger.Warnf("onResponseNodeData() notFound err=%v", err)
	}
}

func (r *ReactorV1) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.logger.Tracef("OnFailure() pi=%s, err=%+v", pi, err)
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

	r.logger.Tracef("RequestData() peer=%v", peer)
	return r.ph.Unicast(protoRequestNodeData, b, peer)
}

func newReactorV1(database db.Database, logger log.Logger) *ReactorV1 {
	merkleTrie, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		logger.Panicf("Failed to get bucket for MerkleTrie err=%+v", err)
	}

	bytesByHash, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		logger.Panicf("Failed to get bucket for BytesByHash err=%+v", err)
	}

	reactor := &ReactorV1{
		ReactorCommon: ReactorCommon{
			logger:    logger,
			version:   protoV1,
			readyPool: newPeerPool(),
		},
		merkleTrie:  merkleTrie,
		bytesByHash: bytesByHash,
	}
	reactor.sender = reactor
	return reactor
}
