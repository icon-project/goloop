package consensus

import "github.com/icon-project/goloop/module"

type voteList struct {
}

func (vl *voteList) Verify(block module.Block) error {
	panic("not implemented")
}

func (vl *voteList) Bytes() []byte {
	panic("not implemented")
}

func (vl *voteList) Hash() []byte {
	panic("not implemented")
}
