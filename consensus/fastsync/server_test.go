package fastsync

import (
	"crypto/rand"
	"testing"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/internal/test"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

const tNumLongBlocks = 1
const tNumShortBlocks = 10
const tNumBlocks = tNumShortBlocks + tNumLongBlocks

type serverTestSetUp struct {
	*fastSyncTestSetUp

	nm  *test.NetworkManager
	nm2 *test.NetworkManager
	r2  *tReactor
	ph2 module.ProtocolHandler
	m   Manager
}

func createABytes(l int) []byte {
	b := make([]byte, l)
	_, _ = rand.Read(b)
	return b
}

func newServerTestSetUp(t *testing.T) *serverTestSetUp {
	s := &serverTestSetUp{}
	s.fastSyncTestSetUp = newFastSyncTestSetUp(t)
	s.nm = test.NewNetworkManager()
	s.nm2 = test.NewNetworkManager()
	s.nm.Join(s.nm2)
	s.r2 = newTReactor()
	var err error
	s.ph2, err = s.nm2.RegisterReactorForStreams("fastsync", module.ProtoFastSync, s.r2, protocols, configFastSyncPriority, module.NotRegisteredProtocolPolicyClose)
	assert.Nil(t, err)
	s.m, err = NewManager(s.nm, s.bm, s.bm, log.New(), 0)
	assert.Nil(t, err)
	s.m.StartServer()
	return s
}

func (s *serverTestSetUp) sendBlockRequest(ph module.ProtocolHandler, rid uint32, height int64) {
	bs := codec.MustMarshalToBytes(&BlockRequest{
		RequestID: rid,
		Height:    height,
	})
	err := s.ph2.Unicast(ProtoBlockRequest, bs, s.nm.ID)
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
	s.assertEqualReceiveEvent(ProtoBlockMetadata, md, s.nm.ID, ev)
	recv := 0
	data := make([]byte, md.BlockLength)
	for recv < int(md.BlockLength) {
		ev = <-s.r2.ch
		ev := ev.(tReceiveEvent)
		var msg BlockData
		codec.MustUnmarshalFromBytes(ev.b, &msg)
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
