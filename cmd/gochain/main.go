package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"syscall"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	DefaultKeyStorePass = "gochain"
)

type GoChainConfig struct {
	chain.Config
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	EESocket      string `json:"ee_socket"`
	RPCAddr       string `json:"rpc_addr"`
	RPCDump       bool   `json:"rpc_dump"`
	EEInstances   int    `json:"ee_instances"`

	Key          []byte          `json:"key,omitempty"`
	KeyStoreData json.RawMessage `json:"key_store"`
	KeyStorePass string          `json:"key_password"`
}

func (config *GoChainConfig) String() string {
	return ""
}

func (config *GoChainConfig) Type() string {
	return "GoChainConfig"
}

func (config *GoChainConfig) Set(name string) error {
	config.FilePath, _ = filepath.Abs(name)
	if bs, e := ioutil.ReadFile(name); e == nil {
		if err := json.Unmarshal(bs, config); err != nil {
			return err
		}
	}
	return nil
}

var memProfileCnt int32 = 0

var (
	version = "unknown"
	build   = "unknown"
)

func main() {
	var genesisStorage, genesisPath string
	var keyStoreFile, keyStoreSecret string
	var saveFile, saveKeyStore string
	var cfg GoChainConfig
	var cpuProfile, memProfile string
	var chainDir string
	var eeSocket string

	flag.Var(&cfg, "config", "Parsing configuration file")
	flag.StringVar(&saveFile, "save", "", "File path for storing current configuration(it exits after save)")
	flag.StringVar(&saveKeyStore, "save_key_store", "", "File path for storing current KeyStore")
	flag.StringVar(&cfg.Channel, "channel", "default", "Channel name for the chain")
	flag.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	flag.StringVar(&cfg.P2PListenAddr, "p2p_listen", "", "Listen ip-port of P2P")
	flag.IntVar(&cfg.NID, "nid", 0, "Chain Network ID")
	flag.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.BoolVar(&cfg.RPCDump, "rpc_dump", false, "JSON-RPC Request, Response Dump flag")
	flag.StringVar(&cfg.SeedAddr, "seed", "", "Ip-port of Seed")
	flag.StringVar(&genesisStorage, "genesis_storage", "", "Genesis storage path")
	flag.StringVar(&genesisPath, "genesis", "", "Genesis template directory or file")
	flag.StringVar(&cfg.DBType, "db_type", "goleveldb", "Name of database system(badgerdb, *goleveldb, boltdb, mapdb)")
	flag.UintVar(&cfg.Role, "role", 2, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.StringVar(&eeSocket, "ee_socket", "", "Execution engine socket path(default:.chain/<address>/ee.sock")
	flag.StringVar(&keyStoreFile, "key_store", "", "KeyStore file for wallet")
	flag.StringVar(&keyStoreSecret, "key_secret", "", "Secret(password) file for KeyStore")
	flag.StringVar(&cfg.KeyStorePass, "key_password", "", "Password for the KeyStore file")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	flag.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")
	flag.StringVar(&chainDir, "chain_dir", "", "Chain data directory(default:.chain/<address>/<nid>")
	flag.IntVar(&cfg.EEInstances, "ee_instances", 1, "Number of execution engines")
	flag.IntVar(&cfg.ConcurrencyLevel, "concurrency", 1, "Maximum number of executors to use for concurrency")
	flag.Parse()

	if len(keyStoreFile) > 0 {
		if ks, err := ioutil.ReadFile(keyStoreFile); err != nil {
			log.Panicf("Fail to open KeyStore file=%s err=%+v", keyStoreFile, err)
		} else {
			cfg.KeyStoreData = ks
			cfg.Key = []byte{}
		}
	}

	keyStorePass := []byte(cfg.KeyStorePass)
	if len(keyStoreSecret) > 0 {
		if ks, err := ioutil.ReadFile(keyStoreSecret); err != nil {
			log.Panicf("Fail to open KeySecret file=%s err=%+v", keyStoreSecret, err)
		} else {
			keyStorePass = ks
		}
	}

	var priK *crypto.PrivateKey
	if len(cfg.Key) > 0 {
		var err error
		if priK, err = crypto.ParsePrivateKey(cfg.Key); err != nil {
			log.Panicf("Illegal key data=[%x]", cfg.Key)
		}
		cfg.Key = nil
	}

	if len(cfg.KeyStoreData) > 0 {
		var err error
		if len(keyStorePass) == 0 {
			log.Panicf("There is no password information for the KeyStore")
		}
		priK, err = wallet.DecryptKeyStore(cfg.KeyStoreData, keyStorePass)
		if err != nil {
			log.Panicf("Fail to decrypt KeyStore err=%+v", err)
		}
	} else {
		// make sure that cfg.KeyStoreData always has valid value to let them
		// be stored with -save_key_store option even though the key is
		// provided by cfg.Key value.
		if priK == nil {
			priK, _ = crypto.GenerateKeyPair()
		}
		if len(keyStorePass) == 0 {
			cfg.KeyStorePass = DefaultKeyStorePass
			keyStorePass = []byte(cfg.KeyStorePass)
		}

		if ks, err := wallet.EncryptKeyAsKeyStore(priK, keyStorePass); err != nil {
			log.Panicf("Fail to encrypt private key err=%+v", err)
		} else {
			cfg.KeyStoreData = ks
		}
	}
	wallet, _ := wallet.NewFromPrivateKey(priK)

	if len(genesisStorage) > 0 {
		storage, err := ioutil.ReadFile(genesisStorage)
		if err != nil {
			log.Panicf("Fail to open genesisStorage=%s err=%+v\n", genesisStorage, err)
		}
		cfg.GenesisStorage, err = chain.NewGenesisStorage(storage)
		if err != nil {
			log.Panicf("Failed to load genesisStorage\n")
		}
	} else if len(genesisPath) > 0 {
		storage := bytes.NewBuffer(nil)
		if err := chain.WriteGenesisStorageFromPath(storage, genesisPath); err != nil {
			log.Printf("FAIL to generate gs. err = %s, path = %s\n", err, genesisPath)
		}
		var err error
		cfg.GenesisStorage, err = chain.NewGenesisStorage(storage.Bytes())
		if err != nil {
			log.Panicf("Failed to load genesisStorage\n")
		}
	} else if len(cfg.Genesis) == 0 {
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
			"chain": map[string]interface{}{
				"validatorList": []string{
					wallet.Address().String(),
				},
			},
			"message": "gochain generated genesis",
		}
		if cfg.NID != 0 {
			genesis["nid"] = fmt.Sprintf("%#x", cfg.NID)
		}
		cfg.Genesis, _ = json.Marshal(genesis)
	}

	if cfg.NID == 0 {
		gtx, _ := transaction.NewGenesisTransaction(cfg.Genesis)
		cfg.NID = gtx.NID()
	}

	if len(saveKeyStore) > 0 {
		ks := bytes.NewBuffer(nil)
		if err := json.Indent(ks, cfg.KeyStoreData, "", "  "); err != nil {
			log.Panicf("Fail to indenting key data err=%+v", err)
		}
		if err := ioutil.WriteFile(saveKeyStore, ks.Bytes(), 0600); err != nil {
			log.Panicf("Fail to save key store to the file=%s err=%+v", saveKeyStore, err)
		}
	}

	if saveFile != "" {
		f, err := os.OpenFile(saveFile,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", saveFile, err)
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

	if chainDir != "" {
		cfg.BaseDir = cfg.ResolveRelative(chainDir)
	}

	if cfg.BaseDir == "" {
		cfg.BaseDir = cfg.ResolveRelative(path.Join(".chain",
			wallet.Address().String(), strconv.FormatInt(int64(cfg.NID), 16)))
	}

	if eeSocket != "" {
		cfg.EESocket = cfg.ResolveRelative(eeSocket)
	}

	if cfg.EESocket == "" {
		cfg.EESocket = cfg.ResolveRelative(path.Join(".chain",
			wallet.Address().String(), "ee.sock"))
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
			os.Exit(128 + int(syscall.SIGINT))
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

	metric.Initialize(wallet)
	nt := network.NewTransport(cfg.P2PAddr, wallet)
	if cfg.P2PListenAddr != "" {
		_ = nt.SetListenAddress(cfg.P2PListenAddr)
	}
	err := nt.Listen()
	if err != nil {
		log.Panicf("FAIL to listen P2P err=%+v", err)
	}
	defer nt.Close()

	ee, err := eeproxy.NewPythonEE()
	if err != nil {
		log.Panicf("FAIL to create PythonEE err=%+v", err)
	}

	pm, err := eeproxy.NewManager("unix", cfg.EESocket, ee)
	if err != nil {
		log.Panicln("FAIL to start EEManager")
	}
	go pm.Loop()

	pm.SetInstances(cfg.EEInstances, cfg.EEInstances, cfg.EEInstances)

	// TODO : server-chain setting
	srv := server.NewManager(cfg.RPCAddr, cfg.RPCDump, wallet)
	c := chain.NewChain(wallet, nt, srv, pm, &cfg.Config)
	err = c.Init(true)
	if err != nil {
		log.Panicf("FAIL to initialize Chain err=%+v", err)
	}
	err = c.Start(true)
	if err != nil {
		log.Panicf("FAIL to start Chain err=%+v", err)
	}

	// main loop
	srv.Start()
}
