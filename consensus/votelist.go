package consensus

import (
	"fmt"

	"github.com/icon-project/goloop/common"
)

type VoteItem struct {
	PrototypeIndex int16
	Timestamp      int64
	Signature      common.Signature
	NTSDProofParts [][]byte
}

type VoteList struct {
	Prototypes []voteBase
	VoteItems  []VoteItem
}

func (vl VoteList) String() string {
	res := fmt.Sprintf("{Prototypes:%+v,VoteItems:[", vl.Prototypes)
	for i, vi := range vl.VoteItems {
		msg := vl.Get(i)
		if i > 0 {
			res += " "
		}
		res += fmt.Sprintf("{I:%v,T:%d,Addr:%v}", vi.PrototypeIndex, vi.Timestamp, common.HexPre(msg.address().ID()))
	}
	res += "]}"
	return res
}

func (vl *VoteList) AddVote(msg *VoteMessage) {
	index := -1
	for i, p := range vl.Prototypes {
		if p.Equal(&msg.voteBase) {
			index = i
			break
		}
	}
	if index == -1 {
		vl.Prototypes = append(vl.Prototypes, msg.voteBase)
		index = len(vl.Prototypes) - 1
	}
	vl.VoteItems = append(vl.VoteItems, VoteItem{
		PrototypeIndex: int16(index),
		Timestamp:      msg.Timestamp,
		Signature:      msg.Signature,
		NTSDProofParts: msg.NTSDProofParts,
	})
}

func (vl *VoteList) Len() int {
	return len(vl.VoteItems)
}

func (vl *VoteList) Get(i int) *VoteMessage {
	msg := newVoteMessage()
	proto := &vl.Prototypes[vl.VoteItems[i].PrototypeIndex]
	msg.voteBase = *proto
	msg.Timestamp = vl.VoteItems[i].Timestamp
	msg.setSignature(vl.VoteItems[i].Signature)
	msg.NTSDProofParts = vl.VoteItems[i].NTSDProofParts
	return msg
}

func NewVoteList() *VoteList {
	return &VoteList{}
}
