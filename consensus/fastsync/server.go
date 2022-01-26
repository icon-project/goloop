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
	id        module.PeerID
	msgCh     chan MessageItem
	cancelCh  chan struct{}
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
		pids := s.nm.GetPeers()
		for _, id := range pids {
			s._addPeer(id)
		}
	}
}

func (s *server) _addPeer(id module.PeerID) {
	speer := &speer{
		id:        id,
		msgCh:     make(chan MessageItem),
		cancelCh:  make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
	s.peers = append(s.peers, speer)
	h := newSConHandler(
		speer.msgCh, speer.cancelCh, speer.stoppedCh, speer.id,
		s.ph, s.bm, s.bpp, s.log,
	)
	go h.handle()
}

func (s *server) stop() {
	s.Lock()
	defer s.Unlock()

	if s.running {
		s.running = false
		for _, p := range s.peers {
			close(p.cancelCh)
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
			pp := p // to capture loop var
			close(p.cancelCh)
			s.CallAfterUnlock(func() {
				<-pp.stoppedCh
			})
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
			if pi == ProtoCancelAllBlockRequests {
				s.CallAfterUnlock(func() {
					p.cancelCh <- struct{}{}
				})
			}
			s.CallAfterUnlock(func() {
				p.msgCh <- MessageItem{pi, b}
			})
			break
		}
	}
}

type sconHandler struct {
	msgCh     <-chan MessageItem
	cancelCh  <-chan struct{}
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
	cancelCh <-chan struct{},
	stoppedCh chan<- struct{},
	id module.PeerID,
	ph module.ProtocolHandler,
	bm module.BlockManager,
	bpp BlockProofProvider,
	logger log.Logger,
) *sconHandler {
	h := &sconHandler{
		msgCh:     msgCh,
		cancelCh:  cancelCh,
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
	for {
		msgItem := <-h.msgCh
		if msgItem.pi == ProtoCancelAllBlockRequests {
			break
		}
	}
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

func (h *sconHandler) processRequestMsg(msgItem *MessageItem) {
	if msgItem.pi == ProtoBlockRequest {
		var msg BlockRequest
		_, err := codec.UnmarshalFromBytes(msgItem.b, &msg)
		if err != nil {
			// TODO log
			return
		}
		h.log.Debugf("Received BlockRequest %d\n", msg.Height)
		h.nextItems = append(h.nextItems, &msg)
	}
}

func (h *sconHandler) handle() {
loop:
	for {
		select {
		case _, more := <-h.cancelCh:
			if !more {
				break loop
			}
			h.cancelAllRequests()
			continue loop
		default:
		}

		h.updateNextMsg()
		var err error
		if h.nextMsg != nil {
			err = h.ph.Unicast(h.nextMsgPI, h.nextMsg, h.id)
			if err == nil {
				// TODO: refactor
				h.nextMsg = nil
				h.updateNextMsg()
			} else if !isTemporary(err) {
				h.log.Warnf("unicast error %+v\n", err)
				h.cancelAllRequests()
			}
		}

		// if packet is dropped too much, use ticker to slow down sending
		if len(h.nextMsg) > 0 && err == nil {
			select {
			case _, more := <-h.cancelCh:
				if !more {
					break loop
				}
				h.cancelAllRequests()
				continue loop
			case msgItem := <-h.msgCh:
				h.processRequestMsg(&msgItem)
			default:
			}
		} else if len(h.nextMsg) > 0 && isTemporary(err) {
			timer := time.NewTimer(configSendInterval)
			select {
			case _, more := <-h.cancelCh:
				timer.Stop()
				if !more {
					break loop
				}
				h.cancelAllRequests()
				continue loop
			case msgItem := <-h.msgCh:
				timer.Stop()
				h.processRequestMsg(&msgItem)
			case <-timer.C:
			}
		} else {
			select {
			case _, more := <-h.cancelCh:
				if !more {
					break loop
				}
				h.cancelAllRequests()
				continue
			case msgItem := <-h.msgCh:
				h.processRequestMsg(&msgItem)
			}
		}
	}
	h.stoppedCh <- struct{}{}
}
