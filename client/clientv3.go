package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
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
	DebugEndPoint string
	conns         map[string]*websocket.Conn
}

func guessDebugEndpoint(endpoint string) string {
	uo, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	ps := strings.Split(uo.Path, "/")
	for i, v := range ps {
		if v == "api" {
			if len(ps) > i+1 && ps[i+1] == "v3" {
				ps[i+1] = "v3d"
				uo.Path = strings.Join(ps, "/")
				return uo.String()
			}
			break
		}
	}
	return ""
}

func NewClientV3(endpoint string) *ClientV3 {
	client := new(http.Client)
	apiClient := NewJsonRpcClient(client, endpoint)

	return &ClientV3{
		JsonRpcClient: apiClient,
		DebugEndPoint: guessDebugEndpoint(endpoint),
		conns:         make(map[string]*websocket.Conn),
	}
}

//refer block/blockv2.go blockv2.ToJSON
type Block struct {
	BlockHash              jsonrpc.HexBytes  `json:"block_hash" validate:"required,t_hash"`
	Version                jsonrpc.HexInt    `json:"version" validate:"required,t_int"`
	Height                 int64             `json:"height" validate:"required,t_int"`
	Timestamp              int64             `json:"time_stamp" validate:"required,t_int"`
	Proposer               jsonrpc.HexBytes  `json:"peer_id" validate:"optional,t_addr_eoa"`
	PrevID                 jsonrpc.HexBytes  `json:"prev_block_hash" validate:"required,t_hash"`
	NormalTransactionsHash jsonrpc.HexBytes  `json:"merkle_tree_root_hash" validate:"required,t_hash"`
	Signature              jsonrpc.HexBytes  `json:"signature" validate:"optional,t_hash"`
	NormalTransactions     []json.RawMessage `json:"confirmed_transaction_list" `
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
	StepDetails        interface{}      `json:"stepUsedDetails,omitempty"`
}

//refer service/txresult/receipt.go:29 eventLogJSON
type EventLog struct {
	Addr    jsonrpc.Address `json:"scoreAddress"`
	Indexed []*string       `json:"indexed"`
	Data    []*string       `json:"data"`
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

type NetworkInfo struct {
	Platform  string         `json:"platform"`
	NID       jsonrpc.HexInt `json:"nid"`
	Channel   string         `json:"channel"`
	Earliest  jsonrpc.HexInt `json:"earliest"`
	Latest    jsonrpc.HexInt `json:"latest"`
	StepPrice jsonrpc.HexInt `json:"stepPrice"`
}

//refer service/state/btp.go:887 network.ToJSON
//refer server/v3/api_v3.go:692 getBTPNetworkInfo
type BTPNetworkInfo struct {
	StartHeight             jsonrpc.HexInt   `json:"startHeight"`
	NetworkTypeID           jsonrpc.HexInt   `json:"networkTypeID"`
	NetworkName             string           `json:"networkName"`
	Open                    jsonrpc.HexInt   `json:"open"`
	Owner                   jsonrpc.Address  `json:"owner"`
	NextMessageSN           jsonrpc.HexInt   `json:"nextMessageSN"`
	NextProofContextChanged jsonrpc.HexInt   `json:"nextProofContextChanged"`
	PrevNSHash              jsonrpc.HexBytes `json:"prevNSHash"`
	LastNSHash              jsonrpc.HexBytes `json:"lastNSHash"`
	NetworkID               jsonrpc.HexInt   `json:"networkID"`
	NetworkTypeName         string           `json:"networkTypeName"`
}

//refer service/state/btp.go:752 networkType.ToJSON
//refer server/v3/api_v3.go:743 getBTPNetworkTypeInfo
type BTPNetworkTypeInfo struct {
	NetworkTypeName  string           `json:"networkTypeName"`
	NextProofContext jsonrpc.HexBytes `json:"nextProofContext"`
	OpenNetworkIDs   []jsonrpc.HexInt `json:"openNetworkIDs"`
	NetworkTypeID    jsonrpc.HexInt   `json:"networkTypeID"`
}

//refer server/v3/api_v3.go:953 getBTPSourceInformation
type BTPSourceInformation struct {
	SrcNetworkUID  string           `json:"srcNetworkUID"`
	NetworkTypeIDs []jsonrpc.HexInt `json:"networkTypeIDs"`
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

func (c *ClientV3) GetTotalSupply(param *v3.HeightParam) (*jsonrpc.HexInt, error) {
	var result jsonrpc.HexInt
	var nullableParam interface{}
	if param != nil {
		nullableParam = param
	}
	_, err := c.Do("icx_getTotalSupply", nullableParam, &result)
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

func (c *ClientV3) WaitTransactionResult(param *v3.TransactionHashParam) (*TransactionResult, error) {
	tr := &TransactionResult{}
	if _, err := c.Do("icx_waitTransactionResult", param, tr); err != nil {
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
	param.Timestamp = jsonrpc.HexInt(intconv.FormatInt(time.Now().UnixNano() / int64(time.Microsecond)))
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

func (c *ClientV3) SendRawTransaction(w module.Wallet, param map[string]interface{}) (*jsonrpc.HexBytes, error) {
	param["timestamp"] = intconv.FormatInt(time.Now().UnixNano() / int64(time.Microsecond))
	bs, err := transaction.SerializeMap(param, nil, txSerializeExcludes)
	if err != nil {
		return nil, err
	}
	bs = append([]byte("icx_sendTransaction."), bs...)
	sig, err := w.Sign(crypto.SHA3Sum256(bs))
	if err != nil {
		return nil, err
	}

	param["signature"] = base64.StdEncoding.EncodeToString(sig)
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

func (c *ClientV3) GetProofForEvents(param *v3.ProofEventsParam) ([][][]byte, error) {
	var result [][][]byte
	_, err := c.Do("icx_getProofForEvents", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetBTPNetworkInfo(param *v3.BTPQueryParam) (*BTPNetworkInfo, error) {
	ni := &BTPNetworkInfo{}
	if _, err := c.Do("btp_getNetworkInfo", param, ni); err != nil {
		return nil, err
	}
	return ni, nil
}

func (c *ClientV3) GetBTPNetworkTypeInfo(param *v3.BTPQueryParam) (*BTPNetworkTypeInfo, error) {
	nti := &BTPNetworkTypeInfo{}
	if _, err := c.Do("btp_getNetworkTypeInfo", param, nti); err != nil {
		return nil, err
	}
	return nti, nil
}

func (c *ClientV3) GetBTPMessages(param *v3.BTPMessagesParam) ([]string, error) {
	var msgs []string
	if _, err := c.Do("btp_getMessages", param, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (c *ClientV3) GetBTPHeader(param *v3.BTPMessagesParam) (string, error) {
	//refer block/btpblock.go:90 HeaderBytes, btpBlockHeaderFormat
	//refer server/v3/api_v3.go:865 getBTPHeader
	var s string
	if _, err := c.Do("btp_getHeader", param, &s); err != nil {
		return "", err
	}
	return s, nil
}

func (c *ClientV3) GetBTPProof(param *v3.BTPMessagesParam) (string, error) {
	//refer btp/ntm/secp256k1proof.go:55 secp256k1Proof
	//refer server/v3/api_v3.go:909 getBTPProof
	var s string
	if _, err := c.Do("btp_getProof", param, &s); err != nil {
		return "", err
	}
	return s, nil
}

func (c *ClientV3) GetBTPSourceInformation() (*BTPSourceInformation, error) {
	si := &BTPSourceInformation{}
	if _, err := c.Do("btp_getSourceInformation", nil, si); err != nil {
		return nil, err
	}
	return si, nil
}

func (c *ClientV3) GetScoreStatus(param *v3.ScoreAddressParam) (interface{}, error) {
	var result interface{}
	_, err := c.Do("icx_getScoreStatus", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetNetworkInfo() (*NetworkInfo, error) {
	var result *NetworkInfo
	_, err := c.Do("icx_getNetworkInfo", nil, &result)
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

func (c *ClientV3) MonitorBtp(param *server.BTPRequest, cb func(v *server.BTPNotification), cancelCh <-chan bool) error {
	resp := &server.BTPNotification{}
	return c.Monitor("/btp", param, resp, func(v interface{}) {
		if en, ok := v.(*server.BTPNotification); ok {
			cb(en)
		}
	}, cancelCh)
}

func (c *ClientV3) Monitor(reqUrl string, reqPtr, respPtr interface{},
	cb func(v interface{}), cancelCh <-chan bool) error {
	if cb == nil {
		return fmt.Errorf("callback function cannot be nil")
	}
	conn, _, err := c.wsConnect(reqUrl, nil, reqPtr)
	if err != nil {
		return err
	}
	if cancelCh != nil {
		ch := make(chan interface{})
		go func() {
			defer c.wsClose(conn)
			for {
				select {
				case <-cancelCh:
					return
				case <-ch:
					return
				}
			}
		}()
		go func() {
			defer func() {
				close(ch)
			}()
			c.wsReadJSONLoop(conn, respPtr, cb)
		}()
	} else {
		defer c.wsClose(conn)
		c.wsReadJSONLoop(conn, respPtr, cb)
	}
	return nil
}

func (c *ClientV3) Cleanup() {
	for _, conn := range c.conns {
		c.wsClose(conn)
	}
}

type wsConnectError struct {
	error
	httpErr error
}

func (we *wsConnectError) Error() string {
	return fmt.Sprintf("%+v http:%v", we.error, we.httpErr)
}

func (c *ClientV3) wsConnect(reqUrl string, reqHeader http.Header, reqPtr interface{}) (*websocket.Conn, *server.WSResponse, error) {
	if reqPtr == nil {
		return nil, nil, fmt.Errorf("reqPtr cannot be nil")
	}
	wsResp := &server.WSResponse{}
	wsEndpoint := strings.Replace(c.Endpoint, "http", "ws", 1)
	conn, httpResp, err := websocket.DefaultDialer.Dial(wsEndpoint+reqUrl, reqHeader)
	if err != nil {
		return nil, nil, &wsConnectError{error: err, httpErr: NewHttpError(httpResp)}
	}

	if err = conn.WriteJSON(reqPtr); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("fail to WriteJSON err:%+v", err)
	}

	if err = conn.ReadJSON(wsResp); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("fail to ReadJSON err:%+v", err)
	}

	if wsResp.Code != 0 {
		conn.Close()
		return nil, wsResp, fmt.Errorf("invalid WSResponse code:%d, message:%s", wsResp.Code, wsResp.Message)
	}
	la := conn.LocalAddr().String()
	c.conns[la] = conn
	return conn, wsResp, nil
}

func (c *ClientV3) wsClose(conn *websocket.Conn) {
	la := conn.LocalAddr().String()
	_, ok := c.conns[la]
	if ok {
		delete(c.conns, la)
	}
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()
}

func (c *ClientV3) wsReadJSONLoop(conn *websocket.Conn, respPtr interface{}, cb func(v interface{})) {
	elem := reflect.ValueOf(respPtr).Elem()
	for {
		v := reflect.New(elem.Type())
		ptr := v.Interface()
		if err := conn.ReadJSON(ptr); err != nil {
			cb(err)
			return
		}
		cb(ptr)
	}
}

func (c *ClientV3) EstimateStep(param *v3.TransactionParamForEstimate) (*common.HexInt, error) {
	if len(c.DebugEndPoint) == 0 {
		return nil, errors.InvalidStateError.New("UnavailableDebugEndPoint")
	}
	param.Timestamp = jsonrpc.HexInt(intconv.FormatInt(time.Now().UnixNano() / int64(time.Microsecond)))
	var result common.HexInt
	if _, err := c.DoURL(c.DebugEndPoint,
		"debug_estimateStep", param, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
