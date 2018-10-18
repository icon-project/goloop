package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/icon-project/goloop/block"
)

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

type Wallet struct {
	url string
}

var wallet = Wallet{
	url: "https://testwallet.icon.foundation/api/v3",
}

func (w *Wallet) Call(method string, params map[string]interface{}) ([]byte, error) {
	d := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		d["params"] = params
	}
	req, err := json.Marshal(d)
	if err != nil {
		log.Println("Making request fails")
		log.Println("Data", d)
		return nil, err
	}
	resp, err := http.Post(w.url, "application/json", bytes.NewReader(req))
	if resp.StatusCode != 200 {
		return nil, errors.New(
			fmt.Sprintf("FAIL to call res=%d", resp.StatusCode))
	}

	var buf = make([]byte, 2048*1024)
	var bufLen, readed int = 0, 0

	for true {
		readed, _ = resp.Body.Read(buf[bufLen:])
		if readed < 1 {
			break
		}
		bufLen += readed
	}
	var r JSONRPCResponse
	err = json.Unmarshal(buf[0:bufLen], &r)
	if err != nil {
		log.Println("JSON Parse Fail")
		log.Println("JSON=", string(buf[0:bufLen]))
		return nil, err
	}
	return r.Result.MarshalJSON()
}

func (w *Wallet) GetBlockByHeight(h int) ([]byte, error) {
	p := map[string]interface{}{
		"height": fmt.Sprintf("0x%x", h),
	}
	return w.Call("icx_getBlockByHeight", p)
}

func (w *Wallet) GetLastBlock() ([]byte, error) {
	return w.Call("icx_getLastBlock", nil)
}

func (w *Wallet) GetTransactionByHash(txHash string) ([]byte, error) {
	p := map[string]interface{}{
		"txHash": txHash,
	}
	return w.Call("icx_getTransactionByHash", p)
}

func (w *Wallet) GetTransactionResultByHash(txHash string) ([]byte, error) {
	p := map[string]interface{}{
		"txHash": txHash,
	}
	return w.Call("icx_getTransactionResultByHash", p)
}

func VerifyBlock(b []byte) error {
	blk, err := block.NewBlockV1(b)
	if err != nil {
		return err
	}
	return blk.Verify()
}
func GetBlocksFromHeight(from int) {
	for i := from; ; i++ {
		//for i := 1; ; i++ {
		b, err := wallet.GetBlockByHeight(i)
		if err != nil {
			log.Println("GetBlock ERROR", err)
			break
		}
		err = VerifyBlock(b)
		if err != nil {
			log.Println("VerifyBlock ERROR", err)
			log.Println("Block", string(b))
			break
		}
	}
}

func main() {
	height := flag.Int("height", -1, "Height of block")
	flag.Parse()
	for _, a := range flag.Args() {
		switch a {
		case "verify":
			GetBlocksFromHeight(*height)
		case "get":
			var b []byte
			var err error
			if *height < 0 {
				b, err = wallet.GetLastBlock()
				if err != nil {
					log.Printf("GetLastBlock() FAILs error=%v\n", err)
					break
				}
			} else {
				b, err = wallet.GetBlockByHeight(*height)
				if err != nil {
					log.Printf("GetBlockByHeight(%d) FAILs error=%v\n",
						*height, err)
					break
				}
			}
			log.Println("<> BLOCK", *height)
			fmt.Println(string(b))
			blk, err := block.NewBlockV1(b)
			if err != nil {
				log.Println("PARSE FAILs", err)
				break
			}
			log.Printf("PARSED %+v\n", blk)
		}
	}
	return
}
