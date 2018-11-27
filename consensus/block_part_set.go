package consensus

import "io"

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
