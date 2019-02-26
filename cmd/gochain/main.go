package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"syscall"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/eeproxy"
)

type GoChainConfig struct {
	chain.Config
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	Key           []byte `json:"key"`
	EESocket      string `json:"ee_socket"`

	fileName string
}

func (config *GoChainConfig) String() string {
	return ""
}

var (
	version = "unknown"
	build   = "unknown"
)

func (config *GoChainConfig) Set(name string) error {
	config.fileName = name
	if bs, e := ioutil.ReadFile(name); e == nil {
		if err := json.Unmarshal(bs, config); err != nil {
			return err
		}
	}
	return nil
}

var memProfileCnt int32 = 0

func main() {
	var genesisFile string
	var saveFile string
	var cfg GoChainConfig
	var cpuProfile, memProfile string
	var nodePath, chainPath string

	flag.Var(&cfg, "config", "Parsing configuration file")
	flag.StringVar(&saveFile, "save", "", "File path for storing current configuration(it exits after save)")
	flag.StringVar(&cfg.Channel, "channel", "default", "Channel name for the chain")
	flag.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	flag.StringVar(&cfg.P2PListenAddr, "p2p_listen", ":8080", "Listen ip-port of P2P")
	flag.IntVar(&cfg.NID, "nid", 1, "Chain Network ID")
	flag.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.StringVar(&cfg.SeedAddr, "seed", "", "Ip-port of Seed")
	flag.StringVar(&genesisFile, "genesis", "", "Genesis transaction param")
	flag.StringVar(&cfg.DBType, "db_type", "mapdb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	flag.StringVar(&cfg.DBDir, "db_dir", "", "Database directory")
	flag.StringVar(&cfg.DBName, "db_name", "", "Database name for the chain(default:<channel name>)")
	flag.UintVar(&cfg.Role, "role", 2, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.StringVar(&cfg.WALDir, "wal_dir", "", "WAL directory")
	flag.StringVar(&cfg.ContractDir, "contract_dir", "", "Contract directory")
	flag.StringVar(&cfg.EESocket, "ee_socket", "", "Execution engine socket path")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	flag.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")
	flag.StringVar(&nodePath, "node_dir", "", "Node data directory(default:.chain/<address>)")
	flag.StringVar(&chainPath, "chain_dir", "", "Chain data directory(default:<node_dir>/<nid>")
	flag.Parse()

	if len(genesisFile) > 0 {
		genesis, err := ioutil.ReadFile(genesisFile)
		if err != nil {
			log.Panicf("Fail to open genesis file=%s err=%+v", genesisFile, err)
		}
		cfg.Genesis = genesis
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
	wallet, _ := common.NewWalletFromPrivateKey(priK)

	if len(cfg.Genesis) == 0 {
		genesis := map[string]interface{}{
			"accounts": []map[string]interface{}{
				{
					"name":    "god",
					"address": wallet.Address().String(),
					"balance": "0x2961fff8ca4a62327800000",
				},
				{
					"name":    "treasury",
					"address": "hx1000000000000000000000000000000000000000",
					"balance": "0x0",
				},
			},
			"message": "gochain generated gensis",
			"validatorlist": []string{
				wallet.Address().String(),
			},
		}
		cfg.Genesis, _ = json.Marshal(genesis)
	}

	if saveFile != "" {
		f, err := os.OpenFile(saveFile,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", cfg.fileName, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	prefix := fmt.Sprintf("%x|--|", wallet.Address().ID()[0:2])
	log.SetPrefix(prefix)

	addr := wallet.Address()

	if nodePath == "" {
		nodePath = path.Join(".", ".chain", addr.String())
	}

	if chainPath == "" {
		chainPath = path.Join(nodePath, strconv.FormatInt(int64(cfg.NID), 16))
	}

	if cfg.DBDir == "" {
		cfg.DBDir = path.Join(chainPath, "db")
	}

	if cfg.WALDir == "" {
		cfg.WALDir = path.Join(chainPath, "wal")
	}

	if cfg.ContractDir == "" {
		cfg.ContractDir = path.Join(nodePath, "contract")
	}

	if cfg.EESocket == "" {
		cfg.EESocket = path.Join(nodePath, "socket")
	}

	if cfg.DBType != "mapdb" {
		if err := os.MkdirAll(cfg.DBDir, 0700); err != nil {
			log.Panicf("Fail to create directory %s err=%+v", cfg.DBDir, err)
		}
	}

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatalf("Fail to create %s for profile err=%+v", cpuProfile, err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Fail to start profiling err=%+v", err)
		}
		defer func() {
			pprof.StopCPUProfile()
		}()
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func(c chan os.Signal) {
			<-c
			pprof.StopCPUProfile()
		}(c)
	}

	if memProfile != "" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		go func(c chan os.Signal) {
			for {
				<-c
				cnt := atomic.AddInt32(&memProfileCnt, 1)
				fileName := fmt.Sprintf("%s.%03d", memProfile, cnt)
				if f, err := os.Create(fileName); err == nil {
					pprof.WriteHeapProfile(f)
					f.Close()
				}
			}
		}(c)
	}

	logoLines := []string{
		"  ____  ___   ____ _   _    _    ___ _   _ ",
		" / ___|/ _ \\ / ___| | | |  / \\  |_ _| \\ | |",
		"| |  _| | | | |   | |_| | / _ \\  | ||  \\| |",
		"| |_| | |_| | |___|  _  |/ ___ \\ | || |\\  |",
		" \\____|\\___/ \\____|_| |_/_/   \\_\\___|_| \\_|",
	}
	for _, l := range logoLines {
		log.Println(l)
	}
	log.Printf("Version : %s", version)
	log.Printf("Build   : %s", build)

	nt := network.NewTransport(cfg.P2PAddr, wallet)
	if cfg.P2PListenAddr != "" {
		_ = nt.SetListenAddress(cfg.P2PListenAddr)
	}
	err := nt.Listen()
	if err != nil {
		log.Panicf("FAIL to listen P2P err=%+v", err)
	}
	defer nt.Close()

	pm, err := eeproxy.NewManager("unix", cfg.EESocket)
	if err != nil {
		log.Panicln("FAIL to start EEManager")
	}
	go pm.Loop()

	ee, err := eeproxy.NewPythonEE()
	if err != nil {
		log.Panicf("FAIL to create PythonEE err=%+v", err)
	}
	pm.SetEngine("python", ee)
	pm.SetInstances("python", 1)

	c := chain.NewChain(wallet, nt, pm, &cfg.Config)
	c.Start()
}
