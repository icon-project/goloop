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

package sync2

import (
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

type testWatcher struct {
	lock  sync.Mutex
	peers map[string]*peer
	t     *testing.T
}

func (w *testWatcher) OnPeerJoin(p *peer) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// w.t.Logf("OnJoinPeer(%s)", p.id)

	key := PeerIDToKey(p.id)
	if p2, ok := w.peers[key]; ok {
		if p2 != p {
			w.t.Errorf("already joined peer old=%s new=%s", p2, p)
			w.t.Fail()
		}
	} else {
		w.peers[key] = p
	}
}

func (w *testWatcher) OnPeerLeave(p *peer) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// w.t.Logf("OnLeavePeer(%s)", p.id)

	key := PeerIDToKey(p.id)
	if p2, ok := w.peers[key]; !ok {
		w.t.Errorf("unknown peer=%s", p)
		w.t.Fail()
	} else if p2 != p {
		w.t.Errorf("different peer=%s remove=%s", p2, p)
		w.t.Fail()
	} else {
		delete(w.peers, key)
	}
}

func (w *testWatcher) monitorPeersOf(r SyncReactor) {
	w.lock.Lock()
	defer w.lock.Unlock()

	peers := r.WatchPeers(w)
	for _, p := range peers {
		key := PeerIDToKey(p.id)
		w.peers[key] = p
	}
}

func (w *testWatcher) getPeerIDs() []module.PeerID {
	w.lock.Lock()
	defer w.lock.Unlock()

	var ids []module.PeerID
	for _, p := range w.peers {
		ids = append(ids, p.id)
	}
	return ids
}

func newTestWatcher(t *testing.T) *testWatcher {
	return &testWatcher{
		peers: make(map[string]*peer),
		t:     t,
	}
}

func TestReactorCommon(t *testing.T) {
	dbase := db.NewMapDB()
	logger := log.New()
	logger.SetOutput(io.Discard)

	type args struct {
		sr SyncReactor
	}
	cases := []struct {
		name    string
		args    args
		version byte
	}{
		{"V1", args{newReactorV1(dbase, logger)}, 1},
		{"V2", args{newReactorV2(dbase, logger)}, 2},
	}

	p1 := network.NewPeerIDFromAddress(common.MustNewAddressFromString("hx77"))
	p2 := network.NewPeerIDFromAddress(common.MustNewAddressFromString("hx78"))
	p3 := network.NewPeerIDFromAddress(common.MustNewAddressFromString("hx79"))

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			sr := tt.args.sr
			reactor := sr.(module.Reactor)
			assert.Equal(t, tt.version, sr.GetVersion())

			reactor.OnJoin(p1)
			watcher := newTestWatcher(t)
			watcher.monitorPeersOf(sr)

			peers := watcher.getPeerIDs()
			assert.Equal(t, 1, len(peers))
			assert.Contains(t, peers, p1)

			reactor.OnJoin(p2)
			ids := watcher.getPeerIDs()
			assert.Equal(t, 2, len(ids))
			assert.Contains(t, ids, p1)
			assert.Contains(t, ids, p2)

			reactor.OnJoin(p3)
			ids = watcher.getPeerIDs()
			assert.Equal(t, 3, len(ids))
			assert.Contains(t, ids, p1)
			assert.Contains(t, ids, p2)
			assert.Contains(t, ids, p3)

			reactor.OnLeave(p2)
			reactor.OnLeave(p3)
			ids = watcher.getPeerIDs()
			assert.NotContains(t, ids, p2)
			assert.NotContains(t, ids, p3)

			wg := new(sync.WaitGroup)
			const joinLeaveCount = 10
			const joinLeaveDelayMax = 20
			testForPeer := func(id module.PeerID) {
				for i := 0; i < joinLeaveCount; i++ {
					reactor.OnJoin(id)

					w := newTestWatcher(t)
					w.monitorPeersOf(sr)

					ids := w.getPeerIDs()
					assert.Contains(t, ids, id)

					reactor.OnJoin(id)

					time.Sleep(time.Millisecond * time.Duration(rand.Intn(joinLeaveDelayMax)))

					reactor.OnLeave(id)
					ids = w.getPeerIDs()
					assert.NotContains(t, ids, id)

					sr.UnwatchPeers(w)

					time.Sleep(time.Millisecond * time.Duration(rand.Intn(joinLeaveDelayMax)))
				}
				wg.Done()
			}

			wg.Add(2)
			go testForPeer(p2)
			go testForPeer(p3)
			wg.Wait()
		})
	}
}
