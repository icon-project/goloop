package module

const (
	PatchTypeSkipTransaction = "skip_txs"
)

type Patch interface {
	Type() string
	Data() []byte
}

type SkipTransactionPatch interface {
	Patch
	Height() int64 // height of the block to skip execution of

	// Verify check internal data is correct
	Verify(vl ValidatorList, roundLimit int64, nid int) error
}

type PatchDecoder func(t string, bs []byte) (Patch, error)
