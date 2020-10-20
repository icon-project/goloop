package fastsync

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	configSendInterval      = time.Millisecond * 100
	configTimeout           = time.Millisecond * 3500
	configMaxPendingResults = 10
	configMaxActive         = 3
)

type client struct {
	common.Mutex
	nm  module.NetworkManager
	ph  module.ProtocolHandler
	bm  module.BlockManager
	log log.Logger

	fetchID uint16
	fr      *fetchRequest
}

type blockResult struct {
	id    module.PeerID
	blk   module.BlockData
	votes []byte
	cl    *client
	fr    *fetchRequest
}

func (br *blockResult) Block() module.BlockData {
	return br.blk
}

func (br *blockResult) Votes() []byte {
	return br.votes
}

func (br *blockResult) Consume() {
	br.cl.Lock()
	defer br.cl.Unlock()

	cl := br.cl
	cl.log.Tracef("Consume %d\n", br.blk.Height())
	fr := br.fr
	if cl.fr != fr {
		return
	}

	fr.consumeOffset++
	copy(fr.pendingResults, fr.pendingResults[1:])
	fr.pendingResults[len(fr.pendingResults)-1] = nil
	fr._reschedule()
	if fr.consumeOffset > fr.heightSet.end || fr.nActivePeers == 0 && fr.pendingResults[0] == nil {
		cb := fr.cb
		cl.log.Tracef("OnEnd Consume %d nActivePeers:%d pendingResult[0]:%p\n", br.blk.Height(), fr.nActivePeers, fr.pendingResults[0])
		fr._cancel()
		go cb.OnEnd(nil)
		return
	}
	if fr.pendingResults[0] != nil {
		cl.notifyBlockResult()
	}
}

func (br *blockResult) Reject() {
	br.cl.Lock()
	defer br.cl.Unlock()

	cl := br.cl
	cl.log.Tracef("Reject %d\n", br.blk.Height())
	fr := br.fr
	if cl.fr != fr {
		return
	}

	for i, p := range fr.validPeers {
		if p.id.Equal(br.id) {
			last := len(fr.validPeers) - 1
			fr.validPeers[i] = fr.validPeers[last]
			fr.validPeers[last] = nil
			fr.validPeers = fr.validPeers[:last]
			if p.f != nil {
				p.f.cancel()
				fr.nActivePeers--
				fr._reschedule()
			}
			break
		}
	}
	fr.pendingResults[0] = nil
	fr.heightSet.add(br.blk.Height())
	for i := 1; i < len(fr.pendingResults); i++ {
		if fr.pendingResults[i] != nil && fr.pendingResults[i].id.Equal(br.id) {
			fr.heightSet.add(fr.pendingResults[i].blk.Height())
			fr.pendingResults[i] = nil
		}
	}
	if fr.nActivePeers != 0 {
		fr._reschedule()
	} else {
		cb := fr.cb
		cl.log.Tracef("OnEnd Reject %d\n", br.blk.Height())
		fr._cancel()
		go cb.OnEnd(nil)
	}
}

type peer struct {
	id        module.PeerID
	requestID uint16
	f         *fetcher
}

type fetchRequest struct {
	cl        *client
	heightSet *heightSet
	cb        FetchCallback
	maxActive int

	validPeers     []*peer
	nActivePeers   int
	consumeOffset  int64
	pendingResults []*blockResult
}

func newClient(nm module.NetworkManager, ph module.ProtocolHandler,
	bm module.BlockManager, logger log.Logger) *client {
	cl := &client{}
	cl.nm = nm
	cl.ph = ph
	cl.bm = bm
	cl.log = logger
	return cl
}

func (cl *client) fetchBlocks(
	begin int64,
	end int64,
	cb FetchCallback,
) (*fetchRequest, error) {
	cl.Lock()
	defer cl.Unlock()

	cl.log.Debugf("fetchBlocks begin:%d end:%d\n", begin, end)

	if cl.fr != nil {
		cl.log.Debugf("fetchBlocks begin:%d end:%d - already in use\n", begin, end)
		return nil, errors.New("already in use")
	}

	cl.fetchID++
	fr := &fetchRequest{}
	fr.cl = cl
	fr.heightSet = newHeightSet(begin, end)
	fr.cb = cb
	fr.maxActive = configMaxActive

	peerIDs := cl.nm.GetPeers()
	fr.validPeers = make([]*peer, len(peerIDs))
	for i, id := range peerIDs {
		fr.validPeers[i] = &peer{id, 0, nil}
	}
	fr.nActivePeers = 0
	fr.consumeOffset = begin
	fr.pendingResults = make([]*blockResult, configMaxPendingResults)
	fr._reschedule()
	cl.fr = fr
	return fr, nil
}

func (cl *client) onReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) {
	cl.Lock()
	defer cl.Unlock()

	fr := cl.fr
	if fr == nil {
		return
	}
	for _, p := range fr.validPeers {
		if p.f != nil && p.id.Equal(id) {
			f := p.f
			cl.CallAfterUnlock(func() {
				f.onReceive(pi, b)
			})
			return
		}
	}
}

func (cl *client) onJoin(id module.PeerID) {
	cl.Lock()
	defer cl.Unlock()

	fr := cl.fr
	if fr == nil {
		return
	}
	for _, p := range fr.validPeers {
		if p.id.Equal(id) {
			return
		}
	}
	peer := &peer{id, 0, nil}
	fr.validPeers = append(fr.validPeers, peer)
	fr._reschedule()
}

func (cl *client) onLeave(id module.PeerID) {
	cl.Lock()
	defer cl.Unlock()

	fr := cl.fr
	if fr == nil {
		return
	}
	for i, p := range fr.validPeers {
		if p.id.Equal(id) {
			last := len(fr.validPeers) - 1
			fr.validPeers[i] = fr.validPeers[last]
			fr.validPeers[last] = nil
			fr.validPeers = fr.validPeers[:last]
			if p.f != nil {
				p.f.cancel()
				fr.nActivePeers--
				fr._reschedule()
			}
			return
		}
	}
}

func (fr *fetchRequest) _reschedule() {
	for {
		if fr.nActivePeers >= fr.maxActive {
			return
		}
		if len(fr.validPeers) == fr.nActivePeers {
			return
		}
		l, ok := fr.heightSet.getLowest()
		if !ok || fr.consumeOffset+int64(len(fr.pendingResults)) <= l {
			return
		}
		var peer *peer
		for _, p := range fr.validPeers {
			if p.f == nil {
				peer = p
				break
			}
		}
		if peer == nil {
			panic("wrong validPeers state")
		}
		requestID := uint32(fr.cl.fetchID)<<16 | uint32(peer.requestID)
		peer.f = fr.newFetcher(peer.id, l, requestID)
		peer.requestID++
		fr.heightSet.popLowest()
		fr.nActivePeers++
	}
}

func (cl *client) _findPeerByFetcher(f *fetcher) (int, *peer) {
	for i, p := range cl.fr.validPeers {
		if p.f == f {
			return i, p
		}
	}
	return -1, nil
}

func (cl *client) onResult(f *fetcher, err error, blk module.BlockData, votes []byte) {
	cl.Lock()
	defer cl.Unlock()

	if isNoBlock(err) {
		cl.log.Debugf("onResult %v\n", err)
	} else if err != nil {
		cl.log.Debugf("onResult %+v\n", err)
	} else {
		cl.log.Debugf("onResult %d\n", blk.Height())
	}

	fr := cl.fr
	if fr != f.fr {
		cl.log.Tracef("onResult: fr %p != f.fr %p\n", fr, f.fr)
		return
	}

	if err != nil {
		i, p := cl._findPeerByFetcher(f)
		if p == nil {
			return
		}
		fr.nActivePeers--
		last := len(fr.validPeers) - 1
		fr.validPeers[i] = fr.validPeers[last]
		fr.validPeers[last] = nil
		fr.validPeers = fr.validPeers[:last]
		fr.heightSet.add(f.height)
		if !isNoBlock(err) {
			for i := 1; i < len(fr.pendingResults); i++ {
				ri := fr.pendingResults[i]
				if fr.pendingResults[i] != nil && fr.pendingResults[i].id.Equal(f.id) {
					fr.pendingResults[i] = nil
					fr.heightSet.add(ri.blk.Height())
				}
			}
		}
		if fr.nActivePeers != 0 {
			fr._reschedule()
		} else if fr.pendingResults[0] == nil {
			cb := fr.cb
			fr._cancel()
			cl.CallAfterUnlock(func() {
				cl.log.Tracef("OnEnd onResult\n")
				cb.OnEnd(nil)
			})
		}
		return
	}
	_, p := cl._findPeerByFetcher(f)
	if p == nil {
		return
	}
	cl.log.Tracef("height=%d consumeOffset=%d\n", f.height, fr.consumeOffset)
	offset := f.height - fr.consumeOffset
	fr.pendingResults[offset] = &blockResult{
		id:    f.id,
		blk:   blk,
		votes: votes,
		cl:    cl,
		fr:    fr,
	}
	fr.nActivePeers--
	p.f = nil

	fr._reschedule()
	if offset == 0 {
		cl.notifyBlockResult()
	}
}

func (cl *client) notifyBlockResult() {
	fr := cl.fr
	br := fr.pendingResults[0]
	cl.log.Tracef("onResult: block notification\n")
	cb := fr.cb
	go cb.OnBlock(br)
}

var errNoBlock = errors.New("errNoBlock")

func isNoBlock(err error) bool {
	return errors.Is(err, errNoBlock)
}

type fstep byte

//goland:noinspection GoUnusedConst
const (
	fstepSend fstep = iota
	fstepWaitResp
	fstepWaitData
	fstepFin  // canceled or succeeded
)

type fetcher struct {
	common.Mutex
	id        module.PeerID
	height    int64
	requestID uint32
	fr        *fetchRequest
	cl        *client

	step     fstep
	timer    *time.Timer
	left     int32
	voteList []byte
	dataList [][]byte
}

func (fr *fetchRequest) newFetcher(id module.PeerID, height int64, requestID uint32) *fetcher {
	f := &fetcher{
		id:        id,
		height:    height,
		requestID: requestID,
		fr:        fr,
		cl:        fr.cl,
	}

	f.Lock()
	defer f.Unlock()
	f._doSend()
	return f
}

func (fr *fetchRequest) _cancel() bool {
	if fr.cl.fr == fr {
		fr.cl.fr = nil
	}

	for _, p := range fr.validPeers {
		if p.f != nil {
			p.f.cancel()
		}
	}

	return false
}

func (fr *fetchRequest) cancel() bool {
	fr.cl.Lock()
	defer fr.cl.Unlock()

	return fr._cancel()
}

func (f *fetcher) _doSend() {
	var msg BlockRequest
	msg.RequestID = f.requestID
	msg.Height = f.height
	bs := codec.MustMarshalToBytes(&msg)
	f.cl.log.Debugf("Request RequestID:%d, Height:%d\n", f.requestID, f.height)
	if f.timer != nil {
		f.timer.Stop()
		f.timer = nil
	}
	err := f.cl.ph.Unicast(protoBlockRequest, bs, f.id)
	if err == nil {
		f.step = fstepWaitResp
		var timer *time.Timer
		timer = time.AfterFunc(configTimeout, func() {
			f.Lock()
			defer f.Unlock()

			if f.timer != timer {
				return
			}
			f.timer = nil
			f._cancel()
			cl := f.cl
			f.CallAfterUnlock(func() {
				cl.onResult(f, errors.Errorf("Timed out"), nil, nil)
			})
		})
		f.timer = timer
	} else if isTemporary(err) {
		var timer *time.Timer
		timer = time.AfterFunc(configSendInterval, func() {
			f.Lock()
			defer f.Unlock()

			if f.timer == timer {
				f.timer = nil
				f._doSend()
			}
		})
		f.timer = timer
	} else {
		f._cancel()
		cl := f.cl
		f.CallAfterUnlock(func() {
			cl.onResult(f, err, nil, nil)
		})
	}
}

func (f *fetcher) cancel() {
	f.Lock()
	defer f.Unlock()

	f._cancel()
}

func (f *fetcher) _cancel() {
	if f.timer != nil {
		f.timer.Stop()
		f.timer = nil
	}
	var msg CancelAllBlockRequests
	bs := codec.MustMarshalToBytes(&msg)
	f.step = fstepFin
	for {
		err := f.cl.ph.Unicast(protoCancelAllBlockRequests, bs, f.id)
		if err == nil || !isTemporary(err) {
			return
		}
		time.Sleep(configSendInterval)
	}
}

func (f *fetcher) onReceive(pi module.ProtocolInfo, b []byte) {
	f.Lock()
	defer f.Unlock()

	if f.step == fstepWaitResp {
		if pi != protoBlockMetadata {
			return
		}
		var msg BlockMetadata
		_, err := codec.UnmarshalFromBytes(b, &msg)
		if err != nil {
			return
		}
		if msg.RequestID != f.requestID {
			return
		}
		f.cl.log.Tracef("onReceive BlockMetadata rid=%d, len=%d\n", msg.RequestID, msg.BlockLength)
		if msg.BlockLength < 0 {
			f.step = fstepFin
			if f.timer != nil {
				f.timer.Stop()
				f.timer = nil
			}
			f.CallAfterUnlock(func() {
				// TODO: remove stack trace
				f.cl.onResult(f, errNoBlock, nil, nil)
			})
		}
		f.left = msg.BlockLength
		f.voteList = msg.VoteList
		f.step = fstepWaitData
	} else if f.step == fstepWaitData {
		if pi != protoBlockData {
			return
		}
		var msg BlockData
		_, err := codec.UnmarshalFromBytes(b, &msg)
		if err != nil {
			return
		}
		if msg.RequestID != f.requestID {
			return
		}
		f.dataList = append(f.dataList, msg.Data)
		f.left -= int32(len(msg.Data))
		f.cl.log.Tracef("onReceive BlockData rid=%d, data len=%d left=%d\n", msg.RequestID, len(msg.Data), f.left)
		if f.left == 0 {
			f.step = fstepFin
			if f.timer != nil {
				f.timer.Stop()
				f.timer = nil
			}
			f.CallAfterUnlock(func() {
				bufs := make([]io.Reader, len(f.dataList))
				for i, d := range f.dataList {
					bufs[i] = bytes.NewReader(d)
				}
				r := io.MultiReader(bufs...)
				blk, err := f.cl.bm.NewBlockDataFromReader(r)
				if err != nil {
					f.cl.onResult(f, err, nil, nil)
				} else if blk.Height() != f.height {
					f.cl.onResult(f, errors.Errorf("bad Height"), nil, nil)
				} else {
					f.cl.onResult(f, nil, blk, f.voteList)
				}
			})
		} else if f.left < 0 {
			f.step = fstepFin
			if f.timer != nil {
				f.timer.Stop()
				f.timer = nil
			}
			cl := f.cl
			f.CallAfterUnlock(func() {
				cl.onResult(f, errors.Errorf("bad data"), nil, nil)
			})
		}
	}
}

func isTemporary(err error) bool {
	ne, ok := err.(module.NetworkError)
	return ok && ne.Temporary()
}
