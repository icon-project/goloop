package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime/pprof"
	"sync/atomic"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc/metric"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	DefaultKeyStorePass = "gochain"
)

type GoLoopConfig struct {
	NodeConfig
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	EESocket      string `json:"ee_socket"`
	RPCAddr       string `json:"rpc_addr"`
	EEInstances   int    `json:"ee_instances"`

	Key          []byte          `json:"key,omitempty"`
	KeyStoreData json.RawMessage `json:"key_store"`
	KeyStorePass string          `json:"key_password"`

	fileName string
}

func (config *GoLoopConfig) String() string {
	return ""
}

func (config *GoLoopConfig) Type() string {
	return "GoLoopConfig"
}

func (config *GoLoopConfig) Set(name string) error {
	config.fileName = name
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

	cfg                          GoLoopConfig
	keyStoreFile, keyStoreSecret string
	saveFile, saveKeyStore       string

	cpuProfile, memProfile string

	w module.Wallet
)

func initConfig() {
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
			log.Println("Generated KeyStore", common.NewAccountAddressFromPublicKey(priK.PublicKey()).String())
			cfg.KeyStoreData = ks
		}
	}
	w, _ = wallet.NewFromPrivateKey(priK)

	if len(saveKeyStore) > 0 {
		ks := bytes.NewBuffer(nil)
		if err := json.Indent(ks, cfg.KeyStoreData, "", "  "); err != nil {
			log.Panicf("Fail to indenting key data err=%+v", err)
		}
		if err := ioutil.WriteFile(saveKeyStore, ks.Bytes(), 0700); err != nil {
			log.Panicf("Fail to save key store to the file=%s err=%+v", saveKeyStore, err)
		}
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
	prefix := fmt.Sprintf("%x|--|", w.Address().ID()[0:2])
	log.SetPrefix(prefix)

	addr := w.Address()

	if cfg.NodeDir == "" {
		cfg.NodeDir = path.Join(".", ".chain", addr.String())
	}
	if cfg.CliSocket == "" {
		sockPath := os.Getenv("GOLOOP_SOCK")
		if sockPath != "" {
			cfg.CliSocket = sockPath
		} else {
			cfg.CliSocket = path.Join(cfg.NodeDir, "cli.sock")
		}
	}
	if cfg.EESocket == "" {
		cfg.EESocket = path.Join(cfg.NodeDir, "ee.sock")
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
}

func main() {
	cobra.OnInitialize(initConfig)
	rootCmd := &cobra.Command{Use: "goloop"}
	rootFlags := rootCmd.Flags()
	rootCmd.PersistentFlags().VarP(&cfg, "config", "c", "Parsing configuration file")
	rootCmd.PersistentFlags().StringVarP(&cfg.CliSocket, "node_sock", "s", "",
		"Node Command Line Interface socket path(default $GOLOOP_SOCK=[node_dir]/cli.sock)")
	rootFlags.StringVar(&saveFile, "save", "", "File path for storing current configuration(it exits after save)")
	rootFlags.StringVar(&saveKeyStore, "save_key_store", "", "File path for storing current KeyStore")
	rootFlags.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	rootFlags.StringVar(&cfg.P2PListenAddr, "p2p_listen", "", "Listen ip-port of P2P")
	rootFlags.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	rootFlags.StringVar(&cfg.EESocket, "ee_socket", "", "Execution engine socket path")
	rootFlags.StringVar(&keyStoreFile, "key_store", "", "KeyStore file for w")
	rootFlags.StringVar(&keyStoreSecret, "key_secret", "", "Secret(password) file for KeyStore")
	rootFlags.StringVar(&cfg.KeyStorePass, "key_password", "", "Password for the KeyStore file")
	rootFlags.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	rootFlags.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")
	rootFlags.StringVar(&cfg.NodeDir, "node_dir", "", "Node data directory(default:.chain/<address>)")
	rootFlags.IntVar(&cfg.EEInstances, "ee_instances", 1, "Number of execution engines")
	rootFlags.StringVar(&cfg.DBType, "db_type", "goleveldb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	rootFlags.IntVar(&cfg.ConcurrencyLevel, "concurrency", 1, "Maximum number of executors to use for concurrency")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start goloop",
	}
	startCmd.Run = func(cmd *cobra.Command, args []string) {
		logoLines := []string{
			" ____  ___  _     ___   ___  ____",
			"/ ___|/ _ \\| |   / _ \\ / _ \\|  _ \\",
			"| |  _| | | | |  | | | | | | | |_) |",
			"| |_| | |_| | |__| |_| | |_| |  __/",
			"\\____|\\___/|_____\\___/ \\___/|_|",
			"",
			//"generated by http://patorjk.com/software/taag/#p=display&f=Ivrit&t=GOLOOP",
		}
		for _, l := range logoLines {
			log.Println(l)
		}
		log.Printf("Version : %s", version)
		log.Printf("Build   : %s", build)

		metric.Initialize(w)
		nt := network.NewTransport(cfg.P2PAddr, w)
		if cfg.P2PListenAddr != "" {
			_ = nt.SetListenAddress(cfg.P2PListenAddr)
		}
		err := nt.Listen()
		if err != nil {
			log.Panicf("FAIL to listen P2P err=%+v", err)
		}
		defer func() {
			if err := nt.Close(); err != nil {
				log.Panicf("FAIL to close P2P err=%+v", err)
			}
		}()

		ee, err := eeproxy.NewPythonEE()
		if err != nil {
			log.Panicf("FAIL to create PythonEE err=%+v", err)
		}
		pm, err := eeproxy.NewManager("unix", cfg.EESocket, ee)
		if err != nil {
			log.Panicln("FAIL to start EEManager")
		}
		if err := pm.SetInstances(cfg.EEInstances, cfg.EEInstances, cfg.EEInstances); err != nil {
			log.Panicf("FAIL to EEManager.SetInstances err=%+v", err)
		}
		go pm.Loop()

		srv := server.NewManager(cfg.RPCAddr, w)

		n := NewNode(w, nt, srv, pm, &cfg.NodeConfig)
		n.Start()
	}

	chainCmd := NewChainCmd(&cfg)
	systemCmd := NewSystemCmd(&cfg)
	rootCmd.AddCommand(startCmd, chainCmd, systemCmd)
	rootCmd.Execute()
}
