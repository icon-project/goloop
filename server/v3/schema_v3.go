package v3

import (
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/server/jsonrpc"
)

const (
	Version = 3
)

var (
	VersionValue = jsonrpc.HexInt(intconv.FormatInt(Version))
)

type BlockHeightParam struct {
	Height jsonrpc.HexInt `json:"height" validate:"required,t_int"`
}

type HeightParam struct {
	Height jsonrpc.HexInt `json:"height,omitempty" validate:"optional,t_int"`
}

type BlockHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

type CallParam struct {
	FromAddress jsonrpc.Address `json:"from,omitempty" validate:"optional,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr_score"`
	DataType    string          `json:"dataType" validate:"required,call"`
	Data        interface{}     `json:"data"`
	Height      jsonrpc.HexInt  `json:"height,omitempty" validate:"optional,t_int"`
}

type AddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr"`
	Height  jsonrpc.HexInt  `json:"height,omitempty" validate:"optional,t_int"`
}

type ScoreAddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr_score"`
	Height  jsonrpc.HexInt  `json:"height,omitempty" validate:"optional,t_int"`
}

type TransactionHashParam struct {
	Hash jsonrpc.HexBytes `json:"txHash" validate:"required,t_hash"`
}

type TransactionParamForEstimate struct {
	Version     jsonrpc.HexInt  `json:"version" validate:"required,t_int"`
	FromAddress jsonrpc.Address `json:"from" validate:"required,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr"`
	Value       jsonrpc.HexInt  `json:"value,omitempty" validate:"optional,t_int"`
	Timestamp   jsonrpc.HexInt  `json:"timestamp" validate:"required,t_int"`
	NetworkID   jsonrpc.HexInt  `json:"nid" validate:"required,t_int"`
	Nonce       jsonrpc.HexInt  `json:"nonce,omitempty" validate:"optional,t_int"`
	DataType    string          `json:"dataType,omitempty" validate:"optional,call|deploy|message|deposit"`
	Data        interface{}     `json:"data,omitempty"`
}

type TransactionParam struct {
	Version     jsonrpc.HexInt  `json:"version" validate:"required,t_int"`
	FromAddress jsonrpc.Address `json:"from" validate:"required,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr"`
	Value       jsonrpc.HexInt  `json:"value,omitempty" validate:"optional,t_int"`
	StepLimit   jsonrpc.HexInt  `json:"stepLimit" validate:"required,t_int"`
	Timestamp   jsonrpc.HexInt  `json:"timestamp" validate:"required,t_int"`
	NetworkID   jsonrpc.HexInt  `json:"nid" validate:"required,t_int"`
	Nonce       jsonrpc.HexInt  `json:"nonce,omitempty" validate:"optional,t_int"`
	Signature   string          `json:"signature" validate:"required,t_sig"`
	DataType    string          `json:"dataType,omitempty" validate:"optional,call|deploy|message|deposit"`
	Data        interface{}     `json:"data,omitempty"`
}

type DataHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

type ProofResultParam struct {
	BlockHash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
	Index     jsonrpc.HexInt   `json:"index" validate:"required,t_int"`
}

type ProofEventsParam struct {
	BlockHash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
	Index     jsonrpc.HexInt   `json:"index" validate:"required,t_int"`
	Events    []jsonrpc.HexInt `json:"events" validate:"gt=0,dive,t_int"`
}

type RosettaTraceParam struct {
	Tx     jsonrpc.HexBytes `json:"tx,omitempty" validate:"optional,t_hash"`
	Block  jsonrpc.HexBytes `json:"block,omitempty" validate:"optional,t_hash"`
	Height jsonrpc.HexInt   `json:"height,omitempty" validate:"optional,gte=0,t_int"`
}
