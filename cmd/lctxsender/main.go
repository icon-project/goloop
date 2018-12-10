package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/icon-project/goloop/common/legacy"
	client "github.com/ybbus/jsonrpc"
)

const (
	ClearLine = "\x1b[2K"
)

func main() {
	var cfg struct {
		URL      string
		Height   int
		TxDelay  int64
		BlkDelay int64
		BlockDB  string
	}

	flag.StringVar(&cfg.URL, "u", "http://localhost:9080/api", "API Base URL")
	flag.IntVar(&cfg.Height, "h", -1, "Maximum height of blocks(-1 for infinite)")
	flag.Int64Var(&cfg.TxDelay, "t", 0, "Delay(microseconds) between transactions")
	flag.Int64Var(&cfg.BlkDelay, "b", 0, "Delay(microseconds) between blocks ")
	flag.StringVar(&cfg.BlockDB, "db", "./data/testnet/block", "Path to legacy block database")
	flag.Parse()

	v3Client := client.NewClient((cfg.URL) + "/v3")
	v2Client := client.NewClient((cfg.URL) + "/v2")

	db, err := legacy.OpenDatabase(cfg.BlockDB, "")
	if err != nil {
		log.Printf("Fail to open database err=%+v", err)
		return
	}

	var logs [5]struct {
		when  time.Time
		total int64
	}
	var front, back int

	logs[front].when = time.Now()
	logs[front].total = 0

	var total int64 = 0
	for bh := 1; cfg.Height < 0 || bh < cfg.Height; bh++ {
		blk, err := db.GetBlockByHeight(bh)
		if err != nil {
			log.Printf("Fail to get block err=%+v", err)
			return
		}
		txl := blk.NormalTransactions()
		j := 0
		for i := txl.Iterator(); i.Has(); i.Next() {
			tx, _, err := i.Get()
			if err != nil {
				log.Printf("Fail to get transaction err=%+v", err)
				os.Exit(-1)
			}

			var response *client.RPCResponse

			for response == nil {
				if tx.Version() == 2 {
					response, err = v2Client.Call("icx_sendTransaction", tx)
				} else {
					response, err = v3Client.Call("icx_sendTransaction", tx)
				}
				if err != nil {
					fmt.Printf("\nFail to RPC Call err=%+v\r", err)
					os.Exit(-1)
				}

				if response.Error != nil {
					fmt.Printf("%sTX<%x> Retry msg=%v\r", ClearLine, tx.ID(), response.Error)
					time.Sleep(time.Millisecond * 500)
					response = nil
				}
			}

			j++
			total++
			if (total % 1000) == 0 {
				front = (front + 1) % len(logs)
				if front == back {
					back = (back + 1) % len(logs)
				}
				logs[front].when = time.Now()
				logs[front].total = total
				tps := float64(logs[front].total-logs[back].total) / float64(logs[front].when.Sub(logs[back].when)/time.Millisecond) * 1000
				fmt.Printf("%sBlk[%7d] Tx[%8d] TPS=%.2f\n", ClearLine, bh, total, tps)
			} else {
				fmt.Printf("%sBlk[%7d] Tx[%8d]\r", ClearLine, bh, total)
			}
			time.Sleep(time.Duration(cfg.TxDelay) * time.Microsecond)
		}
		time.Sleep(time.Duration(cfg.BlkDelay) * time.Microsecond)
	}
	fmt.Printf("Total Tx : %d\n", total)
}
