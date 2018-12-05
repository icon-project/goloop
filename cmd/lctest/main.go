package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/icon-project/goloop/common/legacy"
	"log"
	"net/http"
)

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

type Wallet struct {
	url string
}

const (
	ClearLine = "\x1b[2K"
)

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
	blk, err := legacy.ParseBlockV1(b)
	if err != nil {
		return err
	}
	if blk == nil {
		log.Printf("Parsing failure:%s", string(b))
		return errors.New("Parse Fail")
	}
	var info = map[int]int{}
	txs := blk.NormalTransactions()
	if txs != nil {
		for i := txs.Iterator(); i.Has(); i.Next() {
			if t, _, err := i.Get(); err == nil {
				info[t.Version()] += 1
			}
		}
	}
	fmt.Printf("%s<> BLOCK %8d %s tx=%v\r", ClearLine,
		blk.Height(), hex.EncodeToString(blk.ID()), info)
	return blk.Verify()
}
func VerifyBlocksFromHeight(wallet Wallet, from int) {
	for i := from; ; i++ {
		//for i := 1; ; i++ {
		b, err := wallet.GetBlockByHeight(i)
		if err != nil {
			fmt.Println()
			log.Println("GetBlock ERROR", err)
			break
		}
		err = VerifyBlock(b)
		if err != nil {
			fmt.Println()
			log.Printf("VerifyBlock ERROR %+v", err)
			log.Println("Block", string(b))
			break
		}
	}
}

func main() {
	height := flag.Int("height", -1, "Height of block")
	network := flag.String("network", "main", "Name of network to use")
	api := flag.String("api", "v3", "JSON RPC API Version")
	flag.Parse()

	wallet := Wallet{"https://wallet.icon.foundation/api/" + *api}
	if *network == "test" {
		wallet = Wallet{"https://testwallet.icon.foundation/api/" + *api}
	}
	for _, a := range flag.Args() {
		switch a {
		case "verify":
			VerifyBlocksFromHeight(wallet, *height)
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
			blk, err := legacy.ParseBlockV1(b)
			if err != nil {
				log.Println("PARSE FAILs", err)
				break
			}
			if err := blk.Verify(); err != nil {
				log.Println("VERIFY FAILs", err)
				break
			}
		}
	}
	return
}
