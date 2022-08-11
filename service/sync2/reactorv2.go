// Reactor for protocol v2

package sync2

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ReactorV2 struct {
	ReactorCommon
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

func (r *ReactorV2) RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error {
	msg := &requestData{reqID, reqData}
	b, _ := c.MarshalToBytes(msg)

	return r.ph.Unicast(protoV2Request, b, peer)
}

func newReactorV2(s *server, logger log.Logger) *ReactorV2 {
	reactor := &ReactorV2{
		ReactorCommon: ReactorCommon{
			log:       logger,
			version:   protoV2,
			server:    s,
			readyPool: newPeerPool(),
		},
	}
	reactor.sender = reactor

	return reactor
}
