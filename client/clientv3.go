package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
	"github.com/icon-project/goloop/service/transaction"
)

//jsonrpc client using echo
//commands reference server/v3/api_v3.go
//params reference server/v3/schema_v3.go
//response schema Block, ConfirmedTransaction, TransactionResult{EventLog, Failure}, ScoreApi,
type ClientV3 struct {
	*JsonRpcClient
	conns []*websocket.Conn
}

func NewClientV3(endpoint string) *ClientV3 {
	return &ClientV3{JsonRpcClient: NewJsonRpcClient(&http.Client{}, endpoint)}
}

//refer block/blockv2.go blockv2.ToJSON
type Block struct {
	BlockHash              jsonrpc.HexBytes    `json:"block_hash" validate:"required,t_hash"`
	Version                jsonrpc.HexInt      `json:"version" validate:"required,t_int"`
	Height                 int64               `json:"height" validate:"required,t_int"`
	Timestamp              int64               `json:"time_stamp" validate:"required,t_int"`
	Proposer               jsonrpc.HexBytes    `json:"peer_id" validate:"optional,t_addr_eoa"`
	PrevID                 jsonrpc.HexBytes    `json:"prev_block_hash" validate:"required,t_hash"`
	NormalTransactionsHash jsonrpc.HexBytes    `json:"merkle_tree_root_hash" validate:"required,t_hash"`
	Signature              jsonrpc.HexBytes    `json:"signature" validate:"optional,t_hash"`
	NormalTransations      []NormalTransaction `json:"confirmed_transaction_list" `
}

//refer service/transaction/transaction_v3.go:24 transactionV3Data, transactionV3.ToJSON
type NormalTransaction struct {
	TxHash    jsonrpc.HexBytes `json:"txHash"`
	Version   jsonrpc.HexInt   `json:"version"`
	From      jsonrpc.Address  `json:"from"`
	To        jsonrpc.Address  `json:"to"`
	Value     jsonrpc.HexInt   `json:"value,omitempty" `
	StepLimit jsonrpc.HexInt   `json:"stepLimit"`
	TimeStamp jsonrpc.HexInt   `json:"timestamp"`
	NID       jsonrpc.HexInt   `json:"nid,omitempty"`
	Nonce     jsonrpc.HexInt   `json:"nonce,omitempty"`
	Signature jsonrpc.HexBytes `json:"signature"`
	DataType  string           `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

//refer service/txresult/receipt.go:220 receiptJSON, receipt.ToJSON
//refer server/v3/api_v3.go:260 getTransactionResult
type TransactionResult struct {
	To                 jsonrpc.Address  `json:"to"`
	CumulativeStepUsed jsonrpc.HexInt   `json:"cumulativeStepUsed"`
	StepUsed           jsonrpc.HexInt   `json:"stepUsed"`
	StepPrice          jsonrpc.HexInt   `json:"stepPrice"`
	EventLogs          []EventLog       `json:"eventLogs"`
	LogsBloom          jsonrpc.HexBytes `json:"logsBloom"`
	Status             jsonrpc.HexInt   `json:"status"`
	Failure            *FailureReason   `json:"failure,omitempty"`
	SCOREAddress       jsonrpc.Address  `json:"scoreAddress,omitempty"`
	BlockHash          jsonrpc.HexBytes `json:"blockHash" validate:"required,t_hash"`
	BlockHeight        jsonrpc.HexInt   `json:"blockHeight" validate:"required,t_int"`
	TxIndex            jsonrpc.HexInt   `json:"txIndex" validate:"required,t_int"`
	TxHash             jsonrpc.HexBytes `json:"txHash" validate:"required,t_int"`
}

//refer service/txresult/receipt.go:29 eventLogJSON
type EventLog struct {
	Addr    jsonrpc.Address `json:"scoreAddress"`
	Indexed []string        `json:"indexed"`
	Data    []string        `json:"data"`
}

//refer service/txresult/receipt.go:193 failureReason
type FailureReason struct {
	CodeValue    jsonrpc.HexInt `json:"code"`
	MessageValue string         `json:"message"`
}

//refer server/v3/api_v3.go:307 getTransactionByHash
type Transaction struct {
	NormalTransaction
	BlockHash   jsonrpc.HexBytes `json:"blockHash" validate:"required,t_hash"`
	BlockHeight jsonrpc.HexInt   `json:"blockHeight" validate:"required,t_int"`
	TxIndex     jsonrpc.HexInt   `json:"txIndex" validate:"required,t_int"`
}

func (c *ClientV3) GetLastBlock() (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getLastBlock", nil, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) GetBlockByHeight(param *v3.BlockHeightParam) (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getBlockByHeight", param, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) GetBlockByHash(param *v3.BlockHashParam) (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getBlockByHash", param, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) Call(param *v3.CallParam) (interface{}, error) {
	var result interface{}
	_, err := c.Do("icx_call", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetBalance(param *v3.AddressParam) (*jsonrpc.HexInt, error) {
	var result jsonrpc.HexInt
	_, err := c.Do("icx_getBalance", param, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//refer servicce/scoreapi/info.go Info.ToJSON
func (c *ClientV3) GetScoreApi(param *v3.ScoreAddressParam) ([]interface{}, error) {
	var result []interface{}
	_, err := c.Do("icx_getScoreApi", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetTotalSupply() (*jsonrpc.HexInt, error) {
	var result jsonrpc.HexInt
	_, err := c.Do("icx_getTotalSupply", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClientV3) GetTransactionResult(param *v3.TransactionHashParam) (*TransactionResult, error) {
	tr := &TransactionResult{}
	_, err := c.Do("icx_getTransactionResult", param, tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *ClientV3) GetTransactionByHash(param *v3.TransactionHashParam) (*Transaction, error) {
	t := &Transaction{}
	_, err := c.Do("icx_getTransactionByHash", param, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

var txSerializeExcludes = map[string]bool{"signature": true}

func (c *ClientV3) SendTransaction(w module.Wallet, param *v3.TransactionParam) (*jsonrpc.HexBytes, error) {
	param.Timestamp = jsonrpc.HexInt(common.FormatInt(time.Now().UnixNano() / int64(time.Microsecond)))
	js, err := json.Marshal(param)
	if err != nil {
		return nil, err
	}

	bs, err := transaction.SerializeJSON(js, nil, txSerializeExcludes)
	if err != nil {
		return nil, err
	}
	bs = append([]byte("icx_sendTransaction."), bs...)
	sig, err := w.Sign(crypto.SHA3Sum256(bs))
	if err != nil {
		return nil, err
	}

	param.Signature = base64.StdEncoding.EncodeToString(sig)

	var result jsonrpc.HexBytes
	if _, err = c.Do("icx_sendTransaction", param, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

//using blockHeader.NextValidatorsHash
func (c *ClientV3) GetDataByHash(param *v3.DataHashParam) ([]byte, error) {
	var result []byte
	_, err := c.Do("icx_getDataByHash", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetBlockHeaderByHeight(param *v3.BlockHeightParam) ([]byte, error) {
	var result []byte
	_, err := c.Do("icx_getBlockHeaderByHeight", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetVotesByHeight(param *v3.BlockHeightParam) ([]byte, error) {
	var result []byte
	_, err := c.Do("icx_getVotesByHeight", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}


//refer common/trie/ompt/mtp.go mpt.GetProof(index)
func (c *ClientV3) GetProofForResult(param *v3.ProofResultParam) ([][]byte, error) {
	var result [][]byte
	_, err := c.Do("icx_getProofForResult", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) MonitorBlock(param *server.BlockRequest, cb func(v *server.BlockNotification), cancelCh <-chan bool) error {
	resp := &server.BlockNotification{}
	return c.Monitor("/block", param, resp, func(v interface{}) {
		if bn, ok := v.(*server.BlockNotification); ok {
			cb(bn)
		}
	}, cancelCh)
}

func (c *ClientV3) MonitorEvent(param *server.EventRequest, cb func(v *server.EventNotification), cancelCh <-chan bool) error {
	resp := &server.EventNotification{}
	return c.Monitor("/event", param, resp, func(v interface{}) {
		if en, ok := v.(*server.EventNotification); ok {
			cb(en)
		}
	}, cancelCh)
}

func (c *ClientV3) Monitor(reqUrl string, reqPtr, respPtr interface{},
	cb func(v interface{}), cancelCh <-chan bool) error {
	endpoint := strings.Replace(c.Endpoint, "http", "ws", 1)
	conn, _, err := WSConnect(endpoint+reqUrl, nil, reqPtr)
	if err != nil {
		return err
	}
	if cb != nil {
		WSReadJSONLoop(conn, respPtr, cb, cancelCh)
	}
	return nil
}

func WSReadJSONLoop(c *websocket.Conn, respPtr interface{}, cb func(v interface{}), cancelCh <-chan bool) {
	ch := make(chan interface{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			close(ch)
			wg.Done()
		}()
		for {
			if err := c.ReadJSON(respPtr); err != nil {
				cb(err)
				return
			}
			cb(respPtr)
		}
	}()
	if cancelCh != nil {
		go func() {
			defer c.Close()
			for {
				select {
				case <-cancelCh:
					return
				case <-ch:
					return
				}
			}
		}()
	} else {
		wg.Wait()
		_ = c.Close()
	}
}

func WSConnect(urlStr string, reqHeader http.Header, reqPtr interface{}) (c *websocket.Conn, wsResp *server.WSResponse, err error) {
	if reqPtr == nil {
		err = fmt.Errorf("reqPtr cannot be nil")
		return
	}
	var httpResp *http.Response
	c, httpResp, err = websocket.DefaultDialer.Dial(urlStr, reqHeader)
	if err != nil {
		fmt.Printf("Dial fail %+v", err)
		if httpResp != nil {
			defer httpResp.Body.Close()
			wsResp = &server.WSResponse{}
			if dErr := json.NewDecoder(httpResp.Body).Decode(wsResp); dErr != nil {
				wsResp = nil
				err = fmt.Errorf("resp:%v, err:%+v", httpResp, err)
				return
			}
		}
		return
	}
	if err = c.WriteJSON(reqPtr); err == nil {
		wsResp = &server.WSResponse{}
		if err = c.ReadJSON(wsResp); err == nil {
			if wsResp.Code == 0 {
				return
			}
			err = fmt.Errorf("receive WSResponse.Code is not zero")
		} else {
			fmt.Printf("ReadJSON err:%+v", err)
			wsResp = nil
		}
	}
	fmt.Printf("WriteJSON err:%+v", err)
	c.Close()
	return
}
