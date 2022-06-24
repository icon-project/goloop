package consensus

import (
	"bytes"
	"io"
	"log"
	"testing"

	"github.com/icon-project/goloop/module"
)

type testBlock struct {
	module.Block
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

func TestBlockParts(t *testing.T) {
	blk := new(testBlock)
	psb := NewPartSetBuffer(1024)
	if err := blk.MarshalHeader(psb); err != nil {
		t.Errorf("Fail to marshal header err=%+v", err)
		return
	}
	if err := blk.MarshalBody(psb); err != nil {
		t.Errorf("Fail to marshal body err=%+v", err)
		return
	}
	ps := psb.PartSet(0)

	hdr := ps.ID()
	log.Printf("ID : %+v", hdr)
	log.Printf("Number of parts : %d", ps.Parts())

	parts := make([]Part, ps.Parts())
	for i := 0; i < len(parts); i++ {
		p := ps.GetPart(i)
		bs := p.Bytes()
		log.Printf("Part[%d] %d bytes\n", i, len(bs))
		if p2, err := NewPart(bs); err != nil {
			t.Errorf("Fail to parse part[%d]", i)
			return
		} else {
			parts[i] = p2
		}
	}

	ps2 := NewPartSetFromID(hdr)
	if ps2.IsComplete() {
		t.Error("Before adding parts, it's already completed")
	}

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
	if err := blk.MarshalHeader(buf1); err != nil {
		t.Errorf("Fail to marshal header for check err=%+v", err)
	}
	if err := blk.MarshalBody(buf1); err != nil {
		t.Errorf("Fail to marshal body for check err=%+v", err)
	}

	buf2 := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf2, ps2.NewReader()); err != nil {
		t.Errorf("Fail to io.Copy err=%+v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("Recovered bytes are not same")
	}
}
