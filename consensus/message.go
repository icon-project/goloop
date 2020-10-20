package consensus

import (
	"bytes"
	"fmt"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

var msgCodec = codec.BC

const (
	protoProposal module.ProtocolInfo = iota << 8
	protoBlockPart
	protoVote
	protoRoundState
	protoVoteList
)

type protocolConstructor struct {
	proto       module.ProtocolInfo
	constructor func() message
}

var protocolConstructors = [...]protocolConstructor{
	{protoProposal, func() message { return newProposalMessage() }},
	{protoBlockPart, func() message { return newBlockPartMessage() }},
	{protoVote, func() message { return newVoteMessage() }},
	{protoRoundState, func() message { return newRoundStateMessage() }},
	{protoVoteList, func() message { return newVoteListMessage() }},
}

func unmarshalMessage(sp uint16, bs []byte) (message, error) {
	for _, pc := range protocolConstructors {
		if sp == uint16(pc.proto) {
			msg := pc.constructor()
			if _, err := msgCodec.UnmarshalFromBytes(bs, msg); err != nil {
				return nil, err
			}
			return msg, nil
		}
	}
	return nil, errors.New("Unknown protocol")
}

type message interface {
	verify() error
	subprotocol() uint16
}

type _HR struct {
	Height int64
	Round  int32
}

func (hr *_HR) height() int64 {
	return hr.Height
}

func (hr *_HR) round() int32 {
	return hr.Round
}

func (hr *_HR) verify() error {
	if hr.Height <= 0 {
		return errors.Errorf("bad height %v", hr.Height)
	}
	if hr.Round < 0 {
		return errors.Errorf("bad round %v", hr.Round)
	}
	return nil
}

type proposal struct {
	_HR
	BlockPartSetID *PartSetID
	POLRound       int32
}

func (p *proposal) bytes() []byte {
	bs, err := msgCodec.MarshalToBytes(p)
	if err != nil {
		panic(err)
	}
	return bs
}

type proposalMessage struct {
	signedBase
	proposal
}

func newProposalMessage() *proposalMessage {
	msg := &proposalMessage{}
	msg.signedBase._byteser = msg
	return msg
}

func (msg *proposalMessage) verify() error {
	if err := msg._HR.verify(); err != nil {
		return err
	}
	if msg.BlockPartSetID.Count <= 0 || msg.POLRound < -1 || msg.POLRound >= msg.Round {
		return errors.New("bad field value")
	}
	return msg.signedBase.verify()
}

func (msg *proposalMessage) subprotocol() uint16 {
	return uint16(protoProposal)
}

func (msg *proposalMessage) String() string {
	return fmt.Sprintf("ProposalMessage{H:%d R:%d BPSID:%v Addr:%v}", msg.Height, msg.Round, msg.BlockPartSetID, common.HexPre(msg.address().ID()))
}

type blockPartMessage struct {
	// V1 Fields
	// for debugging
	Height int64
	Index  uint16

	BlockPart []byte

	// V2 Fields
	Nonce int32
}

func newBlockPartMessage() *blockPartMessage {
	return &blockPartMessage{}
}

func (msg *blockPartMessage) verify() error {
	if msg.Height <= 0 {
		return errors.Errorf("bad height %v", msg.Height)
	}
	return nil
}

func (msg *blockPartMessage) subprotocol() uint16 {
	return uint16(protoBlockPart)
}

func (msg *blockPartMessage) String() string {
	return fmt.Sprintf("BlockPartMessage{H:%d,I:%d}", msg.Height, msg.Index)
}

type voteType byte

const (
	voteTypePrevote voteType = iota
	voteTypePrecommit
	numberOfVoteTypes
)

func (vt voteType) String() string {
	switch vt {
	case voteTypePrevote:
		return "PreVote"
	case voteTypePrecommit:
		return "PreCommit"
	default:
		return "Unknown"
	}
}

type voteBase struct {
	_HR
	Type           voteType
	BlockID        []byte
	BlockPartSetID *PartSetID
}

func (vb *voteBase) Equal(v2 *voteBase) bool {
	return vb.Height == v2.Height && vb.Round == v2.Round && vb.Type == v2.Type &&
		bytes.Equal(vb.BlockID, v2.BlockID) &&
		vb.BlockPartSetID.Equal(v2.BlockPartSetID)
}

func (vb voteBase) String() string {
	return fmt.Sprintf("{%s H:%d R:%d BID:%v BPSID:%v}", vb.Type, vb.Height, vb.Round, common.HexPre(vb.BlockID), vb.BlockPartSetID)
}

type vote struct {
	voteBase
	Timestamp int64
}

func (v *vote) Equal(v2 *vote) bool {
	return v.voteBase.Equal(&v2.voteBase) && v.Timestamp == v2.Timestamp
}

func (v *vote) bytes() []byte {
	bs, err := msgCodec.MarshalToBytes(v)
	if err != nil {
		panic(err)
	}
	return bs
}

func (v *vote) String() string {
	return fmt.Sprintf("Vote{%s H=%d R=%d bid=%v}", v.Type, v.Height, v.Round, common.HexPre(v.BlockID))
}

type voteMessage struct {
	signedBase
	vote
}

func newVoteMessage() *voteMessage {
	msg := &voteMessage{}
	msg.signedBase._byteser = msg
	return msg
}

func (msg *voteMessage) verify() error {
	if err := msg._HR.verify(); err != nil {
		return err
	}
	if msg.Type < voteTypePrevote || msg.Type > numberOfVoteTypes {
		return errors.New("bad field value")
	}
	return msg.signedBase.verify()
}

func (msg *voteMessage) subprotocol() uint16 {
	return uint16(protoVote)
}

func (msg *voteMessage) String() string {
	return fmt.Sprintf("VoteMessage{%s,H:%d,R:%d,BlockID:%v,Addr:%v}", msg.Type, msg.Height, msg.Round, common.HexPre(msg.BlockID), common.HexPre(msg.address().ID()))
}

type peerRoundState struct {
	_HR
	PrevotesMask   *bitArray
	PrecommitsMask *bitArray
	BlockPartsMask *bitArray
	Sync           bool
}

func (prs peerRoundState) String() string {
	return fmt.Sprintf("PeerRoundState{H:%v R:%v PV:%v PC:%v BP:%v Sync:%t}", prs.Height, prs.Round, prs.PrevotesMask, prs.PrecommitsMask, prs.BlockPartsMask, prs.Sync)
}

type roundStateMessage struct {
	peerRoundState
	Timestamp int64
	// TODO: add LastMaskType, LastIndex
}

func (msg roundStateMessage) String() string {
	return fmt.Sprintf("PeerRoundStateMessage{H:%v R:%v PV:%v PC:%v BP:%v Sync:%t}", msg.Height, msg.Round, msg.PrevotesMask, msg.PrecommitsMask, msg.BlockPartsMask, msg.Sync)
}

func newRoundStateMessage() *roundStateMessage {
	return &roundStateMessage{
		Timestamp: time.Now().UnixNano(),
	}
}

func (msg *roundStateMessage) verify() error {
	if err := msg.peerRoundState._HR.verify(); err != nil {
		return err
	}
	return nil
}

func (msg *roundStateMessage) subprotocol() uint16 {
	return uint16(protoRoundState)
}

type voteListMessage struct {
	VoteList *voteList
}

func newVoteListMessage() *voteListMessage {
	return &voteListMessage{}
}

func (msg *voteListMessage) verify() error {
	if msg.VoteList == nil {
		return errors.Errorf("nil VoteList")
	}
	return nil
}

func (msg voteListMessage) String() string {
	return fmt.Sprintf("VoteListMessage%+v", msg.VoteList)
}

func (msg *voteListMessage) subprotocol() uint16 {
	return uint16(protoVoteList)
}
