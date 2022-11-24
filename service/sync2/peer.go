package sync2

import (
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type DataSender interface {
	RequestData(peer module.PeerID, reqID uint32, reqData []BucketIDAndBytes) error
}

type DataHandler func(reqID uint32, sender *peer, data []BucketIDAndBytes)

type peerRequest struct {
	timer   *time.Timer
	handler DataHandler
}

type peer struct {
	logger  log.Logger
	lock    sync.Mutex
	id      module.PeerID
	reqID   uint32
	expired time.Duration
	sender  DataSender
	reqMap  map[uint32]peerRequest
}

func newPeer(id module.PeerID, sender DataSender, logger log.Logger) *peer {
	return &peer{
		id:      id,
		sender:  sender,
		expired: configExpiredTime,
		logger:  logger,
		reqMap:  make(map[uint32]peerRequest),
	}
}

func (p *peer) String() string {
	return fmt.Sprintf("{id=%v}", p.id)
}

func (p *peer) getExpired() time.Duration {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.expired
}

func (p *peer) RequestData(reqData []BucketIDAndBytes, handler DataHandler) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	reqID := p.reqID
	p.logger.Tracef("RequestData() peer=%v, reqID=%v, reqData=%d", p.id, reqID, len(reqData))
	if err := p.sender.RequestData(p.id, reqID, reqData); err == nil {
		p.reqID += 1
		p.reqMap[reqID] = peerRequest{
			handler: handler,
			timer: time.AfterFunc(p.expired, func() {
				_ = p.OnData(reqID, ErrTimeExpired, nil)
			}),
		}
		return nil
	} else {
		return err
	}
}

func (p *peer) OnData(reqID uint32, status errCode, data []BucketIDAndBytes) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.logger.Tracef("OnData() peer=%v expired=%v reqID=%d status=%s data=%d", p.id, p.expired, reqID, status, len(data))
	if status == ErrTimeExpired && p.expired < configMaxExpiredTime {
		p.expired += 200 * time.Millisecond
	}

	if request, ok := p.reqMap[reqID]; ok {
		delete(p.reqMap, reqID)
		request.timer.Stop()
		go request.handler(reqID, p, data)
		return nil
	} else {
		p.logger.Debugf("OnData() peer=%v, reqID=%v: unknown request", p.id, reqID)
		return errors.NotFoundError.Errorf("UnknownRequestID(req=%d)", reqID)
	}
}
