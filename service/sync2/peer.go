package sync2

import (
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type DataSender interface {
	RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error
}

type DataHandler func(sender *peer, data []BucketIDAndBytes)

type peer struct {
	log     log.Logger
	lock    sync.Mutex
	id      module.PeerID
	reqID   uint32
	expired time.Duration
	sender  DataSender
	reqMap  map[uint32]DataHandler
}

func newPeer(id module.PeerID, sender DataSender, logger log.Logger) *peer {
	return &peer{
		id:      id,
		sender:  sender,
		expired: configExpiredTime,
		log:     logger,
		reqMap:  make(map[uint32]DataHandler),
	}
}

func (p *peer) String() string {
	return fmt.Sprintf("peer id(%s), reqID(%d)", p.id, p.reqID)
}

func (p *peer) RequestData(reqData []BucketIDAndBytes, handler DataHandler) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.sender.RequestData(p.id, p.reqID, reqData); err == nil {
		p.reqMap[p.reqID] = handler
		p.reqID += 1
		return nil
	} else {
		return err
	}
}

func (p *peer) OnData(reqID uint32, data []BucketIDAndBytes) error {
	p.log.Debugf("OnData()")
	locker := common.LockForAutoCall(&p.lock)
	defer locker.Unlock()

	if handler, ok := p.reqMap[reqID]; ok {
		delete(p.reqMap, reqID)
		locker.CallAfterUnlock(func() {
			handler(p, data)
		})
		return nil
	} else {
		p.log.Debugf("OnData() notFound %v", reqID)
		return errors.NotFoundError.Errorf("UnknownRequestID(req=%d)", reqID)
	}
}
