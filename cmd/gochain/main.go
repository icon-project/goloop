package main

import (
	"encoding/json"
	"flag"
	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/network"
	"io/ioutil"
	"log"
	"os"
)

type GoChainConfig struct {
	chain.Config
	P2PAddr string `json:"p2p"`
	Key     []byte `json:"key"`
}

func main() {
	var configFile, genesisFile string
	var generate bool
	var cfg GoChainConfig

	flag.StringVar(&configFile, "config", "", "Parsing configuration file")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.StringVar(&cfg.Channel, "channel", "default", "Channel name for the chain")
	flag.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Listen ip-port of P2P")
	flag.IntVar(&cfg.NID, "nid", 1, "Chain Network ID")
	flag.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.StringVar(&cfg.SeedAddr, "seed", "", "Ip-port of Seed")
	flag.StringVar(&genesisFile, "genesis", "", "Genesis transaction param")
	flag.UintVar(&cfg.Role, "role", 0, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.Parse()

	if len(genesisFile) > 0 {
		genesis, err := ioutil.ReadFile(genesisFile)
		if err != nil {
			log.Panicf("Fail to open genesis file=%s err=%+v", genesisFile, err)
		}
		cfg.Genesis = genesis
	}

	if len(configFile) > 0 && !generate {
		if bs, e := ioutil.ReadFile(configFile); e == nil {
			if err := json.Unmarshal(bs, &cfg); err != nil {
				log.Panicf("Illegal config file=%s err=%+v", configFile, err)
			}
		} else {
			log.Panicf("Fail to open config file=%s err=%+v", configFile, e)
		}
	}

	key := cfg.Key
	var priK *crypto.PrivateKey
	if len(key) == 0 {
		priK, _ = crypto.GenerateKeyPair()
		cfg.Key = priK.Bytes()
	} else {
		var err error
		if priK, err = crypto.ParsePrivateKey(key); err != nil {
			log.Panicf("Illegal key data=[%x]", key)
		}
	}

	if generate {
		f, err := os.OpenFile(configFile, os.O_CREATE|os.O_WRONLY, 0777)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", configFile, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	wallet, _ := common.WalletFromPrivateKey(priK)
	nt := network.NewTransport(cfg.P2PAddr, priK)
	nt.Listen()
	defer nt.Close()

	c := chain.NewChain(wallet, nt, &cfg.Config)
	c.Start()
}
