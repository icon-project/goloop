// Reactor for protocol v2

package sync2

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ReactorV2 struct {
	ReactorCommon
}

func (r *ReactorV2) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.logger.Tracef("OnReceive() pi(%d), peer id(%v)\n", pi, id)

	switch pi {
	case protoV2Request:
		go r.onRequest(b, id)
	case protoV2Response:
		go r.onResponse(b, id)
	}

	return false, nil
}

func (r *ReactorV2) _resolveData(bnbs []BucketIDAndBytes) (errCode, []BucketIDAndBytes) {
	resData := make([]BucketIDAndBytes, 0, len(bnbs))

	for _, bnb := range bnbs {
		var err error
		var v []byte
		var bucket db.Bucket

		switch bnb.BkID {
		case db.MerkleTrie:
			bucket = r.merkleTrie
		case db.BytesByHash:
			bucket = r.bytesByHash
		default:
			bucket = nil
			continue
		}

		if v, err = bucket.Get(bnb.Bytes); err == nil && v != nil {
			rbnb := BucketIDAndBytes{BkID: bnb.BkID, Bytes: v}
			resData = append(resData, rbnb)
		}
	}

	if len(resData) == 0 {
		return ErrNoData, nil
	}
	return NoError, resData
}

func (r *ReactorV2) request(msg []byte, id module.PeerID) *responseData {
	req := new(requestData)
	if _, err := c.UnmarshalFromBytes(msg, &req); err != nil {
		r.logger.Info("Failed to unmarshal error(%+v), (%#x)\n", err, msg)
		return nil
	}

	r.logger.Tracef("request() requestData : reqID(%d), dataLen(%d)\n", req.ReqID, len(req.Data))
	status, data := r._resolveData(req.Data)
	r.logger.Tracef("request() responseData : dataLen(%d), status(%d), peer(%v)\n", len(data), status, id)
	res := &responseData{req.ReqID, status, data}

	return res
}

func (r *ReactorV2) onRequest(msg []byte, id module.PeerID) {
	res := r.request(msg, id)

	b, err := c.MarshalToBytes(res)
	if err != nil {
		r.logger.Warnf("Failed to marshal for responseData(%v)\n", res)
		return
	}
	r.logger.Tracef("responseData ReqID(%d), Status(%d), peer(%s)\n", res.ReqID, res.Status, id)
	if err = r.ph.Unicast(protoV2Response, b, id); err != nil {
		r.logger.Infof("Failed to send data peerID(%s)\n", id)
	}
}

func (r *ReactorV2) processMsg(msg []byte, id module.PeerID) (*responseData, error) {
	r.logger.Tracef("processMsg() msg(%#x), peer id(%v)\n", msg, id)
	data := new(responseData)
	_, err := c.UnmarshalFromBytes(msg, data)

	if err != nil {
		r.logger.Infof(
			"Failed onReceive. err(%v), receivedReqID(%d)\n", err, data.ReqID)
		return nil, errors.New("parse responseData failed")
	}
	return data, nil
}

func (r *ReactorV2) onResponse(msg []byte, id module.PeerID) {
	r.logger.Tracef("onResponse() peer id(%v)", id)
	d, err := r.processMsg(msg, id)
	if err != nil {
		return
	}

	if d.Status != NoError {
		r.logger.Warnf("onResponse() peer id(%v), status(%v)", id, d.Status)
		return
	}

	peer := r.readyPool.getPeer(id)
	if err := peer.OnData(d.ReqID, d.Data); err != nil {
		r.logger.Warnf("onResponse() notFound err(%v)", err)
	}
}

func (r *ReactorV2) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.logger.Tracef("OnFailure() err(%+v), pi(%s)\n", err, pi)
}

func (r *ReactorV2) RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	r.logger.Tracef("requestData() peerid(%v), reqID(%d)\n", peer, reqID)
	msg := &requestData{reqID, reqData}
	b, _ := c.MarshalToBytes(msg)

	return r.ph.Unicast(protoV2Request, b, peer)
}

func newReactorV2(database db.Database, logger log.Logger) *ReactorV2 {
	merkleTrie, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicf("Failed to get bucket for MerkleTrie err(%s)\n", err)
	}

	bytesByHash, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		log.Panicf("Failed to get bucket for BytesByHash err(%s)\n", err)
	}

	reactor := &ReactorV2{
		ReactorCommon: ReactorCommon{
			logger:      logger,
			version:     protoV2,
			merkleTrie:  merkleTrie,
			bytesByHash: bytesByHash,
			readyPool:   newPeerPool(),
		},
	}
	reactor.sender = reactor

	return reactor
}
