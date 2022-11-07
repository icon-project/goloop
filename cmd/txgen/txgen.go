package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	ClearLine = "\x1b[2K"
)

var (
	ErrEndOfTransaction = errors.New("EndOfTransaction")
)

type Client struct {
	*client.JsonRpcClient
}

func (c *Client) SendTx(tx interface{}) (string, error) {
	for {
		r, err := c.Do("icx_sendTransaction", tx, nil)
		if err != nil {
			if re, ok := err.(*jsonrpc.Error); ok {
				if re.Code == jsonrpc.ErrorCodeTxPoolOverflow {
					continue
				}
				return "", errors.Errorf("RPC Server Error code=%d msg=%s", r.Error.Code, r.Error.Message)
			}
			return "", err
		}
		var txHash string
		if err := json.Unmarshal(r.Result, &txHash); err != nil {
			return "", err
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

func (c *Client) GetTxResult(tid string, wait time.Duration) (*TransactionResult, error) {
	params := map[string]interface{}{
		"txHash": tid,
	}
	startTime := time.Now()
	for {
		result := new(TransactionResult)
		_, err := c.Do("icx_getTransactionResult", params, result)
		if err != nil {
			if re, ok := err.(*jsonrpc.Error); ok {
				if time.Now().Sub(startTime) > wait {
					return nil, errors.Errorf("RPC Error timeout=%s code=%d msg=%s",
						wait, re.Code, re.Message)
				}
				time.Sleep(time.Millisecond * 100)
				continue
			}
			return nil, err
		}
		return result, nil
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

func (c *Client) SendTxAndGetResult(tx interface{}, wait time.Duration) (*TransactionResult, error) {
	tid, err := c.SendTx(tx)
	if err != nil {
		return nil, err
	}
	return c.GetTxResult(tid, wait)
}

type TransactionMaker interface {
	Prepare(client *Client) error
	MakeOne() (interface{}, error)
}

type Context struct {
	concurrent int
	tps        int64
	delay      time.Duration
	timeout    time.Duration

	lock           sync.Mutex
	firstTime      time.Time
	lastTime       time.Time
	lastTxCount    int64
	currentTxCount int64
	resetCount     int64

	maker TransactionMaker
}

func NewContext(concurrent int, tps int64, maker TransactionMaker, timeout int64) *Context {
	return &Context{
		tps:        tps,
		concurrent: concurrent,
		maker:      maker,
		timeout:    time.Duration(timeout) * time.Millisecond,
	}
}

func (ctx *Context) sendRequests(wg *sync.WaitGroup, client *Client) {
	method := "icx_sendTransaction"
	if ctx.timeout > 0 {
		method = "icx_sendTransactionAndWait"
	}

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
			if err != ErrEndOfTransaction {
				log.Printf("Fail to make transaction err=%+v", err)
			}
			return
		}

		for {
			r, err := client.Do(method, tx, nil)
			if err != nil {
				if re, ok := err.(*jsonrpc.Error); ok {
					if re.Code == jsonrpc.ErrorCodeTxPoolOverflow {
						time.Sleep(ctx.delay / 3)
						continue
					}
					if re.Code == jsonrpc.ErrorCodeTimeout || re.Code == jsonrpc.ErrorCodeSystemTimeout {
						time.Sleep(ctx.delay / 3)
						continue
					}
					js, _ := json.MarshalIndent(tx, "", "  ")
					log.Panicf("Get ERROR on %s Code=%d Msg=%s TX=%s",
						method, r.Error.Code, r.Error.Message, js)
				}
				if ue, ok := err.(*url.Error); ok {
					if ne, ok := ue.Err.(*net.OpError); ok {
						if se, ok := ne.Err.(*os.SyscallError); ok && se.Err == syscall.ECONNRESET {
							atomic.AddInt64(&ctx.resetCount, 1)
							continue
						}
					}
				}
				js, _ := json.MarshalIndent(tx, "", "  ")
				log.Panicf("Fail to send TX err=%+v tx=%s", err, js)
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
		resetCnt := atomic.LoadInt64(&ctx.resetCount)
		fmt.Printf("%scurrent_TPS [%8.2f] average_TPS [%8.2f] reset_CNT [%4d] TX=%6d \r",
			ClearLine, tps, avgTps, resetCnt,
			ctx.currentTxCount)

		ctx.lastTxCount = ctx.currentTxCount
		ctx.lastTime = now
	}
}

func (ctx *Context) Run(urls []string) error {
	ctx.delay = (time.Second * time.Duration(ctx.concurrent*len(urls))) /
		time.Duration(ctx.tps)

	headers := map[string]string{}
	timeoutOption := int64(ctx.timeout / time.Millisecond)
	if timeoutOption > 200 {
		iconOpts := jsonrpc.IconOptions{}
		iconOpts.SetInt(jsonrpc.IconOptionsTimeout, timeoutOption)
		headers[jsonrpc.HeaderKeyIconOptions] = iconOpts.ToHeaderValue()
	}

	c := &Client{client.NewJsonRpcClient(&http.Client{}, urls[0])}
	if err := ctx.maker.Prepare(c); err != nil {
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
			c := &Client{client.NewJsonRpcClient(&http.Client{}, url)}
			c.CustomHeader = headers
			go ctx.sendRequests(&wg, c)
			time.Sleep(ctx.delay / time.Duration(ctx.concurrent))
		}
	}
	wg.Wait()
	log.Println("\n[#] End of transaction generation")
	return nil
}

func TimeStampNow() string {
	return "0x" + strconv.FormatInt(time.Now().UnixNano()/1000, 16)
}
