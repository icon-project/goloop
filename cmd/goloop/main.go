package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"sync/atomic"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

const (
	DefaultKeyStorePass = "gochain"
)

type GoLoopConfig struct {
	NodeConfig

	Key          []byte          `json:"key,omitempty"`
	KeyStoreData json.RawMessage `json:"key_store"`
	KeyStorePass string          `json:"key_password"`
}

func (config *GoLoopConfig) String() string {
	return ""
}

func (config *GoLoopConfig) Type() string {
	return "GoLoopConfig"
}

func (config *GoLoopConfig) Set(name string) error {
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

	cfg                          GoLoopConfig
	keyStoreFile, keyStoreSecret string
	saveKeyStore                 string
	nodeDir string
	cliSocket, eeSocket string

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


	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	prefix := fmt.Sprintf("%x|--|", w.Address().ID()[0:2])
	log.SetPrefix(prefix)

	if nodeDir != "" {
		cfg.BaseDir = cfg.ResolveRelative(nodeDir)
	}
	if cliSocket != "" {
		cfg.CliSocket = cfg.ResolveRelative(cliSocket)
	}
	if eeSocket != "" {
		cfg.EESocket = cfg.ResolveRelative(eeSocket)
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
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.VarP(&cfg, "config", "c", "Parsing configuration file")
	rootPFlags.StringVarP(&cliSocket, "node_sock", "s", "",
		"Node Command Line Interface socket path(default $GOLOOP_SOCK=[node_dir]/cli.sock)")
	rootPFlags.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	rootPFlags.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")

	serverCmd := &cobra.Command{Use: "server", Short: "Server management"}
	serverFlags := serverCmd.PersistentFlags()
	serverFlags.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	serverFlags.StringVar(&cfg.P2PListenAddr, "p2p_listen", "", "Listen ip-port of P2P")
	serverFlags.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	serverFlags.StringVar(&eeSocket, "ee_socket", "", "Execution engine socket path")
	serverFlags.StringVar(&keyStoreFile, "key_store", "", "KeyStore file for wallet")
	serverFlags.StringVar(&keyStoreSecret, "key_secret", "", "Secret(password) file for KeyStore")
	serverFlags.StringVar(&cfg.KeyStorePass, "key_password", "", "Password for the KeyStore file")
	serverFlags.IntVar(&cfg.EEInstances, "ee_instances", 1, "Number of execution engines")
	serverFlags.StringVar(&nodeDir, "node_dir", "",
		"Node data directory(default:<configuration file path>/<address>)")

	saveCmd := &cobra.Command{
		Use:   "save [file]",
		Short: "Save configuration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			saveFilePath := args[0]
			f, err := os.OpenFile(saveFilePath,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				log.Panicf("Fail to open file=%s err=%+v", saveFilePath, err)
			}
			enc := json.NewEncoder(f)
			enc.SetIndent("", "  ")
			if err := enc.Encode(&cfg); err != nil {
				log.Panicf("Fail to generate JSON for %+v", cfg)
			}
			f.Close()

			if len(saveKeyStore) > 0 {
				ks := bytes.NewBuffer(nil)
				if err := json.Indent(ks, cfg.KeyStoreData, "", "  "); err != nil {
					log.Panicf("Fail to indenting key data err=%+v", err)
				}
				if err := ioutil.WriteFile(saveKeyStore, ks.Bytes(), 0700); err != nil {
					log.Panicf("Fail to save key store to the file=%s err=%+v", saveKeyStore, err)
				}
			}
		},
	}
	saveCmd.Flags().StringVar(&saveKeyStore, "save_key_store", "", "File path for storing current KeyStore")
	serverCmd.AddCommand(saveCmd)

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start server",
	}
	startCmd.Run = func(cmd *cobra.Command, args []string) {
		logoLines := []string{
			"  ____  ___  _     ___   ___  ____",
			" / ___|/ _ \\| |   / _ \\ / _ \\|  _ \\",
			"| |  _| | | | |  | | | | | | | |_) |",
			"| |_| | |_| | |__| |_| | |_| |  __/",
			" \\____|\\___/|_____\\___/ \\___/|_|",
		}
		for _, l := range logoLines {
			log.Println(l)
		}
		log.Printf("Version : %s", version)
		log.Printf("Build   : %s", build)





		n := NewNode(w, &cfg.NodeConfig)
		n.Start()
	}
	serverCmd.AddCommand(startCmd)

	chainCmd := NewChainCmd(&cfg)
	systemCmd := NewSystemCmd(&cfg)
	rootCmd.AddCommand(serverCmd, chainCmd, systemCmd)

	rootCmd.Execute()
}
