package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

var vlCodec = codec.BC

type blockCommitVoteItem struct {
	Timestamp int64
	Signature common.Signature
}

type blockCommitVoteList struct {
	Round                         int32
	BlockPartSetIDAndNTSVoteCount *PartSetIDAndAppData
	Items                         []blockCommitVoteItem
	bytes                         []byte
}

func (bvl *blockCommitVoteList) BlockVoteSetBytes() []byte {
	if bvl.bytes == nil {
		bvl.bytes = vlCodec.MustMarshalToBytes(bvl)
	}
	return bvl.bytes
}

func (bvl *blockCommitVoteList) VerifyBlock(block module.BlockData, validators module.ValidatorList) ([]bool, error) {
	if block.Height() == 0 || validators == nil {
		if len(bvl.Items) == 0 {
			return nil, nil
		} else {
			return nil, errors.Errorf("voters for height 0 or nil validator list\n")
		}
	}
	vset := make([]bool, validators.Len())
	msg := newVoteMessage()
	msg.Height = block.Height()
	msg.Round = bvl.Round
	msg.Type = VoteTypePrecommit
	msg.SetRoundDecision(block.ID(), bvl.BlockPartSetIDAndNTSVoteCount, nil)
	for i, item := range bvl.Items {
		msg.Timestamp = item.Timestamp
		msg.setSignature(item.Signature)
		index := validators.IndexOf(msg.address())
		if index < 0 {
			return nil, errors.Errorf("bad voter %v at index %d in vote list", msg.address(), i)
		}
		if vset[index] {
			return nil, errors.Errorf("bvl.VerifyBlock: duplicated validator %v\n", msg.address())
		}
		vset[index] = true
	}
	if enoughVote(len(bvl.Items), validators.Len()) {
		return vset, nil
	}
	return nil, errors.Errorf("votes(%d) <= 2/3 of validators(%d)", len(bvl.Items), validators.Len())
}

func enoughVote(voted int, voters int) bool {
	if voters == 0 {
		return true
	}
	twoThirds := voters * 2 / 3
	return voted > twoThirds
}

func (bvl *blockCommitVoteList) String() string {
	return fmt.Sprintf("VoteList(R=%d,ID=%v,len(Signs)=%d)",
		bvl.Round, bvl.BlockPartSetIDAndNTSVoteCount, len(bvl.Items))
}

func (bvl *blockCommitVoteList) Timestamp() int64 {
	l := len(bvl.Items)
	if l == 0 {
		return 0
	}
	ts := make([]int64, l)
	for i := range ts {
		ts[i] = bvl.Items[i].Timestamp
	}
	sort.Slice(ts, func(i, j int) bool {
		return ts[i] < ts[j]
	})
	if l%2 == 1 {
		return ts[l/2]
	}
	return (ts[l/2-1] + ts[l/2]) / 2
}

func (bvl *blockCommitVoteList) VoteRound() int32 {
	return bvl.Round
}

type CommitVoteList struct {
	blockCommitVoteList
	NTSDProves [][]byte
}

func (vl *CommitVoteList) RLPEncodeSelf(e codec.Encoder) error {
	if len(vl.NTSDProves) == 0 {
		return e.EncodeListOf(vl.Round, vl.BlockPartSetIDAndNTSVoteCount, vl.Items)
	}
	return e.EncodeListOf(
		vl.Round,
		vl.BlockPartSetIDAndNTSVoteCount,
		vl.Items,
		vl.NTSDProves,
	)
}

func (vl *CommitVoteList) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	cnt, err := d2.DecodeMulti(
		&vl.Round,
		&vl.BlockPartSetIDAndNTSVoteCount,
		&vl.Items,
		&vl.NTSDProves,
	)
	if cnt == 3 && err == io.EOF {
		vl.NTSDProves = nil
		return nil
	}
	return err
}

func (vl *CommitVoteList) Bytes() []byte {
	if vl.bytes == nil {
		vl.bytes = vlCodec.MustMarshalToBytes(vl)
	}
	return vl.bytes
}

func (vl *CommitVoteList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

func (vl *CommitVoteList) NTSDProofCount() int {
	return len(vl.NTSDProves)
}

func (vl *CommitVoteList) NTSDProofAt(i int) []byte {
	return vl.NTSDProves[i]
}

func (vl *CommitVoteList) String() string {
	return fmt.Sprintf("VoteList(R=%d,ID=%v,len(Signs)=%d,len(NTS)=%d)",
		vl.Round, vl.BlockPartSetIDAndNTSVoteCount, len(vl.Items), len(vl.NTSDProves))
}

func (vl *CommitVoteList) toVoteListWithBlock(
	blk module.BlockData,
	prevBlk module.Block,
	dbase db.Database,
) (*VoteList, error) {
	ntsHashEntries, err := blk.NTSHashEntryList()
	if err != nil {
		return nil, err
	}
	if prevBlk == nil {
		return vl.toVoteList(
			blk.Height(), blk.ID(), nil, nil, ntsHashEntries, dbase,
		)
	}
	return vl.toVoteList(
		blk.Height(), blk.ID(), prevBlk.Result(), prevBlk.NextValidators(), ntsHashEntries, dbase,
	)
}

// toVoteList converts CommitVoteList to VoteList. result is the result in
// height-1 block. Note that there should be no NTS votes if prevResult is nil.
// validators is the nextValidators in height-1 block.
func (vl *CommitVoteList) toVoteList(
	height int64, bid []byte, prevResult []byte,
	validators module.ValidatorList,
	ntsHashEntries module.NTSHashEntryList, dbase db.Database,
) (*VoteList, error) {
	bCtx, err := service.NewBTPContext(dbase, prevResult)
	if err != nil {
		return nil, err
	}
	valLen := 0
	if validators != nil {
		valLen = validators.Len()
	}
	proofParts := make([] /*valIdx*/ [] /*ntsdProofIdx*/ []byte, valLen)
	ntsVoteBases := make([]ntsVoteBase, 0, ntsHashEntries.NTSHashEntryCount())
	ntsdProofIndex := 0
	for i := 0; i < ntsHashEntries.NTSHashEntryCount(); i++ {
		ntsHashEntry := ntsHashEntries.NTSHashEntryAt(i)
		nt, err := bCtx.GetNetworkType(ntsHashEntry.NetworkTypeID)
		if errors.Is(err, errors.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if nt.NextProofContext() == nil {
			continue
		}
		mod := ntm.ForUID(nt.UID())
		if len(vl.NTSDProves) <= ntsdProofIndex {
			return nil, errors.Errorf("NTS count mismatch len(NTDSProves)=%d NTSHashEntryCount=%d", len(vl.NTSDProves), ntsHashEntries.NTSHashEntryCount())
		}
		pf, err := mod.NewProofFromBytes(vl.NTSDProves[ntsdProofIndex])
		if err != nil {
			return nil, err
		}
		for v := 0; v < pf.ValidatorCount(); v++ {
			var bys []byte
			pp := pf.ProofPartAt(v)
			if pp != nil {
				bys = pp.Bytes()
				proofParts[v] = append(proofParts[v], bys)
			}
		}
		ntsVoteBases = append(ntsVoteBases, ntsVoteBase(ntsHashEntry))
		ntsdProofIndex++
	}
	rvl := NewVoteList()
	msg := newVoteMessage()
	msg.Height = height
	msg.Round = vl.Round
	msg.Type = VoteTypePrecommit
	msg.SetRoundDecision(bid, vl.BlockPartSetIDAndNTSVoteCount, ntsVoteBases)
	if len(vl.Items) > 0 && validators == nil {
		return nil, errors.Errorf("nil validators with voteListItems len(vl.Items)=%d", len(vl.Items))
	}
	for _, item := range vl.Items {
		msg.Timestamp = item.Timestamp
		msg.setSignature(item.Signature)
		vIdx := validators.IndexOf(msg.address())
		if vIdx < 0 {
			return nil, errors.Errorf("not a validator address=%s", msg.address().String())
		}
		msg.NTSDProofParts = proofParts[vIdx]
		rvl.AddVote(msg)
	}
	return rvl, nil
}

// newCommitVoteList returns a new CommitVoteList.
// pcm must be the pcm for height of the msgs. i.e. nextPCM in block height-1.
func newCommitVoteList(
	pcm module.BTPProofContextMap,
	msgs []*VoteMessage,
) (*CommitVoteList, error) {
	vl := &CommitVoteList{}
	l := len(msgs)
	if l == 0 {
		return vl, nil
	}
	vl.Round = msgs[0].Round
	vl.BlockPartSetIDAndNTSVoteCount = msgs[0].BlockPartSetIDAndNTSVoteCount
	vl.Items = make([]blockCommitVoteItem, l)
	rdd := msgs[0].RoundDecisionDigest()
	for i := 0; i < l; i++ {
		vl.Items[i] = blockCommitVoteItem{
			msgs[i].Timestamp,
			msgs[i].Signature,
		}
		if !bytes.Equal(rdd, msgs[i].RoundDecisionDigest()) {
			return nil, errors.Errorf(
				"NewVoteList: bad RDD in messages msgs[0].BlockID=%s msgs[i].BlockID=%s i=%d",
				common.HexPre(msgs[0].BlockID),
				common.HexPre(msgs[i].BlockID),
				i,
			)
		}
	}
	if pcm == nil {
		return vl, nil
	}
	for i := range msgs[0].NTSVoteBases {
		pc, err := pcm.ProofContextFor(msgs[0].NTSVoteBases[i].NetworkTypeID)
		if err != nil {
			return nil, err
		}
		pf := pc.NewProof()
		for _, msg := range msgs {
			pp, err := pc.NewProofPartFromBytes(msg.NTSDProofParts[i])
			if err != nil {
				return nil, err
			}
			pf.Add(pp)
		}
		vl.NTSDProves = append(vl.NTSDProves, pf.Bytes())
	}
	return vl, nil
}

func NewCommitVoteList(pcm module.BTPProofContextMap, msgs ...*VoteMessage) module.CommitVoteSet {
	cvl, _ := newCommitVoteList(pcm, msgs)
	return cvl
}

func NewEmptyCommitVoteList() module.CommitVoteSet {
	cvl, _ := newCommitVoteList(nil, nil)
	return cvl
}

// NewCommitVoteSetFromBytes returns VoteList from serialized bytes
func NewCommitVoteSetFromBytes(bs []byte) module.CommitVoteSet {
	vl := &CommitVoteList{}
	if bs == nil {
		return vl
	}
	_, err := vlCodec.UnmarshalFromBytes(bs, vl)
	if err != nil {
		return nil
	}
	return vl
}

func WALRecordBytesFromCommitVoteListBytes(
	bs []byte, h int64, bid []byte, result []byte,
	validators module.ValidatorList,
	ntsHashEntries module.NTSHashEntryList,
	dbase db.Database, c codec.Codec,
) ([]byte, error) {
	cvl := &CommitVoteList{}
	if bs != nil {
		_, err := c.UnmarshalFromBytes(bs, cvl)
		if err != nil {
			return nil, err
		}
	}
	vlm := newVoteListMessage()
	vl, err := cvl.toVoteList(h, bid, result, validators, ntsHashEntries, dbase)
	if err != nil {
		return nil, err
	}
	vlm.VoteList = vl
	rec := make([]byte, 2, 32)
	binary.BigEndian.PutUint16(rec, vlm.subprotocol())
	writer := bytes.NewBuffer(rec)
	if err := c.Marshal(writer, vlm); err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}
