package fastsync

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
)

type clientTestSetUp struct {
	t   *testing.T
	bm  *tBlockManager
	nms []*tNetworkManager
	phs []module.ProtocolHandler

	reactors  []*tReactor
	m         Manager
	votes     [][]byte
	rawBlocks [][]byte
	blks      []module.Block
	cb        *tFetchCallback
}

func newClientTestSetUp(t *testing.T, n int) *clientTestSetUp {
	s := &clientTestSetUp{}
	s.t = t
	s.bm = newTBlockManager()
	s.votes = make([][]byte, tNumBlocks)
	s.rawBlocks = make([][]byte, tNumBlocks)
	s.blks = make([]module.Block, tNumBlocks)
	for i := 0; i < tNumBlocks; i++ {
		var b []byte
		if i < tNumLongBlocks {
			b = createABytes(configChunkSize * 10)
		} else {
			b = createABytes(2)
		}
		if i > 0 {
			s.blks[i] = newTBlock(int64(i), b[:1], s.blks[i-1].ID(), b[1:])
			s.votes[i] = s.blks[i-1].ID()
		} else {
			s.blks[i] = newTBlock(int64(i), b[:1], nil, b[1:])
			s.votes[i] = nil
		}
		buf := bytes.NewBuffer(nil)
		err := s.blks[i].MarshalHeader(buf)
		assert.Nil(s.t, err)
		err = s.blks[i].MarshalBody(buf)
		assert.Nil(s.t, err)
		s.rawBlocks[i] = buf.Bytes()
	}
	s.nms = make([]*tNetworkManager, n)
	s.reactors = make([]*tReactor, n)
	s.phs = make([]module.ProtocolHandler, n)
	for i := 0; i < n; i++ {
		s.nms[i] = newTNetworkManager()
		if i > 0 {
			s.nms[0].join(s.nms[i])
			s.reactors[i] = newTReactor()
			var err error
			s.phs[i], err = s.nms[i].RegisterReactorForStreams("fastsync", s.reactors[i], protocols, configFastSyncPriority)
			assert.Nil(t, err)
		}
	}
	var err error
	s.m, err = newManager(s.nms[0], s.bm)
	assert.Nil(t, err)
	s.cb = newTFetchCallback()
	return s
}

type tOnBlockEvent struct {
	blk module.Block
	vs  module.CommitVoteSet
	br  BlockResult
}

type tOnEndEvent struct {
	err error
}

type tFetchCallback struct {
	ch chan interface{}
}

func (cb *tFetchCallback) OnBlock(br BlockResult) {
	blk := br.Block()
	vs := br.Votes()
	cb.ch <- tOnBlockEvent{blk, vs, br}
}

func (cb *tFetchCallback) OnEnd(err error) {
	cb.ch <- tOnEndEvent{err}
}

func newTFetchCallback() *tFetchCallback {
	cb := &tFetchCallback{
		ch: make(chan interface{}),
	}
	return cb
}

func (s *clientTestSetUp) assertEqualReceiveEvent(pi module.ProtocolInfo, msg interface{}, id module.PeerID, actual interface{}) {
	b := codec.MustMarshalToBytes(msg)
	assert.Equal(s.t, tReceiveEvent{pi, b, id}, actual)
}

func (s *clientTestSetUp) send(ph module.ProtocolHandler, pi module.ProtocolInfo, msg interface{}, id module.PeerID) {
	bs := codec.MustMarshalToBytes(msg)
	err := ph.Unicast(pi, bs, id)
	assert.Nil(s.t, err)
}

func (s *clientTestSetUp) respondBlockRequest(
	ph module.ProtocolHandler,
	rid uint32,
	blk []byte,
	votes []byte,
	id module.PeerID,
) {
	s.send(ph, protoBlockMetadata, &BlockMetadata{rid, int32(len(blk)), votes}, id)
	s.send(ph, protoBlockData, &BlockData{rid, blk}, id)
}

func (s *clientTestSetUp) assertBlockEvent(expected []byte, actual interface{}) {
	bev, ok := actual.(tOnBlockEvent)
	assert.True(s.t, ok, "event is not tOnBlockEvent: %s\n", fmt.Sprintf("%T %#v", actual, actual))
	buf := bytes.NewBuffer(nil)
	err := bev.blk.MarshalHeader(buf)
	assert.Nil(s.t, err)
	err = bev.blk.MarshalBody(buf)
	assert.Nil(s.t, err)
	assert.Equal(s.t, expected, buf.Bytes())
}

func (s *clientTestSetUp) assertEndEvent(expected error, actual interface{}) {
	eev, ok := actual.(tOnEndEvent)
	assert.True(s.t, ok, "event is not tOnEndEvent: %s\n", fmt.Sprintf("%T %#v", actual, actual))
	assert.Equal(s.t, expected, eev.err)
}

func (s *clientTestSetUp) assertNoEvent(ch chan interface{}) {
	select {
	case ev := <-ch:
		assert.Failf(s.t, "unexpected event", " %T %#v\n", ev, ev)
	default:
	}
}

func TestClient_Success(t *testing.T) {
	s := newClientTestSetUp(t, 2)
	_, err := s.m.FetchBlocks(1, 10, s.blks[0], newTCommitVoteSet, s.cb)
	assert.Nil(t, err)
	ev := <-s.reactors[1].ch
	s.assertEqualReceiveEvent(protoBlockRequest, &BlockRequest{0x10000, 1}, s.nms[0].id, ev)

	s.respondBlockRequest(s.phs[1], 0x10000, s.rawBlocks[1], s.votes[2], s.nms[0].id)

	ev2 := <-s.cb.ch
	s.assertBlockEvent(s.rawBlocks[1], ev2)
	ev2.(tOnBlockEvent).br.Consume()
}

func TestClient_SuccessMulti(t *testing.T) {
	s := newClientTestSetUp(t, 3)
	_, err := s.m.FetchBlocks(1, 3, s.blks[0], newTCommitVoteSet, s.cb)
	assert.Nil(t, err)

	ev := <-s.reactors[1].ch
	s.assertEqualReceiveEvent(protoBlockRequest, &BlockRequest{0x10000, 1}, s.nms[0].id, ev)

	ev = <-s.reactors[2].ch
	s.assertEqualReceiveEvent(protoBlockRequest, &BlockRequest{0x10000, 2}, s.nms[0].id, ev)

	s.respondBlockRequest(s.phs[2], 0x10000, s.rawBlocks[2], s.votes[3], s.nms[0].id)
	s.assertNoEvent(s.cb.ch)

	ev = <-s.reactors[2].ch
	s.assertEqualReceiveEvent(protoBlockRequest, &BlockRequest{0x10001, 3}, s.nms[0].id, ev)

	s.respondBlockRequest(s.phs[2], 0x10001, s.rawBlocks[3], s.votes[4], s.nms[0].id)
	s.assertNoEvent(s.cb.ch)

	s.respondBlockRequest(s.phs[1], 0x10000, s.rawBlocks[1], s.votes[2], s.nms[0].id)

	ev2 := <-s.cb.ch
	s.assertBlockEvent(s.rawBlocks[1], ev2)
	ev2.(tOnBlockEvent).br.Consume()

	ev2 = <-s.cb.ch
	s.assertBlockEvent(s.rawBlocks[2], ev2)
	ev2.(tOnBlockEvent).br.Consume()

	ev2 = <-s.cb.ch
	s.assertBlockEvent(s.rawBlocks[3], ev2)
	ev2.(tOnBlockEvent).br.Consume()

	ev2 = <-s.cb.ch
	s.assertEndEvent(nil, ev2)
}
