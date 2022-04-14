package service

const (
	ActiveNetworkTypeIDsKey = "activeNetworkTypeIDs"
	NetworkTypeByIDKey      = "networkTypeByID"
	NetworkByIDKey          = "networkByID"
)

type NetworkType struct {
	Name                 string
	NextProofContextHash []byte
	NextProofContext     []byte
	ConnectedNetworks    []int32
}

type Network struct {
	NetworkTypeID          int32
	LastMessagesRootNumber int64
	PrevNetworkSectionHash []byte
	LastNetworkSectionHash []byte
}
