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

type DataHandler func(reqID uint32, sender *peer, data []BucketIDAndBytes)

type peer struct {
	logger  log.Logger
	lock    sync.Mutex
	id      module.PeerID
	reqID   uint32
	timer   *time.Timer
	expired time.Duration
	sender  DataSender
	reqMap  map[uint32]DataHandler
}

func newPeer(id module.PeerID, sender DataSender, logger log.Logger) *peer {
	return &peer{
		id:      id,
		sender:  sender,
		expired: configExpiredTime,
		logger:  logger,
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
		p.logger.Tracef("RequestData() peer id(%v), reqID(%v)", p.id, p.reqID)
		p.reqMap[p.reqID] = handler
		p.reqID += 1
		p.timer = time.AfterFunc(p.expired*time.Millisecond, func() {
			delete(p.reqMap, p.reqID)
			handler(p, nil)
		})
		return nil
	} else {
		return err
	}
}

func (p *peer) OnData(reqID uint32, data []BucketIDAndBytes) error {
	p.logger.Tracef("OnData() peer id(%v), reqID(%v)", p.id, reqID)
	locker := common.LockForAutoCall(&p.lock)
	defer locker.Unlock()

	if handler, ok := p.reqMap[reqID]; ok {
		p.timer.Stop()
		delete(p.reqMap, reqID)
		locker.CallAfterUnlock(func() {
			handler(reqID, p, data)
		})
		return nil
	} else {
		return errors.NotFoundError.Errorf("UnknownRequestID(req=%d)", reqID)
	}
}

func (p *peer) Reset() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.reqID = 0
	p.reqMap = make(map[uint32]DataHandler)
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
}
