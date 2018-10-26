package db

//	Database IDs.
const (
	// StateTrie maps account from sha3(address)
	StateTrie = ""
	// StorageTrie maps storage value from storage key
	StorageTrie = ""
	// NormalTransactionTrie maps transaction from index
	NormalTransactionTrie = ""
	// PatchTransactionTrie maps transaction from index
	PatchTranscationTrie = ""
	// NormalReceiptTrie maps receipt from index
	NormalReceiptTrie = ""
	// PatchReceiptTrie maps receipt from index
	PatchReceiptTrie = ""
	// BlockHeaderByHash maps block header from hash of encoded block header.
	BlockHeaderByHash = "S"
	// SignaturesByHash maps signature array from hash of encoded signature
	// array.
	SignaturesByHash = "S"
	// ValidatorsByHash maps validator array from hash of encoded validator
	// array.
	ValidatorsByHash = "S"
	// CodeByHash maps code from hash of code
	CodeByHash = "S"
	// TransactionLocatorByHash maps transaction locator from transaction hash.
	TransactionLocatorByHash = "T"
	// BlockHeaderHashByHeight maps hash of encoded block header from height.
	BlockHeaderHashByHeight = "H"
	// BlockV1ByHash maps block V1 from block V1 hash.
	BlockV1ByHash = "B"
	// BlockV1ByHash maps receipt V1 from tx V3 hash.
	ReceiptV1ByHash = "R"
	// ChainProperty is general key value map for chain property.
	ChainProperty = "C"
)

type Store interface {
	GetDB(id string) (DB, error)
}

func GetStore(name string) (Store, error) {
	return nil, nil
}
