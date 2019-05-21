package consensus

import (
	"github.com/icon-project/goloop/common"
)

type VoteItem struct {
	PrototypeIndex int16
	Timestamp      int64
	Signature      common.Signature
}

// TODO rename -> voteList
type voteList struct {
	Prototypes []voteBase
	VoteItems  []VoteItem
}

func (vl *voteList) AddVote(msg *voteMessage) {
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
	})
}

func (vl *voteList) Len() int {
	return len(vl.VoteItems)
}

func (vl *voteList) Get(i int) *voteMessage {
	msg := newVoteMessage()
	msg.voteBase = vl.Prototypes[vl.VoteItems[i].PrototypeIndex]
	msg.Timestamp = vl.VoteItems[i].Timestamp
	msg.setSignature(vl.VoteItems[i].Signature)
	return msg
}

func newVoteList() *voteList {
	return &voteList{}
}
