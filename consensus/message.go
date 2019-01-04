package consensus

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

var msgCodec = codec.MP

const (
	protoProposal protocolInfo = iota << 8
	protoBlockPart
	protoVote
	protoRoundState
	protoVoteList
)

type protocolConstructor struct {
	proto       protocolInfo
	constructor func() message
}

var protocolConstructors = [...]protocolConstructor{
	{protoProposal, func() message { return newProposalMessage() }},
	{protoBlockPart, func() message { return newBlockPartMessage() }},
	{protoVote, func() message { return newVoteMessage() }},
	{protoRoundState, func() message { return newRoundStateMessage() }},
	{protoVoteList, func() message { return newVoteListMessage() }},
}

func unmarshalMessage(sp module.ProtocolInfo, bs []byte) (message, error) {
	for _, pc := range protocolConstructors {
		if sp.Uint16() == pc.proto.Uint16() {
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

func (msg *proposalMessage) String() string {
	return fmt.Sprintf("ProposalMessage[H=%d,R=%d,parts=%d,eoa=%s", msg.Height, msg.Round, msg.BlockPartSetID.Count, msg.address())
}

type blockPartMessage struct {
	Height    int64 // just for debug
	BlockPart []byte
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

func (msg *blockPartMessage) String() string {
	return fmt.Sprintf("BlockPartMessage[H=%d]", msg.Height)
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

type vote struct {
	_HR
	Type           voteType
	BlockID        []byte
	BlockPartSetID *PartSetID
}

func (v *vote) Equal(v2 *vote) bool {
	return v.Height == v2.Height && v.Round == v2.Round && v.Type == v2.Type &&
		bytes.Equal(v.BlockID, v2.BlockID) &&
		v.BlockPartSetID.Equal(v2.BlockPartSetID)
}

func (v *vote) bytes() []byte {
	bs, err := msgCodec.MarshalToBytes(v)
	if err != nil {
		panic(err)
	}
	return bs
}

func (v *vote) String() string {
	return fmt.Sprintf("Vote[%s,H=%d,R=%d,bid=<%x>]", v.Type, v.Height, v.Round, v.BlockID)
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

func (msg *voteMessage) String() string {
	return fmt.Sprintf("VoteMessage[%s,H=%d,R=%d,bid=<%x>,sig=%s]", msg.Type, msg.Height, msg.Round, msg.BlockID, msg.address())
}

type peerRoundState struct {
	_HR
	// 1 if requesting
	PrevotesMask   *bitArray
	PrecommitsMask *bitArray
	BlockPartsMask *bitArray
}

func (prs *peerRoundState) String() string {
	if prs == nil {
		return "peerRoundState=nil"
	}
	return fmt.Sprintf("H=%v,R=%v,PV=%v,PC=%v,BP=%v", prs.Height, prs.Round, prs.PrevotesMask, prs.PrecommitsMask, prs.BlockPartsMask)
}

type roundStateMessage struct {
	peerRoundState
	// TODO: add LastMaskType, LastIndex
}

func (msg *roundStateMessage) String() string {
	return fmt.Sprintf("RoundStateMessage:%v", msg.peerRoundState)
}

func newRoundStateMessage() *roundStateMessage {
	return &roundStateMessage{}
}

func (msg *roundStateMessage) verify() error {
	if err := msg.peerRoundState._HR.verify(); err != nil {
		return err
	}
	return nil
}

type voteListMessage struct {
	VoteList *roundVoteList
}

func newVoteListMessage() *voteListMessage {
	return &voteListMessage{}
}

func (msg *voteListMessage) verify() error {
	return nil
}
