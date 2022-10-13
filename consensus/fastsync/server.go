package fastsync

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	configChunkSize = 1024 * 10
)

type MessageItem struct {
	pi module.ProtocolInfo
	b  []byte
}

type speer struct {
	id    module.PeerID
	msgCh chan MessageItem

	// toStopCh notifies moderators to stop.
	// Notified from server.stop and server.onLeave
	toStopCh chan struct{}

	// stopCh notifies message sender server.onReceive not to send more
	// and handler sconHandler.handle to stop.
	stopCh chan struct{}

	// stoppedCh is notified by message handler when it ends.
	stoppedCh chan struct{}
}

type server struct {
	common.Mutex
	nm    module.NetworkManager
	ph    module.ProtocolHandler
	bm    module.BlockManager
	bpp   BlockProofProvider
	log   log.Logger
	peers []*speer

	running bool
}

func newServer(
	nm module.NetworkManager,
	ph module.ProtocolHandler,
	bm module.BlockManager,
	bpp BlockProofProvider,
	logger log.Logger,
) *server {
	s := &server{
		nm:  nm,
		ph:  ph,
		bm:  bm,
		bpp: bpp,
		log: logger,
	}
	return s
}

func (s *server) start() {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		s.running = true
		pids := s.ph.GetPeers()
		for _, id := range pids {
			s._addPeer(id)
		}
	}
}

const msgChanSize = 10

func (s *server) _addPeer(id module.PeerID) {
	toStopCh := make(chan struct{}, 1)
	stopCh := make(chan struct{})
	speer := &speer{
		id:        id,
		msgCh:     make(chan MessageItem, msgChanSize),
		toStopCh:  toStopCh,
		stopCh:    stopCh,
		stoppedCh: make(chan struct{}, 1),
	}
	s.peers = append(s.peers, speer)
	h := newSConHandler(
		speer.msgCh, stopCh, speer.stoppedCh,
		speer.id, s.ph, s.bm, s.bpp, s.log,
	)
	go h.handle()
	go func() {
		<-toStopCh
		close(stopCh)
	}()
}

func (s *server) stop() {
	s.Lock()
	defer s.Unlock()

	if s.running {
		s.running = false
		for _, p := range s.peers {
			p.toStopCh <- struct{}{}
			// wait for handle goroutine to end
			pp := p // to capture loop var
			s.CallAfterUnlock(func() {
				<-pp.stoppedCh
			})
		}
		s.peers = nil
	}
}

func (s *server) onJoin(id module.PeerID) {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		return
	}
	for _, p := range s.peers {
		if p.id.Equal(id) {
			return
		}
	}
	s._addPeer(id)
}

func (s *server) onLeave(id module.PeerID) {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		return
	}
	for i, p := range s.peers {
		if p.id.Equal(id) {
			last := len(s.peers) - 1
			s.peers[i] = s.peers[last]
			s.peers[last] = nil
			s.peers = s.peers[:last]
			p.toStopCh <- struct{}{}
			return
		}
	}
}

func (s *server) onReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		return
	}
	for _, p := range s.peers {
		if p.id.Equal(id) {
			p := p // capture loop var
			s.CallAfterUnlock(func() {
				// do not send message if stopCh is closed
				select {
				case <-p.stopCh:
				case p.msgCh <- MessageItem{pi, b}:
				}
			})
			break
		}
	}
}

type sconHandler struct {
	msgCh     <-chan MessageItem
	stopCh    <-chan struct{}
	stoppedCh chan<- struct{}
	id        module.PeerID
	ph        module.ProtocolHandler
	bm        module.BlockManager
	bpp       BlockProofProvider
	log       log.Logger

	nextItems []*BlockRequest
	buf       *bytes.Buffer
	requestID uint32
	nextMsgPI module.ProtocolInfo
	nextMsg   []byte
}

func newSConHandler(
	msgCh <-chan MessageItem,
	stopCh <-chan struct{},
	stoppedCh chan<- struct{},
	id module.PeerID,
	ph module.ProtocolHandler,
	bm module.BlockManager,
	bpp BlockProofProvider,
	logger log.Logger,
) *sconHandler {
	h := &sconHandler{
		msgCh:     msgCh,
		stopCh:    stopCh,
		stoppedCh: stoppedCh,
		id:        id,
		ph:        ph,
		bm:        bm,
		bpp:       bpp,
		log: logger.WithFields(log.Fields{
			"peer": common.HexPre(id.Bytes()),
		}),
	}
	return h
}

func (h *sconHandler) cancelAllRequests() {
	h.nextMsg = nil
	h.buf = nil
	h.nextItems = nil
}

func (h *sconHandler) updateCurrentTask() {
	if len(h.nextItems) == 0 {
		return
	}
	ni := h.nextItems[0]
	copy(h.nextItems, h.nextItems[1:])
	h.nextItems = h.nextItems[:len(h.nextItems)-1]
	h.requestID = ni.RequestID
	blk, err := h.bm.GetBlockByHeight(ni.Height)
	if err != nil {
		h.nextMsgPI = ProtoBlockMetadata
		h.nextMsg = codec.MustMarshalToBytes(&BlockMetadata{
			RequestID:   ni.RequestID,
			BlockLength: -1,
			Proof:       nil,
		})
		h.buf = nil
		return
	}
	proof, err := h.bpp.GetBlockProof(ni.Height, ni.ProofOption)
	if err != nil {
		h.nextMsgPI = ProtoBlockMetadata
		h.nextMsg = codec.MustMarshalToBytes(&BlockMetadata{
			RequestID:   ni.RequestID,
			BlockLength: -1,
			Proof:       nil,
		})
		h.buf = nil
		return
	}
	h.buf = bytes.NewBuffer(nil)
	h.log.Must(blk.MarshalHeader(h.buf))
	h.log.Must(blk.MarshalBody(h.buf))
	h.nextMsgPI = ProtoBlockMetadata
	h.nextMsg = codec.MustMarshalToBytes(&BlockMetadata{
		RequestID:   ni.RequestID,
		BlockLength: int32(h.buf.Len()),
		Proof:       proof,
	})
}

func (h *sconHandler) updateNextMsg() {
	if h.nextMsg != nil {
		return
	}
	if h.buf == nil {
		h.updateCurrentTask()
		return
	}
	chunk := make([]byte, configChunkSize)
	var data []byte
	n, err := h.buf.Read(chunk)
	if n > 0 {
		data = chunk[:n]
	} else if n == 0 && err == io.EOF {
		h.updateCurrentTask()
		return
	} else {
		// n==0 && err!=io.EOF
		h.log.Panicf("n=%d, err=%+v\n", n, err)
	}
	var msg BlockData
	msg.RequestID = h.requestID
	msg.Data = data
	h.nextMsgPI = ProtoBlockData
	h.nextMsg = codec.MustMarshalToBytes(&msg)
}

const maxNextItems = 10

func (h *sconHandler) processMsg(msgItem *MessageItem) {
	if msgItem.pi == ProtoBlockRequest {
		var msg BlockRequest
		_, err := codec.UnmarshalFromBytes(msgItem.b, &msg)
		if err != nil {
			h.log.Debugf("Fail to decode request %+v", err)
			return
		}
		if len(h.nextItems) < maxNextItems {
			h.log.Debugf("Received BlockRequest %d\n", msg.Height)
			h.nextItems = append(h.nextItems, &msg)
		} else {
			h.log.Debugf("Received BlockRequest %d ignored\n", msg.Height)
		}
	} else if msgItem.pi == ProtoCancelAllBlockRequests {
		h.cancelAllRequests()
	} else {
		h.log.Debugf("Unknown msg PI %x", msgItem.pi)
	}
}

func (h *sconHandler) processMsgs0Timeout() bool {
	for {
		select {
		case <-h.stopCh:
			return false
		case msgItem, ok := <-h.msgCh:
			if !ok {
				return false
			}
			h.processMsg(&msgItem)
		default:
			return true
		}
	}
}

func (h *sconHandler) processMsgs(timeout time.Duration) bool {
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		select {
		case <-h.stopCh:
			return false
		case msgItem, ok := <-h.msgCh:
			if !timer.Stop() {
				<-timer.C
			}
			if !ok {
				return false
			}
			h.processMsg(&msgItem)
		case <-timer.C:
			return true
		}
	} else if timeout < 0 {
		select {
		case <-h.stopCh:
			return false
		case msgItem, ok := <-h.msgCh:
			if !ok {
				return false
			}
			h.processMsg(&msgItem)
		}
	}
	return h.processMsgs0Timeout()
}

func (h *sconHandler) handle() {
	timeout := time.Duration(0)
	for {
		if !h.processMsgs(timeout) {
			break
		}
		h.updateNextMsg()

		timeout = 0
		if h.nextMsg != nil {
			err := h.ph.Unicast(h.nextMsgPI, h.nextMsg, h.id)
			if err == nil {
				h.nextMsg = nil
			} else if isTemporary(err) {
				h.log.Warnf("unicast temporary error %+v\n", err)
				timeout = configSendInterval
			} else {
				h.log.Warnf("unicast error %+v\n", err)
				h.cancelAllRequests()
			}
		} else {
			timeout = -1
		}
	}
	h.stoppedCh <- struct{}{}
}
