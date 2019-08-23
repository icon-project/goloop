package sync

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type client struct {
	ph    module.ProtocolHandler
	mutex sync.Mutex
	log   log.Logger
}

func (cl *client) hasNode(p *peer, wsHash, prHash, nrHash, vh []byte) error {
	reqID := p.reqID + 1
	msg := &hasNode{reqID, wsHash, vh, prHash, nrHash}
	cl.log.Debugf("hasNode msg(%s)\n", msg)
	b, _ := c.MarshalToBytes(msg)
	if err := cl.ph.Unicast(protoHasNode, b, p.id); err != nil {
		cl.log.Infof("Failed to request for protoHasNode err(%+v)\n", err)
		return err
	}
	p.reqID = reqID
	p.timer = time.AfterFunc(time.Millisecond*configExpiredTime, func() {
		cl.log.Debug("hasNode time expired")
		p.onReceive(receiveCancled, protoResult, b)
	})
	return nil
}

func (cl *client) requestNodeData(p *peer, hash [][]byte, t syncType,
	cb func(msgType int, pi module.ProtocolInfo, b []byte, p *peer)) error {
	reqID := p.reqID + 1
	msg := &requestNodeData{reqID, t, hash}
	cl.log.Debugf("requestNodeData msg(%s)\n", msg)
	b, _ := c.MarshalToBytes(msg)
	if err := cl.ph.Unicast(protoRequestNodeData, b, p.id); err != nil {
		cl.log.Infof("Failed to request for protoRequestNodeData err(%+v)\n", err)
		return err
	}

	p.reqID = reqID
	p.timer = time.AfterFunc(time.Millisecond*configExpiredTime, func() {
		cl.log.Debug("requestNodeData time expired")
		cb(receiveTimeExpired, protoNodeData, nil, p)
	})
	return nil
}

func newClient(ph module.ProtocolHandler, log log.Logger) *client {
	log.Debug("newClient")
	cl := new(client)
	cl.ph = ph
	cl.log = log
	return cl
}
