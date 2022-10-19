/*
 * Copyright 2022 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sync

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
)

const (
	DataRequestEntryLimit    = 20
	DataRequestNodeLimit     = 3
	DataRequestNodeInterval  = time.Millisecond * 300
	DataRequestRoundInterval = time.Second * 3
)

type dataSyncer struct {
	lock sync.Mutex

	waiter *sync.Cond
	timer  *time.Timer

	ready   *peerPool
	sent    *peerPool
	checked *peerPool

	client *client
	bd     merkle.Builder
	log    log.Logger
}

func (s *dataSyncer) notifyInLock() {
	s.waiter.Signal()
}

func (s *dataSyncer) onJoin(p *peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.ready.has(p.id) || s.sent.has(p.id) || s.checked.has(p.id) {
		return
	}
	p.cb = s
	s.ready.push(p)
	s.notifyInLock()
}

func (s *dataSyncer) onLeave(id module.PeerID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if p := s.ready.remove(id); p != nil {
		return
	}
	if p := s.checked.remove(id); p != nil {
		return
	}
	if p := s.sent.remove(id); p != nil {
		p.timer.Stop()
	}
}

func (s *dataSyncer) Activate(pool *peerPool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.log.Debug("DataSyncer: Activate")

	s.ready.clear()
	peers := pool.peerList()
	for _, p := range peers {
		p.cb = s
		s.ready.push(p)
	}
	s.cancelTimerInLock()
	s.notifyInLock()
}

func (s *dataSyncer) Deactivate() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.log.Debug("DataSyncer: Deactivate")

	s.deactivateInLock()
}

func (s *dataSyncer) deactivateInLock() {
	s.ready.clear()
	s.checked.clear()
	list := s.sent.peerList()
	for _, p := range list {
		p.timer.Stop()
	}
	s.sent.clear()
	s.cancelTimerInLock()
}

func (s *dataSyncer) onResult(status errCode, p *peer) {
	// nothing to do
}

func (s *dataSyncer) onNodeData(p *peer, status errCode, t syncType, data [][]byte) {
	if status == NoError {
		s.log.Debugf("DataSyncer: onNodeData count=%d from=%s", len(data), p)
		for _, value := range data {
			if err := s.bd.OnData(db.BytesByHash, value); err != nil {
				s.log.Warnf("DataSyncer: FAIL on delivery data err=%+v", err)
			}
		}
	}
}

func (s *dataSyncer) onReceive(pi module.ProtocolInfo, b []byte, p *peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	reqID, msg, err := parseMessage(pi, b)
	if err != nil {
		s.log.Warnf("DataSyncer: FAIL to parse message err=%+v", err)
		return
	}
	if !p.IsValidRequest(reqID) {
		return
	}
	p2 := s.sent.remove(p.id)
	if p2 == nil {
		return
	}
	p2.timer.Stop()
	if !p2.onReceive(pi, msg) {
		s.log.Errorf("DataSyncer: INVALID protocol=%d", pi)
	}

	if s.bd.UnresolvedCount() == 0 {
		s.ready.push(p2)
		s.moveCheckedToReadyInLock()
	} else {
		s.checked.push(p2)
		if s.sent.size() == 0 && s.ready.size() == 0 {
			s.ready, s.checked = s.checked, s.ready
			s.notifyInLock()
		}
	}
}

func (s *dataSyncer) serve() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for {
		if s.bd == nil {
			break
		}
		if s.timer != nil {
			s.waiter.Wait()
			continue
		}
		if s.bd.UnresolvedCount() == 0 || s.ready.size() == 0 {
			s.waiter.Wait()
			continue
		}

		s.log.Tracef("DataSyncer: Serve unresolved=%d ready=%d", s.bd.UnresolvedCount(), s.ready.size())

		itr := s.bd.Requests()
		idx := 0
		keys := [][]byte(nil)
		for itr.Next() && idx < DataRequestEntryLimit {
			bkID := itr.BucketIDs()[0]
			switch bkID {
			case db.BytesByHash, db.MerkleTrie:
				keys = append(keys, itr.Key())
				idx += 1
			default:
				s.log.Errorf("DataSyncer: Unknown Bucket bk=%q", bkID)
			}
		}
		if len(keys) > 0 {
			peers := s.ready.peerList()
			if len(peers) > DataRequestNodeLimit {
				peers = peers[0:DataRequestNodeLimit]
			}
			for _, p := range peers {
				s.log.Debugf("DataSyncer: requestNodeData to=%s count=%d", p, len(keys))
				err := s.client.requestNodeData(p, keys, syncWorldState, s.onReceive)
				if err != nil {
					s.log.Errorf("DataSyncer: FAIL to send node request err=%+v", err)
				} else {
					s.ready.remove(p.id)
					s.sent.push(p)
				}
			}
			if s.ready.size() == 0 {
				s.resetTimerInLock(DataRequestRoundInterval)
			} else {
				s.resetTimerInLock(DataRequestNodeInterval)
			}
		}
	}
	s.deactivateInLock()
}

func (s *dataSyncer) cancelTimerInLock() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *dataSyncer) onTimer() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.timer != nil {
		s.timer = nil
	}
	s.notifyInLock()
}

func (s *dataSyncer) resetTimerInLock(d time.Duration) {
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(d, s.onTimer)
}

func (s *dataSyncer) Term() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.bd = nil
	s.cancelTimerInLock()
	s.notifyInLock()
}

func (s *dataSyncer) Start() {
	go s.serve()
}

type onDataHandler func()

func (r onDataHandler) OnData(value []byte, builder merkle.Builder) error {
	r()
	return nil
}

func (s *dataSyncer) moveCheckedToReadyInLock() {
	if s.checked.size() == 0 {
		return
	}
	peers := s.checked.peerList()
	for _, p := range peers {
		s.ready.push(p)
	}
	s.checked.clear()
}

func (s *dataSyncer) AddRequest(id db.BucketID, key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.bd != nil {
		if id.Hasher() == nil {
			return errors.IllegalArgumentError.Errorf("InvalidBucket(id=%q)", id)
		}
		bk, err := s.bd.Database().GetBucket(id)
		if err != nil {
			return err
		}
		if value, err := bk.Get(key); err != nil {
			return err
		} else if len(value) != 0 {
			return nil
		}
		s.log.Infof("DataSyncer: REQUEST id=%s key=%#x", id, key)
		s.bd.RequestData(id, key, onDataHandler(func() {
			s.log.Infof("DataSyncer: ADD id=%s key=%#x", id, key)
		}))
		s.moveCheckedToReadyInLock()
		s.notifyInLock()
		return nil
	} else {
		return errors.InvalidStateError.Errorf("Terminated")
	}
}

func newDataSyncer(database db.Database, client *client, logger log.Logger) *dataSyncer {
	s := &dataSyncer{
		client:  client,
		sent:    newPeerPool(),
		ready:   newPeerPool(),
		checked: newPeerPool(),
		bd:      merkle.NewBuilderWithRawDatabase(database),
		log:     logger,
	}
	s.waiter = sync.NewCond(&s.lock)
	return s
}
