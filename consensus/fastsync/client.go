package fastsync

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

const (
	configSendInterval      = time.Millisecond * 100
	configTimeout           = time.Millisecond * 3500
	configMaxPendingResults = 10
	configMaxActive         = 3
)

type client struct {
	common.Mutex
	nm NetworkManager
	ph module.ProtocolHandler
	bm BlockManager

	fetchID uint16
	fr      *fetchRequest
}

type blockResult struct {
	id    module.PeerID
	blk   module.Block
	votes module.CommitVoteSet
	cl    *client
	fr    *fetchRequest
}

func (br *blockResult) Block() module.Block {
	return br.blk
}

func (br *blockResult) Votes() module.CommitVoteSet {
	return br.votes
}

func (br *blockResult) Consume() {
	br.cl.Lock()
	defer br.cl.Unlock()

	cl := br.cl
	fr := br.fr
	if cl.fr != fr {
		return
	}

	cnt := br.blk.Height() - fr.consumeOffset
	fr.consumeOffset = fr.consumeOffset + cnt
	copy(fr.pendingResults, fr.pendingResults[cnt:])
	for i := len(fr.pendingResults) - int(cnt); i < len(fr.pendingResults); i++ {
		fr.pendingResults[i] = nil
	}
	fr._reschedule()
}

type peer struct {
	id        module.PeerID
	requestID uint16
	f         *fetcher
}

type fetchRequest struct {
	cl         *client
	heightSet  *heightSet
	cb         FetchCallback
	cvsDecoder module.CommitVoteSetDecoder
	maxActive  int

	validPeers     []*peer
	nActivePeers   int
	prevBlock      module.Block
	consumeOffset  int64
	notifyOffset   int64
	pendingResults []*blockResult
}

func newClient(nm NetworkManager, ph module.ProtocolHandler,
	bm BlockManager) *client {
	cl := &client{}
	cl.nm = nm
	cl.ph = ph
	cl.bm = bm
	return cl
}

func (cl *client) fetchBlocks(
	begin int64,
	end int64,
	prev module.Block,
	f module.CommitVoteSetDecoder,
	cb FetchCallback,
) (*fetchRequest, error) {
	cl.Lock()
	defer cl.Unlock()

	if cl.fr != nil {
		return nil, errors.New("already in use")
	}

	cl.fetchID++
	fr := &fetchRequest{}
	fr.cl = cl
	fr.heightSet = newHeightSet(begin, end)
	fr.cb = cb
	fr.cvsDecoder = f
	fr.maxActive = configMaxActive

	peerIDs := cl.nm.GetPeers()
	fr.validPeers = make([]*peer, len(peerIDs))
	for i, id := range peerIDs {
		fr.validPeers[i] = &peer{id, 0, nil}
	}
	fr.nActivePeers = 0
	fr.prevBlock = prev
	fr.consumeOffset = begin
	fr.notifyOffset = begin
	fr.pendingResults = make([]*blockResult, configMaxPendingResults)
	fr._reschedule()
	cl.fr = fr
	return fr, nil
}

func (cl *client) onReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (rebr bool, err error) {
	cl.Lock()
	defer cl.Unlock()

	fr := cl.fr
	if fr == nil {
		return false, nil
	}
	for _, p := range fr.validPeers {
		if p.f != nil && p.id.Equal(id) {
			f := p.f
			cl.CallAfterUnlock(func() {
				f.onReceive(pi, b)
			})
			return false, nil
		}
	}
	return false, nil
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

func (cl *client) onResult(f *fetcher, err error, blk module.Block, votes module.CommitVoteSet) {
	cl.Lock()
	defer cl.Unlock()

	if isNoBlock(err) {
		log.Printf("onResult %v\n", err)
	} else if err != nil {
		log.Printf("onResult %+v\n", err)
	} else {
		log.Printf("onResult %d\n", blk.Height())
	}

	fr := cl.fr
	if fr != f.fr {
		if logDebug {
			log.Printf("onResult: fr %p != f.fr %p\n", fr, f.fr)
		}
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
			start := fr.notifyOffset - fr.consumeOffset
			for i := start; i < int64(len(fr.pendingResults)); i++ {
				ri := fr.pendingResults[i]
				if fr.pendingResults[i] != nil && fr.pendingResults[i].id.Equal(f.id) {
					fr.pendingResults[i] = nil
					fr.heightSet.add(ri.blk.Height())
				}
			}
		}
		if fr.nActivePeers != 0 {
			fr._reschedule()
		} else {
			cb := fr.cb
			cl.CallAfterUnlock(func() {
				cb.OnEnd(nil)
			})
		}
		return
	}
	pi, p := cl._findPeerByFetcher(f)
	if p == nil {
		return
	}
	if logDebug {
		log.Printf("height=%d consumeOffset=%d\n", f.height, fr.consumeOffset)
	}
	fr.pendingResults[f.height-fr.consumeOffset] = &blockResult{
		id:    f.id,
		blk:   blk,
		votes: votes,
		cl:    cl,
		fr:    fr,
	}
	fr.nActivePeers--
	p.f = nil

	offset := int(fr.notifyOffset - fr.consumeOffset)
	i := offset
	prevBlock := fr.prevBlock // last notified block
	for ; i < len(fr.pendingResults); i++ {
		ri := fr.pendingResults[i]
		if ri == nil {
			break
		}
		err := VerifyBlock(ri.blk, prevBlock, ri.votes)
		if err != nil {
			last := len(fr.validPeers) - 1
			fr.validPeers[pi] = fr.validPeers[last]
			fr.validPeers[last] = nil
			fr.validPeers = fr.validPeers[:last]
			fr.pendingResults[i] = nil
			fr.heightSet.add(ri.blk.Height())
			for j := i + 1; j < len(fr.pendingResults); j++ {
				if fr.pendingResults[j] != nil && fr.pendingResults[j].id.Equal(f.id) {
					fr.pendingResults[j] = nil
					fr.heightSet.add(fr.pendingResults[j].blk.Height())
				}
			}
			log.Printf("onResult: %+v\n", err)
			break
		}
		prevBlock = ri.blk
	}

	cnt := i - offset
	notifyItems := make([]*blockResult, cnt)
	copy(notifyItems, fr.pendingResults[offset:i])
	fr.notifyOffset = fr.notifyOffset + int64(cnt)
	fr.prevBlock = prevBlock
	fr._reschedule()
	if logDebug {
		log.Printf("onResult: %d block(s) notification\n", cnt)
	}
	if cnt > 0 {
		cb := fr.cb
		cl.CallAfterUnlock(func() {
			for _, ni := range notifyItems {
				cb.OnBlock(ni)
			}
		})
	}
	if fr.notifyOffset > fr.heightSet.end {
		cb := fr.cb
		cl.CallAfterUnlock(func() {
			// TODO: notify order
			cb.OnEnd(nil)
		})
	}
	return
}

var errNoBlock = errors.New("errNoBlock")

func isNoBlock(err error) bool {
	return errors.Cause(err) == errNoBlock
}

type fstep byte

const (
	fstepSend fstep = iota
	fstepWaitResp
	fstepWaitData
	fstepFin // canceled or succeeded
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

func (fr *fetchRequest) cancel() bool {
	fr.cl.Lock()
	defer fr.cl.Unlock()

	// TODO implement
	return false
}

func (f *fetcher) _doSend() {
	var msg BlockRequest
	msg.RequestID = f.requestID
	msg.Height = f.height
	bs := codec.MustMarshalToBytes(&msg)
	log.Printf("Request %d, %d\n", f.requestID, f.height)
	err := f.cl.ph.Unicast(protoBlockRequest, bs, f.id)
	if f.timer != nil {
		f.timer.Stop()
		f.timer = nil
	}
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
	} else {
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
		if err == nil {
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
		if logDebug {
			log.Printf("onReceive BlockMetadata rid=%d, len=%d\n", msg.RequestID, msg.BlockLength)
		}
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
		if logDebug {
			log.Printf("onReceive BlockData rid=%d, data len=%d left=%d\n", msg.RequestID, len(msg.Data), f.left)
		}
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
				blk, err := f.cl.bm.NewBlockFromReader(r)
				if err != nil {
					f.cl.onResult(f, err, nil, nil)
				} else if blk.Height() != f.height {
					f.cl.onResult(f, errors.Errorf("bad Height"), nil, nil)
				} else {
					vs := f.fr.cvsDecoder(f.voteList)
					f.cl.onResult(f, nil, blk, vs)
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

func VerifyBlock(
	b module.Block,
	prev module.Block,
	vote module.CommitVoteSet,
) error {
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return errors.New("bad prev ID")
	}
	if err := vote.Verify(b, prev.NextValidators()); err != nil {
		return err
	}
	return nil
}
