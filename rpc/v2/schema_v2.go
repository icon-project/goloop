package v2

import (
	"log"
	"reflect"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/icon-project/goloop/module"
	"github.com/osamingo/jsonrpc"
)

// JSON RPC api v2
const jsonRpcV2 int = 2
const jsonRpcV3 int = 3

// JSON-RPC Request Params
type sendTransactionParam struct {
	FromAddress     string `json:"from" valid:"t_addr_eoa,required"`
	ToAddress       string `json:"to" valid:"t_addr_eoa,required"`
	Value           string `json:"value" valid:"t_int,required"`
	Fee             string `json:"fee" valid:"t_int,required"`
	Timestamp       string `json:"timestamp" valid:"int,required"`
	Nonce           string `json:"nonce" valid:"int,optional"`
	TransactionHash string `json:"tx_hash" valid:"t_hash_v2,required"`
	Signature       string `json:"signature" valid:"t_sig,required"`
}

type getTransactionResultParam struct {
	TransactionHash string `json:"tx_hash" valid:"t_hash_v2,required"`
}

type getBalanceParam struct {
	Address string `json:"address" valid:"t_addr,required"`
}

type getBlockByHashParam struct {
	BlockHash string `json:"hash" valid:"t_hash_v2,required"`
}

type getBlockByHeightParam struct {
	BlockHeight string `json:"height" valid:"int,required"`
}

// JSON-RPC Response Result
type sendTranscationResult struct {
	ResponseCode    int64  `json:"response_code"`
	TransactionHash string `json:"tx_hash,omitempty"`
	Message         string `json:"message,omitempty"`
}

type blockResult struct {
	ResponseCode int64   `json:"response_code"`
	Block        blockV2 `json:"block"`
}

type blockV2 struct {
	Version            string        `json:"version"`
	PrevBlockHash      string        `json:"prev_block_hash"`
	MerkleTreeRootHash string        `json:"merkle_tree_root_hash"`
	Timestamp          int64         `json:"time_stamp"`
	Transactions       []interface{} `json:"confirmed_transaction_list"`
	BlockHash          string        `json:"block_hash"`
	Height             int64         `json:"height"`
	PeerID             string        `json:"peer_id"`
	Signature          string        `json:"signature"`
}

type transactionV2 struct {
	FromAddress     string `json:"from"`
	ToAddress       string `json:"to"`
	Value           string `json:"value,omitempty"`
	Fee             string `json:"fee"`
	Timestamp       string `json:"timestamp"`
	TransactionHash string `json:"tx_hash"`
	Signature       string `json:"signature"`
	Method          string `json:"method"`
}

type transactionV3 struct {
	Version          string      `json:"version"`
	FromAddress      string      `json:"from"`
	ToAddress        string      `json:"to"`
	Value            string      `json:"value,omitempty"`
	StepLimit        string      `json:"stepLimit"`
	Timestamp        string      `json:"timestamp"`
	NetworkID        string      `json:"nid"`
	Nonce            string      `json:"nonce,omitempty"`
	TransactionHash  string      `json:"txHash"`
	TransactionIndex string      `json:"txIndex,omitempty"`
	Signature        string      `json:"signature"`
	DataType         string      `json:"dataType,omitempty"`
	Data             interface{} `json:"data,omitempty"`
}

type getTotalSupplyResult struct {
	ResponseCode int64  `json:"response_code"`
	Response     string `json:"response"`
}

type getBalanceResult struct {
	ResponseCode int64  `json:"response_code"`
	Response     string `json:"response"`
}

type getTransactionResultResult struct {
	ResponseCode string                       `json:"response_code"`
	Response     getTransactionResultResponse `json:"response,omitempty"`
	Message      string                       `json:"message,omitempty"`
}

type getTransactionResultResponse struct {
	Code int `json:"code"`
}

const (
	configValidateSendTransaction = false
)

// JSON-RPC Request Params Validator
func validateParam(s interface{}) *jsonrpc.Error {
	if configValidateSendTransaction {
		ok, err := govalidator.ValidateStruct(s)
		if !ok || err != nil {
			if err != nil {
				log.Printf("schema_v2.validateParam FAILs err=%+v", err)
			}
			return jsonrpc.ErrInvalidParams()
		}
	}
	return nil
}

func convertToResult(source interface{}, result interface{}, target reflect.Type) error {
	jsonMap := source.(map[string]interface{})
	// log.Printf("convert : [%s]", target.Name())

	v := reflect.ValueOf(result).Elem()
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)

		tag := field.Tag.Get("json")
		tag = strings.Split(tag, ",")[0]

		value := jsonMap[tag]
		vf := v.FieldByName(field.Name)
		switch vt := value.(type) {
		case string:
			//log.Printf("%s : %s", field.Name, vt)
			vf.SetString(vt)
		case int64:
			//log.Printf("%s : %d", field.Name, vt)
			vf.SetInt(value.(int64))
		}
	}
	return nil
}

func addConfirmedTxList(txList module.TransactionList, result *blockV2) error {

	for it := txList.Iterator(); it.Has(); it.Next() {
		tx, _, _ := it.Get()
		var txMap interface{}

		tx2 := transactionV2{}
		tx3 := transactionV3{}

		var err error
		//log.Printf("tx version (%d)", tx.Version())
		switch tx.Version() {
		case jsonRpcV2:
			txMap, err = tx.ToJSON(jsonRpcV2)
			if err != nil {
				log.Println(err.Error())
			}
			convertToResult(txMap, &tx2, reflect.TypeOf(tx2))
			result.Transactions = append(result.Transactions, tx2)
		case jsonRpcV3:
			txMap, err = tx.ToJSON(jsonRpcV3)
			if err != nil {
				log.Println(err.Error())
			}
			convertToResult(txMap, &tx3, reflect.TypeOf(tx3))
			result.Transactions = append(result.Transactions, tx3)
		}
	}
	return nil
}
