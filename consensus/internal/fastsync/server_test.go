package fastsync

import (
	"crypto/rand"
	"testing"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
)

const tNumLongBlocks = 1
const tNumShortBlocks = 10
const tNumBlocks = tNumShortBlocks + tNumLongBlocks

type serverTestSetUp struct {
	*fastSyncTestSetUp

	nm  *tNetworkManager
	nm2 *tNetworkManager
	r2  *tReactor
	ph2 module.ProtocolHandler
	m   Manager
}

func createABytes(l int) []byte {
	b := make([]byte, l)
	rand.Read(b)
	return b
}

func newServerTestSetUp(t *testing.T) *serverTestSetUp {
	s := &serverTestSetUp{}
	s.fastSyncTestSetUp = newFastSyncTestSetUp(t)
	s.nm = newTNetworkManager()
	s.nm2 = newTNetworkManager()
	s.nm.join(s.nm2)
	s.r2 = newTReactor()
	var err error
	s.ph2, err = s.nm2.RegisterReactorForStreams("fastsync", s.r2, protocols, configFastSyncPriority)
	assert.Nil(t, err)
	s.m, err = newManager(s.nm, s.bm)
	assert.Nil(t, err)
	s.m.StartServer()
	return s
}

func (s *serverTestSetUp) sendBlockRequest(ph module.ProtocolHandler, rid uint32, height int64) {
	bs := codec.MustMarshalToBytes(&BlockRequest{
		RequestID: rid,
		Height:    height,
	})
	err := s.ph2.Unicast(protoBlockRequest, bs, s.nm.id)
	assert.Nil(s.t, err)
}

func (s *serverTestSetUp) assertEqualReceiveEvent(pi module.ProtocolInfo, msg interface{}, id module.PeerID, actual interface{}) {
	b := codec.MustMarshalToBytes(msg)
	assert.Equal(s.t, tReceiveEvent{pi, b, id}, actual)
}

func TestServer_Success(t *testing.T) {
	s := newServerTestSetUp(t)
	s.sendBlockRequest(s.ph2, 0, 0)
	ev := <-s.r2.ch
	md := &BlockMetadata{0, int32(len(s.rawBlocks[0])), s.votes[1]}
	s.assertEqualReceiveEvent(protoBlockMetadata, md, s.nm.id, ev)
	recv := 0
	data := make([]byte, md.BlockLength)
	for recv < int(md.BlockLength) {
		ev = <-s.r2.ch
		ev := ev.(tReceiveEvent)
		var msg BlockData
		codec.UnmarshalFromBytes(ev.b, &msg)
		t.Logf("ev : %v\n", msg)
		copy(data[recv:], msg.Data)
		recv += len(msg.Data)
	}
	assert.Equal(t, data, s.rawBlocks[0])
}

func TestServer_Fail(t *testing.T) {
}

func TestServer_Queue(t *testing.T) {
}

func TestServer_Cancel(t *testing.T) {
}
