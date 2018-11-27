package consensus

import (
	"errors"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

var msgCodec = codec.MP

const (
	protoProposal protocolInfo = iota << 8
	protoBlockPart
	protoVote
)

type unmarshaler func([]byte) (message, error)

type protocolUnmarshaler struct {
	proto     protocolInfo
	unmarshal unmarshaler
}

var protocolUnmarshalers = [...]protocolUnmarshaler{
	{protoProposal, unmarshalProposalMessage},
	{protoBlockPart, unmarshalBlockPartMessage},
	{protoVote, unmarshalVoteMessage},
}

func unmarshalMessage(sp module.ProtocolInfo, bs []byte) (message, error) {
	for _, pu := range protocolUnmarshalers {
		if sp.Uint16() == pu.proto.Uint16() {
			return pu.unmarshal(bs)
		}
	}
	return nil, errors.New("Unknown protocol")
}

type message interface {
	height() int64
	verify() error
	dispatch(cs *consensus) (bool, error)
}

type payloadBase struct {
	Height int64
	Round  int
}

func (b *payloadBase) height() int64 {
	return b.Height
}

// TODO remove duplicated code
type proposal struct {
	payloadBase
	BlockPartsHash []byte // Merkle root of MPT[rlp(index)]BlockPartBytes
	NumBlockParts  int
	POLRound       int
	POLBlockID     []byte
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

func unmarshalProposalMessage(bs []byte) (message, error) {
	msg := &proposalMessage{}
	msg.signedBase._byteser = msg
	if _, err := msgCodec.UnmarshalFromBytes(bs, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (msg *proposalMessage) verify() error {
	if msg.Height < 0 || msg.Round < 0 || msg.NumBlockParts <= 0 || msg.POLRound < -1 {
		return errors.New("bad field value")
	}
	return msg.signedBase.verify()
}

func (msg *proposalMessage) dispatch(cs *consensus) (bool, error) {
	return cs.receiveProposal(msg)
}

type blockPartMessage struct {
	payloadBase
	Height int64
	Round  int
	Index  int
	Proof  [][]byte // Merkle proof to root
}

func unmarshalBlockPartMessage(bs []byte) (message, error) {
	msg := &blockPartMessage{}
	if _, err := msgCodec.UnmarshalFromBytes(bs, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (msg *blockPartMessage) verify() error {
	if msg.Height < 0 || msg.Round < 0 || msg.Index < 0 {
		return errors.New("bad field value")
	}
	return nil
}

func (msg *blockPartMessage) dispatch(cs *consensus) (bool, error) {
	return cs.receiveBlockPart(msg)
}

type voteType byte

const (
	voteTypePrevote voteType = iota
	voteTypePrecommit
	numberOfVoteTypes
)

type vote struct {
	payloadBase
	Type    voteType
	BlockID []byte
}

func (v *vote) bytes() []byte {
	bs, err := msgCodec.MarshalToBytes(v)
	if err != nil {
		panic(err)
	}
	return bs
}

type voteMessage struct {
	signedBase
	vote
}

func unmarshalVoteMessage(bs []byte) (message, error) {
	msg := &voteMessage{}
	msg.signedBase._byteser = msg
	if _, err := msgCodec.UnmarshalFromBytes(bs, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (msg *voteMessage) verify() error {
	if msg.Height < 0 || msg.Round < 0 || msg.Type < voteTypePrevote || msg.Type > numberOfVoteTypes {
		return errors.New("bad field value")
	}
	return msg.signedBase.verify()
}

func (msg *voteMessage) dispatch(cs *consensus) (bool, error) {
	return cs.receiveVote(msg)
}
