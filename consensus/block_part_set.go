package consensus

import (
	"bytes"
	"github.com/icon-project/goloop/module"
	"io"
)

type BlockPartsHeader struct {
	Count int32
	Hash  []byte
}

func (h *BlockPartsHeader) Equal(h2 *BlockPartsHeader) bool {
	if h == h2 {
		return true
	}
	if h == nil || h2 == nil {
		return false
	}
	return h.Count == h2.Count && bytes.Equal(h.Hash, h2.Hash)
}

type BlockParts interface {
	Header() *BlockPartsHeader
	Parts() int
	GetPart(int) BlockPart
	IsComplete() bool
	NewReader() io.Reader
	AddPart(BlockPart) (bool, error)
}

type blockParts struct {
	added int32
	parts []*blockPart
}

func (ps *blockParts) Header() *BlockPartsHeader {
	return &BlockPartsHeader{
		Count: int32(len(ps.parts)),
		Hash:  nil, // TODO get valid hash
	}
}

func (ps *blockParts) Parts() int {
	return len(ps.parts)
}

func (ps *blockParts) GetPart(i int) BlockPart {
	if i < 0 || i >= len(ps.parts) {
		return nil
	}
	// TODO need to implement all interfaces of parts.
	// return ps.parts[i]
	return nil
}

func (ps *blockParts) IsComplete() bool {
	panic("implement me")
}

func (ps *blockParts) NewReader() io.Reader {
	panic("implement me")
}

func (ps *blockParts) AddPart(BlockPart) (bool, error) {
	panic("implement me")
}

func newBlockPartsFromBlock(block module.Block) BlockParts {
	// TODO
	return nil
}

func newBlockPartsFromHeader(h *BlockPartsHeader) BlockParts {
	// TODO
	return nil
}

type BlockPart interface {
	Index() int32
	Bytes() []byte
}

type blockPart struct {
	idx  int
	data []byte
}

func (p *blockPart) Index() int {
	return p.idx
}

func (p *blockPart) Bytes() []byte {
	return p.data
}

func newBlockPart(b []byte) (BlockPart, error) {
	return nil, nil
}

type blockPartSet struct {
}

func (bps *blockPartSet) newReader() io.Reader {
	return nil
}

func (bps *blockPartSet) isComplete() bool {
	return false
}

// return true if added. if item is already received, false nil is returned.
func (bps *blockPartSet) add(index int, proof [][]byte) (bool, error) {
	return false, nil
}

func newBlockPartSet(nParts int) *blockPartSet {
	return nil
}
