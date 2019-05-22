package test

//go:generate -command goimpl go run github.com/icon-project/goloop/cmd/goimpl

//go:generate goimpl blockbase.go test BlockBase module.Block
//go:generate goimpl blockmanagerbase.go test BlockManagerBase module.BlockManager
//go:generate goimpl chainbase.go test ChainBase module.Chain
//go:generate goimpl regulatorbase.go test RegulatorBase module.Regulator
//go:generate goimpl networkmanagerbase.go test NetworkManagerBase module.NetworkManager
//go:generate goimpl servicemanagerbase.go test ServiceManagerBase module.ServiceManager
