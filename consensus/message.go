package consensus

import (
	"fmt"
	"io"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

var msgCodec = codec.BC

const (
	ProtoProposal module.ProtocolInfo = iota << 8
	ProtoBlockPart
	ProtoVote
	ProtoRoundState
	ProtoVoteList
)

type protocolConstructor struct {
	proto       module.ProtocolInfo
	constructor func() Message
}

var protocolConstructors = [...]protocolConstructor{
	{ProtoProposal, func() Message { return NewProposalMessage() }},
	{ProtoBlockPart, func() Message { return newBlockPartMessage() }},
	{ProtoVote, func() Message { return newVoteMessage() }},
	{ProtoRoundState, func() Message { return newRoundStateMessage() }},
	{ProtoVoteList, func() Message { return newVoteListMessage() }},
}

func UnmarshalMessage(sp uint16, bs []byte) (Message, error) {
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

type Message interface {
	Verify() error
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

type ProposalMessage struct {
	signedBase
	proposal
}

func NewProposalMessage() *ProposalMessage {
	msg := &ProposalMessage{}
	msg.signedBase._byteser = msg
	return msg
}

func (msg *ProposalMessage) Verify() error {
	if err := msg._HR.verify(); err != nil {
		return err
	}
	if msg.BlockPartSetID == nil || msg.BlockPartSetID.Count <= 0 || msg.POLRound < -1 || msg.POLRound >= msg.Round {
		return errors.New("bad field value")
	}
	return msg.signedBase.verify()
}

func (msg *ProposalMessage) subprotocol() uint16 {
	return uint16(ProtoProposal)
}

func (msg *ProposalMessage) String() string {
	return fmt.Sprintf("ProposalMessage{H:%d R:%d BPSID:%v Addr:%v}", msg.Height, msg.Round, msg.BlockPartSetID, common.HexPre(msg.address().ID()))
}

type BlockPartMessage struct {
	// V1 Fields
	// for debugging
	Height int64
	Index  uint16

	BlockPart []byte

	// V2 Fields
	Nonce int32
}

func newBlockPartMessage() *BlockPartMessage {
	return &BlockPartMessage{}
}

func (msg *BlockPartMessage) Verify() error {
	if msg.Height <= 0 {
		return errors.Errorf("bad height %v", msg.Height)
	}
	if len(msg.BlockPart) > ConfigBlockPartSize*2 {
		return errors.Errorf("bad height %v", msg.Height)
	}
	_, err := NewPart(msg.BlockPart)
	return err
}

func (msg *BlockPartMessage) subprotocol() uint16 {
	return uint16(ProtoBlockPart)
}

func (msg *BlockPartMessage) String() string {
	return fmt.Sprintf("BlockPartMessage{H:%d,I:%d}", msg.Height, msg.Index)
}

type blockVoteByteser struct {
	msg *VoteMessage
}

func (v *blockVoteByteser) bytes() []byte {
	bv := struct {
		blockVoteBase
		Timestamp int64
	}{
		v.msg.blockVoteBase,
		v.msg.Timestamp,
	}
	return msgCodec.MustMarshalToBytes(&bv)
}

type VoteMessage struct {
	signedBase
	voteBase
	Timestamp      int64
	NTSDProofParts [][]byte
}

func newVoteMessage() *VoteMessage {
	msg := &VoteMessage{}
	msg.signedBase._byteser = &blockVoteByteser{
		msg: msg,
	}
	return msg
}

// NewVoteMessageFromBlock creates a new VoteMessage from block data.
// pcm is blk.Height()-1's nextPCM. Used only for test
func NewVoteMessageFromBlock(
	w module.Wallet,
	wp module.WalletProvider,
	blk module.BlockData,
	round int32,
	voteType VoteType,
	bpsIDAndNTSVoteCount *PartSetIDAndAppData,
	ts int64,
	nid int,
	pcm module.BTPProofContextMap,
) (*VoteMessage, error) {
	vm := newVoteMessage()
	vm.Height = blk.Height()
	vm.Round = round
	vm.Type = voteType
	vm.BlockID = blk.ID()
	vm.BlockPartSetIDAndNTSVoteCount = bpsIDAndNTSVoteCount
	vm.Timestamp = ts
	_ = vm.Sign(w)
	bd, err := blk.BTPDigest()
	if err != nil {
		return nil, err
	}
	if pcm == nil {
		return vm, nil
	}
	for _, ntd := range bd.NetworkTypeDigests() {
		pc, err := pcm.ProofContextFor(ntd.NetworkTypeID())
		if errors.Is(err, errors.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		vm.NTSVoteBases = append(vm.NTSVoteBases, ntsVoteBase{
			NetworkTypeID:          ntd.NetworkTypeID(),
			NetworkTypeSectionHash: ntd.NetworkTypeSectionHash(),
		})
		ntsd := pc.NewDecision(
			module.SourceNetworkUID(nid),
			ntd.NetworkTypeID(),
			blk.Height(),
			round,
			ntd.NetworkTypeSectionHash(),
		)
		pp, err := pc.NewProofPart(ntsd.Hash(), wp)
		if err != nil {
			return nil, err
		}
		vm.NTSDProofParts = append(vm.NTSDProofParts, pp.Bytes())
	}
	return vm, nil
}

// NewVoteMessage returns a new VoteMessage. Used only for test.
func NewVoteMessage(
	w module.Wallet,
	voteType VoteType, height int64, round int32, id []byte,
	partSetID *PartSetID, ts int64,
	ntsHashEntries []module.NTSHashEntryFormat,
	ntsdProofParts [][]byte,
	ntsVoteCount int,
) *VoteMessage {
	vm := newVoteMessage()
	vm.Height = height
	vm.Round = round
	vm.Type = voteType
	vm.BlockID = id
	vm.BlockPartSetIDAndNTSVoteCount = partSetID.WithAppData(uint16(ntsVoteCount))
	vm.Timestamp = ts
	_ = vm.Sign(w)
	for _, ntsHashEntry := range ntsHashEntries {
		vm.NTSVoteBases = append(vm.NTSVoteBases, ntsVoteBase(ntsHashEntry))
	}
	vm.NTSDProofParts = ntsdProofParts
	return vm
}

func (msg *VoteMessage) EqualExceptSigs(msg2 *VoteMessage) bool {
	return msg.voteBase.Equal(&msg2.voteBase) && msg.Timestamp == msg2.Timestamp
}

func (msg *VoteMessage) Verify() error {
	if err := msg._HR.verify(); err != nil {
		return err
	}
	if msg.Type != VoteTypePrevote && msg.Type != VoteTypePrecommit {
		return errors.New("bad field value")
	}
	if msg.Type == VoteTypePrevote && len(msg.NTSVoteBases) > 0 {
		return errors.Errorf(
			"prevote with NTSVotes len=%d", len(msg.NTSVoteBases),
		)
	}
	if len(msg.NTSVoteBases) != len(msg.NTSDProofParts) {
		return errors.Errorf("NTS loop len mismatch NTSVoteBasesLen=%d NTSDProofPartsLen=%d", len(msg.NTSVoteBases), len(msg.NTSDProofParts))
	}
	verifyProofCount := msg.Type == VoteTypePrecommit && msg.BlockPartSetIDAndNTSVoteCount != nil
	if verifyProofCount && int(msg.BlockPartSetIDAndNTSVoteCount.AppData()) != len(msg.NTSDProofParts) {
		return errors.Errorf("NTS loop len mismatch appData=%d NTSDProofPartsLen=%d", msg.BlockPartSetIDAndNTSVoteCount.AppData(), len(msg.NTSDProofParts))
	}
	return msg.signedBase.verify()
}

func (msg *VoteMessage) VerifyNTSDProofParts(
	pcm module.BTPProofContextMap,
	srcUID []byte,
	expValIndex int,
) error {
	for i, nvb := range msg.NTSVoteBases {
		pc, err := pcm.ProofContextFor(nvb.NetworkTypeID)
		if err != nil {
			return err
		}
		ntsd := pc.NewDecision(
			srcUID, nvb.NetworkTypeID,
			msg.Height, msg.Round, nvb.NetworkTypeSectionHash,
		)
		pp, err := pc.NewProofPartFromBytes(msg.NTSDProofParts[i])
		if err != nil {
			return err
		}
		idx, err := pc.VerifyPart(ntsd.Hash(), pp)
		if err != nil {
			return err
		}
		if expValIndex != idx {
			return errors.Errorf("invalid validator index exp=%d actual=%d ntid=%d", expValIndex, idx, nvb.NetworkTypeID)
		}
	}
	return nil
}

func (msg *VoteMessage) subprotocol() uint16 {
	return uint16(ProtoVote)
}

func (msg *VoteMessage) String() string {
	return fmt.Sprintf("VoteMessage{%s,H:%d,R:%d,BlockID:%v,Addr:%v}", msg.Type, msg.Height, msg.Round, common.HexPre(msg.BlockID), common.HexPre(msg.address().ID()))
}

func (msg *VoteMessage) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	err = e2.EncodeMulti(
		&msg.Signature,
		&msg.Height,
		&msg.Round,
		&msg.Type,
		&msg.BlockID,
		&msg.BlockPartSetIDAndNTSVoteCount,
		&msg.Timestamp,
	)
	if err != nil {
		return err
	}
	if len(msg.NTSVoteBases) == 0 {
		return nil
	}
	e3, err := e2.EncodeList()
	if err != nil {
		return err
	}
	for i, ntsVote := range msg.NTSVoteBases {
		err = e3.EncodeListOf(
			&ntsVote.NetworkTypeID,
			&ntsVote.NetworkTypeSectionHash,
			msg.NTSDProofParts[i],
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (msg *VoteMessage) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var ntsVotes []struct {
		ntsVoteBase
		NTSDProofPart []byte
	}
	cnt, err := d2.DecodeMulti(
		&msg.Signature,
		&msg.Height,
		&msg.Round,
		&msg.Type,
		&msg.BlockID,
		&msg.BlockPartSetIDAndNTSVoteCount,
		&msg.Timestamp,
		&ntsVotes,
	)
	if cnt == 7 && err == io.EOF {
		msg.NTSVoteBases = nil
		msg.NTSDProofParts = nil
		return nil
	}
	msg.NTSVoteBases = make([]ntsVoteBase, 0, len(ntsVotes))
	msg.NTSDProofParts = make([][]byte, 0, len(ntsVotes))
	for _, ntsVote := range ntsVotes {
		msg.NTSVoteBases = append(msg.NTSVoteBases, ntsVote.ntsVoteBase)
		msg.NTSDProofParts = append(msg.NTSDProofParts, ntsVote.NTSDProofPart)
	}
	return err
}

type peerRoundState struct {
	_HR
	PrevotesMask   *BitArray
	PrecommitsMask *BitArray
	BlockPartsMask *BitArray
	Sync           bool
}

func (prs peerRoundState) String() string {
	return fmt.Sprintf("PeerRoundState{H:%v R:%v PV:%v PC:%v BP:%v Sync:%t}", prs.Height, prs.Round, prs.PrevotesMask, prs.PrecommitsMask, prs.BlockPartsMask, prs.Sync)
}

type RoundStateMessage struct {
	peerRoundState
	Timestamp int64
	// TODO: add LastMaskType, LastIndex
}

func (msg RoundStateMessage) String() string {
	return fmt.Sprintf("PeerRoundStateMessage{H:%v R:%v PV:%v PC:%v BP:%v Sync:%t}", msg.Height, msg.Round, msg.PrevotesMask, msg.PrecommitsMask, msg.BlockPartsMask, msg.Sync)
}

func newRoundStateMessage() *RoundStateMessage {
	return &RoundStateMessage{
		Timestamp: time.Now().UnixNano(),
	}
}

func (msg *RoundStateMessage) Verify() error {
	if err := msg.peerRoundState._HR.verify(); err != nil {
		return err
	}
	if msg.PrevotesMask == nil || msg.PrecommitsMask == nil {
		return errors.Errorf("invalid RoundStateMessage PrevotesMask=%v PRecommitMask=%v", msg.PrevotesMask, msg.PrecommitsMask)
	}
	if err := msg.PrevotesMask.Verify(); err != nil {
		return err
	}
	if err := msg.PrecommitsMask.Verify(); err != nil {
		return err
	}
	if msg.BlockPartsMask != nil {
		if err := msg.BlockPartsMask.Verify(); err != nil {
			return err
		}
	}
	return nil
}

func (msg *RoundStateMessage) subprotocol() uint16 {
	return uint16(ProtoRoundState)
}

type VoteListMessage struct {
	VoteList *VoteList
}

func newVoteListMessage() *VoteListMessage {
	return &VoteListMessage{}
}

func (msg *VoteListMessage) Verify() error {
	if msg.VoteList == nil {
		return errors.Errorf("nil VoteList")
	}
	return msg.VoteList.Verify()
}

func (msg VoteListMessage) String() string {
	return fmt.Sprintf("VoteListMessage%+v", msg.VoteList)
}

func (msg *VoteListMessage) subprotocol() uint16 {
	return uint16(ProtoVoteList)
}
