package consensus

import (
	"bytes"
	"github.com/icon-project/goloop/module"
	"io"
	"log"
	"testing"
	"time"
)

type testBlock struct {
}

func (*testBlock) Version() int {
	panic("implement me")
}

func (*testBlock) ID() []byte {
	panic("implement me")
}

func (*testBlock) Height() int64 {
	panic("implement me")
}

func (*testBlock) PrevID() []byte {
	panic("implement me")
}

func (*testBlock) NextValidators() module.ValidatorList {
	panic("implement me")
}

func (*testBlock) Verify() error {
	panic("implement me")
}

func (*testBlock) Votes() module.VoteList {
	panic("implement me")
}

func (*testBlock) NormalTransactions() module.TransactionList {
	panic("implement me")
}

func (*testBlock) PatchTransactions() module.TransactionList {
	panic("implement me")
}

func (*testBlock) Timestamp() time.Time {
	panic("implement me")
}

func (*testBlock) Proposer() module.Address {
	panic("implement me")
}

func (*testBlock) LogBloom() []byte {
	panic("implement me")
}

func (*testBlock) Result() []byte {
	panic("implement me")
}

func (*testBlock) MarshalHeader(w io.Writer) error {
	if _, err := w.Write(bytes.Repeat([]byte("TestHeader"), 1000)); err != nil {
		return err
	}
	return nil
}

func (*testBlock) MarshalBody(w io.Writer) error {
	if _, err := w.Write(bytes.Repeat([]byte("TestBody"), 4000)); err != nil {
		return err
	}
	return nil
}

func (*testBlock) ToJSON(rcpVersion int) (interface{}, error) {
	panic("implement me")
}

func TestBlockParts(t *testing.T) {
	blk := new(testBlock)
	psb := newPartSetBuffer()
	if err := blk.MarshalHeader(psb); err != nil {
		t.Errorf("Fail to marshal header err=%+v", err)
		return
	}
	if err := blk.MarshalBody(psb); err != nil {
		t.Errorf("Fail to marshal body err=%+v", err)
		return
	}
	ps := psb.PartSet()

	hdr := ps.ID()
	log.Printf("ID : %+v", hdr)
	log.Printf("Number of parts : %d", ps.Parts())

	parts := make([]Part, ps.Parts())
	for i := 0; i < len(parts); i++ {
		parts[i] = ps.GetPart(i)
	}

	ps2 := newPartSetFromID(hdr)
	for i := 0; i < len(parts); i++ {
		if err := ps2.AddPart(parts[i]); err != nil {
			t.Errorf("Fail to add part(%d) err=%+v", i, err)
			return
		}
	}

	if !ps2.IsComplete() {
		t.Error("After adding all part it's not completed")
	}
	buf1 := bytes.NewBuffer(nil)
	blk.MarshalHeader(buf1)
	blk.MarshalBody(buf1)

	buf2 := bytes.NewBuffer(nil)
	io.Copy(buf2, ps2.NewReader())

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("Recovered bytes are not same")
	}
}
