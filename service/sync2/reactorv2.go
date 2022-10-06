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
	database db.Database
}

func (r *ReactorV2) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.logger.Tracef("OnReceive() pi=%d, peerid=%v", pi, id)

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

		if hash := bnb.BkID.Hasher(); hash == nil {
			r.logger.Warnf("INVALID bucket id=%s (no hasher)", bnb.BkID)
			continue
		}
		bucket, err = r.database.GetBucket(bnb.BkID)
		if err != nil {
			r.logger.Errorf("FAIL to get bucket id=%s", bnb.BkID)
			continue
		}
		if v, err = bucket.Get(bnb.Bytes); err == nil && v != nil {
			r.logger.Tracef("RESOLVED id=%s key=%#x value=%#x", bnb.BkID, bnb.Bytes, v)
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
		r.logger.Infof("Failed to unmarshal error=%+v, msg=%#x", err, msg)
		return nil
	}

	r.logger.Tracef("request() requestData reqID=%d, dataLen=%d", req.ReqID, len(req.Data))
	status, data := r._resolveData(req.Data)
	r.logger.Tracef("request() responseData dataLen=%d, status=%d, peer=%v", len(data), status, id)
	res := &responseData{req.ReqID, status, data}

	return res
}

func (r *ReactorV2) onRequest(msg []byte, id module.PeerID) {
	res := r.request(msg, id)

	b, err := c.MarshalToBytes(res)
	if err != nil {
		r.logger.Warnf("Failed to marshal for responseData=%v", res)
		return
	}
	r.logger.Tracef("onRequest() responseData ReqID=%d, Status=%d, peer=%v", res.ReqID, res.Status, id)
	if err = r.ph.Unicast(protoV2Response, b, id); err != nil {
		r.logger.Infof("onRequest() Failed to send data peer=%v", id)
	}
}

func (r *ReactorV2) processMsg(msg []byte, id module.PeerID) (*responseData, error) {
	r.logger.Tracef("processMsg() msg=%#x, peerid=%v", msg, id)
	data := new(responseData)
	_, err := c.UnmarshalFromBytes(msg, data)

	if err != nil {
		r.logger.Infof("Failed onReceive. ReqID=%d, err=%v", data.ReqID, err)
		return nil, errors.New("parse responseData failed")
	}
	return data, nil
}

func (r *ReactorV2) onResponse(msg []byte, id module.PeerID) {
	r.logger.Tracef("onResponse() peer=%v", id)
	d, err := r.processMsg(msg, id)
	if err != nil {
		return
	}

	peer := r.readyPool.getPeer(id)
	if err := peer.OnData(d.ReqID, d.Status, d.Data); err != nil {
		r.logger.Warnf("onResponse() notFound err=%v", err)
	}
}

func (r *ReactorV2) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.logger.Tracef("OnFailure() pi=%s, err=%+v", pi, err)
}

func (r *ReactorV2) RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	r.logger.Tracef("requestData() peer=%v, reqID=%d", peer, reqID)
	msg := &requestData{reqID, reqData}
	b, _ := c.MarshalToBytes(msg)

	return r.ph.Unicast(protoV2Request, b, peer)
}

func newReactorV2(database db.Database, logger log.Logger) *ReactorV2 {
	reactor := &ReactorV2{
		ReactorCommon: ReactorCommon{
			logger:    logger,
			version:   protoV2,
			readyPool: newPeerPool(),
		},
		database: database,
	}
	reactor.sender = reactor

	return reactor
}
