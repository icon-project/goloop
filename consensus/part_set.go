package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
	"io"
)

const (
	FragmentBytes  int = 10 * 1024
	PartIndexBytes     = 2
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
}

type PartSetBuffer interface {
	io.Writer
	PartSet() PartSet
}

type PartSetID struct {
	Count int32
	Hash  []byte
}

func (id *PartSetID) Equal(id2 *PartSetID) bool {
	if id == id2 {
		return true
	}
	if id == nil || id2 == nil {
		return false
	}
	return id.Count == id2.Count && bytes.Equal(id.Hash, id2.Hash)
}

func (id *PartSetID) String() string {
	return fmt.Sprintf("PartSet(parts=%d,hash=%x)", id.Count, id.Hash)
}

// TODO need to prepare proofs for each parts.
type partSet struct {
	added int
	parts []*part
	hash  []byte
}

func (ps *partSet) ID() *PartSetID {
	if !ps.IsComplete() {
		return nil
	}
	return &PartSetID{
		Count: int32(len(ps.parts)),
		Hash:  ps.Hash(),
	}
}

func (ps *partSet) Hash() []byte {
	if !ps.IsComplete() {
		return nil
	}
	if ps.hash == nil {
		sha := sha3.New256()
		for _, p := range ps.parts {
			sha.Write(p.data)
		}
		hash := sha.Sum([]byte{})[:]
		ps.hash = hash[:]
	}
	return ps.hash
}

func (ps *partSet) Parts() int {
	return len(ps.parts)
}

func (ps *partSet) GetPart(i int) Part {
	if i < 0 || i >= len(ps.parts) {
		return nil
	}
	return ps.parts[i]
}

func (ps *partSet) IsComplete() bool {
	return ps.added == len(ps.parts)
}

type blockPartsReader struct {
	ps          *partSet
	idx, offset int
}

func (r *blockPartsReader) Read(p []byte) (n int, err error) {
	nbs := 0
	for nbs < len(p) && r.idx < len(r.ps.parts) {
		part := r.ps.parts[r.idx]
		read := copy(p[nbs:], part.data[r.offset+PartIndexBytes:])
		r.offset += read
		nbs += read
		if (r.offset + PartIndexBytes) >= len(part.data) {
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

// TODO need prove with the part.
func (ps *partSet) AddPart(p Part) error {
	idx := p.Index()
	if idx < 0 || idx >= len(ps.parts) {
		return errors.New("InvalidIndexValue")
	}
	if ps.parts[idx] != nil {
		return errors.New("AlreadyAdded")
	}
	ps.parts[idx] = p.(*part)
	ps.added += 1
	return nil
}

type partSetBuffer struct {
	ps     *partSet
	part   *part
	offset int
}

func (w *partSetBuffer) Write(p []byte) (n int, err error) {
	written := 0
	for written < len(p) {
		if w.part == nil {
			w.part = &part{
				idx:  len(w.ps.parts),
				data: make([]byte, FragmentBytes+PartIndexBytes),
			}
			binary.BigEndian.PutUint16(w.part.data, uint16(w.part.idx))
		}
		n := copy(w.part.data[PartIndexBytes+w.offset:], p[written:])

		w.offset += n
		written += n
		if w.offset == FragmentBytes {
			w.ps.parts = append(w.ps.parts, w.part)
			w.ps.added += 1
			w.offset = 0
			w.part = nil
		}
	}
	return written, nil
}

func (w *partSetBuffer) PartSet() PartSet {
	if w.part != nil {
		w.part.data = w.part.data[0 : PartIndexBytes+w.offset]
		w.ps.parts = append(w.ps.parts, w.part)
		w.ps.added += 1
		w.part = nil
	}
	return w.ps
}

func newPartSetBuffer() PartSetBuffer {
	return &partSetBuffer{ps: new(partSet)}
}

func newPartSetFromID(h *PartSetID) PartSet {
	return &partSet{
		parts: make([]*part, h.Count),
		hash:  h.Hash,
	}
}

// TODO need to add proof
type part struct {
	idx  int
	data []byte
}

func (p *part) Index() int {
	return p.idx
}

func (p *part) Bytes() []byte {
	return p.data
}

func newPart(b []byte) (Part, error) {
	if len(b) < 2 {
		return nil, errors.New("TooShortPartBytes")
	}
	return &part{
		idx:  int(binary.BigEndian.Uint16(b[:PartIndexBytes])),
		data: b,
	}, nil
}
