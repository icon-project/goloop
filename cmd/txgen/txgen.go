package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	rpc "github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc"
)

const (
	ClearLine = "\x1b[2K"
)

type Client struct {
	jsonrpc.RPCClient
}

func (client *Client) SendTx(tx interface{}) (string, error) {
	for {
		r, err := client.Call("icx_sendTransaction", tx)
		if err != nil {
			return "", err
		}
		if r.Error != nil {
			if rpc.ErrorCode(r.Error.Code) == rpc.ErrorCodeTxPoolOverflow {
				continue
			}
			return "", errors.Errorf("RPC Server Error code=%d msg=%s", r.Error.Code, r.Error.Message)
		}
		txHash, ok := r.Result.(string)
		if !ok {
			return "", errors.Errorf("Fail on parsing txHash")
		}
		return txHash, nil
	}
}

type TransactionFailure struct {
	Code    common.HexInt32 `json:"code"`
	Message string          `json:"message,omitempty"`
}

type TransactionResult struct {
	Status             common.HexInt32     `json:"status"`
	To                 common.Address      `json:"to"`
	Failure            *TransactionFailure `json:"failure,omitempty"`
	TxHash             common.HexBytes     `json:"txHash"`
	TxIndex            common.HexInt32     `json:"txIndex"`
	BlockHeight        common.HexInt64     `json:"blockHeight"`
	BlockHash          common.HexBytes     `json:"blockHash"`
	CumulativeStepUsed common.HexInt       `json:"cumulativeStepUsed"`
	SCOREAddress       *common.Address     `json:"scoreAddress"`
	StepUsed           common.HexInt       `json:"stepUsed"`
	StepPrice          common.HexInt       `json:"stepPrice"`
	LogsBloom          txresult.LogsBloom  `json:"logsBloom"`
}

func (client *Client) GetTxResult(tid string, wait time.Duration) (*TransactionResult, error) {
	params := map[string]interface{}{
		"txHash": tid,
	}
	startTime := time.Now()
	for {
		r, err := client.Call("icx_getTransactionResult", params)
		if err != nil {
			return nil, err
		}
		if r.Error == nil {
			var result TransactionResult
			if err := r.GetObject(&result); err != nil {
				return nil, err
			} else {
				return &result, nil
			}
		}
		if time.Now().Sub(startTime) > wait {
			return nil, errors.Errorf("RPC Error code=%d msg=%s",
				r.Error.Code, r.Error.Message)
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func SignTx(from module.Wallet, tx map[string]interface{}) error {
	js, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	bs, err := transaction.SerializeJSON(js, nil, nil)
	if err != nil {
		return err
	}
	bs = append([]byte("icx_sendTransaction."), bs...)

	sig, err := from.Sign(crypto.SHA3Sum256(bs))
	if err != nil {
		return err
	}
	tx["signature"] = sig
	return nil
}

func (client *Client) SendTxAndGetResult(tx interface{}, wait time.Duration) (*TransactionResult, error) {
	tid, err := client.SendTx(tx)
	if err != nil {
		return nil, err
	}
	return client.GetTxResult(tid, wait)
}

type TransactionMaker interface {
	Prepare(client *Client) error
	MakeOne() (interface{}, error)
}

type Context struct {
	concurrent int
	tps        int64
	delay      time.Duration

	lock           sync.Mutex
	firstTime      time.Time
	lastTime       time.Time
	lastTxCount    int64
	currentTxCount int64

	maker TransactionMaker
}

func NewContext(concurrent int, tps int64, maker TransactionMaker) *Context {
	return &Context{
		tps:        tps,
		concurrent: concurrent,
		maker:      maker,
	}
}

func (ctx *Context) sendRequests(wg sync.WaitGroup, client *Client) {
	nextTs := time.Now()
	defer wg.Done()
	for {
		current := time.Now()
		if nextTs.After(current) {
			time.Sleep(nextTs.Sub(current))
		} else {
			if current.Sub(nextTs) > ctx.delay*2 {
				nextTs = current
			}
		}
		ctx.OnRequest()
		tx, err := ctx.maker.MakeOne()
		if err != nil {
			log.Printf("Fail to make transaction err=%+v", err)
			return
		}

		for {
			r, err := client.Call("icx_sendTransaction", tx)
			if err != nil {
				log.Panicf("Fail to send TX err=%+v", err)
			}
			if r.Error != nil {
				if rpc.ErrorCode(r.Error.Code) == rpc.ErrorCodeTxPoolOverflow {
					time.Sleep(ctx.delay)
					continue
				}
				log.Panicf("Get ERROR on icx_sendTransaction Code=%d Msg=%s\n",
					r.Error.Code, r.Error.Message)
			}
			break
		}

		nextTs = nextTs.Add(ctx.delay)
	}
}

func calcTPS(duration time.Duration, count int64) float64 {
	durationInMilliSec := float64(duration.Nanoseconds() / 1000000)
	return float64(count) / durationInMilliSec * 1000
}

func (ctx *Context) OnRequest() {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	ctx.currentTxCount += 1

	if (ctx.currentTxCount % 100) == 0 {
		now := time.Now()
		tps := calcTPS(now.Sub(ctx.lastTime), ctx.currentTxCount-ctx.lastTxCount)
		avgTps := calcTPS(now.Sub(ctx.firstTime), ctx.currentTxCount)
		fmt.Printf("%scurrent_TPS [%8.2f] average_TPS [%8.2f] TX=%6d\r", ClearLine, tps, avgTps, ctx.currentTxCount)

		ctx.lastTxCount = ctx.currentTxCount
		ctx.lastTime = now
	}
}

func (ctx *Context) Run(urls []string) error {
	ctx.delay = (time.Second * time.Duration(ctx.concurrent*len(urls))) /
		time.Duration(ctx.tps)

	client := &Client{jsonrpc.NewClient(urls[0])}

	if err := ctx.maker.Prepare(client); err != nil {
		log.Printf("Fail to prepare err=%+v", err)
		return err
	}

	time.Sleep(ctx.delay)

	ctx.lastTime = time.Now()
	ctx.firstTime = ctx.lastTime

	var wg sync.WaitGroup
	for _, url := range urls {
		for i := 0; i < ctx.concurrent; i++ {
			wg.Add(1)
			client := &Client{jsonrpc.NewClient(url)}
			go ctx.sendRequests(wg, client)
			time.Sleep(ctx.delay)
		}
	}
	wg.Wait()
	return nil
}

func TimeStampNow() string {
	return "0x" + strconv.FormatInt(time.Now().UnixNano()/1000, 16)
}
