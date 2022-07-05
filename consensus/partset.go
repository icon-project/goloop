package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
)

type Part interface {
	Index() int
	Bytes() []byte
}

type PartSet interface {
	ID() *PartSetID
	Parts() int
	GetPart(int) Part
	IsComplete() bool
	NewReader() io.Reader
	AddPart(Part) error
	GetMask() *bitArray
	AppData() uint16
}

type PartSetBuffer interface {
	io.Writer
	PartSet(appData uint16) PartSet
}

const (
	countWidth = 16
	countMask  = (1 << countWidth) - 1
)

type PartSetID struct {
	// CountWord: MSB AppData(16) Count(16)
	// Use bitfield not to break existing message protocol
	CountWord uint32
	Hash      []byte
}

func (id *PartSetID) Count() uint16 {
	return uint16(id.CountWord) & countMask
}

func (id *PartSetID) AppData() uint16 {
	return uint16(id.CountWord >> countWidth)
}

func (id *PartSetID) Equal(id2 *PartSetID) bool {
	if id == id2 {
		return true
	}
	if id == nil || id2 == nil {
		return false
	}
	return id.CountWord == id2.CountWord && bytes.Equal(id.Hash, id2.Hash)
}

func (id PartSetID) String() string {
	return fmt.Sprintf("{CountWord:%x,Hash:%v}", id.CountWord, common.HexPre(id.Hash))
}

// TODO need to prepare proofs for each parts.
type partSet struct {
	added   int
	parts   []*part
	tree    trie.Immutable
	ba      *bitArray
	appData uint16
}

func (ps *partSet) ID() *PartSetID {
	return &PartSetID{
		CountWord: (uint32(ps.appData) << countWidth) | uint32(len(ps.parts)),
		Hash:      ps.Hash(),
	}
}

func (ps *partSet) Hash() []byte {
	if ps.tree != nil {
		return ps.tree.Hash()
	}
	return nil
}

func (ps *partSet) Parts() int {
	return len(ps.parts)
}

func (ps *partSet) GetPart(i int) Part {
	if i < 0 || i >= len(ps.parts) {
		return nil
	}
	p := ps.parts[i]
	if p == nil {
		// return nil interface not interface to nil value
		return nil
	}
	return p
}

func (ps *partSet) IsComplete() bool {
	return ps.added == len(ps.parts)
}

func (ps *partSet) GetMask() *bitArray {
	if ps == nil {
		return &bitArray{0, nil}
	}
	return ps.ba
}

func (ps *partSet) AppData() uint16 {
	return ps.appData
}

type blockPartsReader struct {
	ps          *partSet
	idx, offset int
}

func (r *blockPartsReader) Read(p []byte) (n int, err error) {
	nbs := 0
	for nbs < len(p) && r.idx < len(r.ps.parts) {
		part := r.ps.parts[r.idx]
		read := copy(p[nbs:], part.data[r.offset:])
		r.offset += read
		nbs += read
		if r.offset >= len(part.data) {
			r.idx += 1
			r.offset = 0
		}
	}
	if nbs == 0 {
		return 0, io.EOF
	}
	return nbs, nil
}

func (ps *partSet) NewReader() io.Reader {
	return &blockPartsReader{ps: ps, idx: 0, offset: 0}
}

func (ps *partSet) AddPart(p Part) error {
	pt, ok := p.(*part)
	if !ok {
		return errors.New("InvalidPartObj")
	}
	idx := p.Index()
	if idx < 0 || idx >= len(ps.parts) {
		return errors.New("InvalidIndexValue")
	}
	if ps.parts[idx] != nil {
		return errors.New("AlreadyAdded")
	}
	key, _ := codec.MarshalToBytes(uint16(pt.idx))
	data, err := ps.tree.Prove(key, pt.proof)
	if err != nil {
		return err
	}
	pt.data = data
	ps.parts[idx] = pt
	ps.added += 1
	ps.ba.Set(idx)
	return nil
}

type partSetBuffer struct {
	ps     *partSet
	part   *part
	offset int
	size   int
}

func (b *partSetBuffer) Write(p []byte) (n int, err error) {
	written := 0
	for written < len(p) {
		if b.part == nil {
			b.part = &part{
				idx:  len(b.ps.parts),
				data: make([]byte, b.size),
			}
			binary.BigEndian.PutUint16(b.part.data, uint16(b.part.idx))
		}
		n := copy(b.part.data[b.offset:], p[written:])

		b.offset += n
		written += n
		if b.offset == b.size {
			b.ps.parts = append(b.ps.parts, b.part)
			b.ps.added += 1
			b.offset = 0
			b.part = nil
		}
	}
	return written, nil
}

func (b *partSetBuffer) PartSet(appData uint16) PartSet {
	if b.part != nil {
		b.part.data = b.part.data[0:b.offset]
		b.ps.parts = append(b.ps.parts, b.part)
		b.ps.added += 1
		b.part = nil

		mt := trie_manager.NewMutable(db.NewNullDB(), nil)
		for i, p := range b.ps.parts {
			key, _ := codec.MarshalToBytes(uint16(i))
			_, _ = mt.Set(key, p.data)
		}
		ss := mt.GetSnapshot()
		for i, p := range b.ps.parts {
			key, _ := codec.MarshalToBytes(uint16(i))
			p.proof = ss.GetProof(key)
			if p.proof == nil {
				return nil
			}
		}
		b.ps.tree = ss
	}
	b.ps.ba = newBitArray(b.ps.added)
	b.ps.ba.Flip()
	b.ps.appData = appData
	return b.ps
}

func NewPartSetBuffer(sz int) PartSetBuffer {
	return &partSetBuffer{ps: new(partSet), size: sz}
}

func NewPartSetFromID(h *PartSetID) PartSet {
	return &partSet{
		parts:   make([]*part, h.Count()),
		tree:    trie_manager.NewImmutable(db.NewNullDB(), h.Hash),
		ba:      newBitArray(int(h.Count())),
		appData: h.AppData(),
	}
}

type partBinary struct {
	Index uint16
	Proof [][]byte
}

type part struct {
	idx   int
	proof [][]byte
	data  []byte
}

func (p *part) Index() int {
	return p.idx
}

func (p *part) Bytes() []byte {
	pb := partBinary{
		Index: uint16(p.idx),
		Proof: p.proof,
	}
	if bs, err := codec.MarshalToBytes(&pb); err != nil {
		log.Panicf("Fail to marshal partBinary err=%+v", err)
		return nil
	} else {
		return bs
	}
}

func NewPart(b []byte) (Part, error) {
	var pb partBinary
	if _, err := codec.UnmarshalFromBytes(b, &pb); err != nil {
		return nil, err
	}
	return &part{
		idx:   int(pb.Index),
		proof: pb.Proof,
	}, nil
}
