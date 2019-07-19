package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"os/signal"
	"path"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/node"
)

const (
	DefaultKeyStorePass = "gochain"
)

type GoLoopConfig struct {
	node.NodeConfig

	KeyStoreData json.RawMessage `json:"key_store"`
	Key          []byte          `json:"key,omitempty"`

	KeyStorePass string `json:"key_password"`

	priK *crypto.PrivateKey

	LogLevel     string `json:"log_level"`
	ConsoleLevel string `json:"console_level"`
}

var memProfileCnt int32 = 0

var (
	version = "unknown"
	build   = "unknown"

	cfg                          GoLoopConfig
	flagCfg                      GoLoopConfig
	keyStoreFile, keyStoreSecret string
	saveKeyStore                 string
	nodeDir                      string
	cliSocket, eeSocket          string

	cpuProfile, memProfile string

	//viper resolving : flag -> env -> config -> viper.default -> flag.default
	vc = viper.New()

	viperDecodeOpt = func(c *mapstructure.DecoderConfig) {
		c.TagName = "json"
		c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			StringInterfaceMapToJsonRawMessageHookFunc,
			c.DecodeHook)
	}
)

func StringInterfaceMapToJsonRawMessageHookFunc(
	inputValType reflect.Type, outValType reflect.Type, input interface{}) (interface{}, error) {
	if outValType.Name() == "RawMessage" {
		if inputValType.Kind() == reflect.Map && inputValType.Key().Kind() == reflect.String {
			return json.Marshal(input)
		} else if inputValType.Kind() == reflect.String && input != "" {
			return ioutil.ReadFile(input.(string))
		}
	}
	return input, nil
}

func initConfig() {
	cfg.BuildVersion = version
	cfg.BuildTags = build

	if cfg.FilePath == "" {
		//if GOLOOP_CONFIG is declared, using it
		cfg.FilePath = vc.GetString("config")
	}

	if cfg.FilePath != "" {
		f, err := os.Open(cfg.FilePath)
		if err != nil {
			log.Panicf("Fail to open config file=%s err=%+v", cfg.FilePath, err)
		}
		vc.SetConfigType("json")
		err = vc.ReadConfig(f)
		if err != nil {
			log.Panicf("Fail to read config file=%s err=%+v", cfg.FilePath, err)
		}
	}

	err := vc.Unmarshal(&cfg, viperDecodeOpt)
	if err != nil {
		log.Panicf("Fail to unmarshall config from env err=%+v", err)
	}
	err = vc.Unmarshal(&cfg.NodeConfig, viperDecodeOpt)
	if err != nil {
		log.Panicf("Fail to unmarshall config from env err=%+v", err)
	}

	//[TBD] flag.KeyStoreSecret -> env.KeyStoreSecret ->flag.KeyStorePass -> env.KeyStorePass -> config.KeyStorePass
	keyStoreSecret = vc.GetString("key_secret")
	if keyStoreSecret != "" {
		if ksp, err := ioutil.ReadFile(keyStoreSecret); err != nil {
			log.Panicf("Fail to open KeySecret file=%s err=%+v", keyStoreSecret, err)
		} else {
			cfg.KeyStorePass = strings.TrimSpace(string(ksp))
		}
	}

	if len(cfg.KeyStoreData) > 0 {
		if len(cfg.KeyStorePass) == 0 {
			log.Panicf("There is no password information for the KeyStore")
		}
		if k, err := wallet.DecryptKeyStore(cfg.KeyStoreData, []byte(cfg.KeyStorePass)); err != nil {
			log.Panicf("Fail to decrypt KeyStore err=%+v", err)
		} else {
			cfg.priK = k
		}
	} else if len(cfg.Key) > 0 {
		if k, err := crypto.ParsePrivateKey(cfg.Key); err != nil {
			log.Panicf("Illegal key data=[%x]", cfg.Key)
		} else {
			cfg.priK = k
		}
	}

	if nodeDir != "" {
		cfg.BaseDir = cfg.ResolveRelative(nodeDir)
	}
	if cliSocket != "" {
		cfg.CliSocket = cfg.ResolveRelative(cliSocket)
	}
	if eeSocket != "" {
		cfg.EESocket = cfg.ResolveRelative(eeSocket)
	}

	if cpuProfile == "" {
		cpuProfile = vc.GetString("cpuprofile")
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

	if memProfile == "" {
		memProfile = vc.GetString("memprofile")
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
	vc.SetEnvPrefix("GOLOOP")
	vc.AutomaticEnv()

	cobra.OnInitialize(initConfig)
	rootCmd := &cobra.Command{Use: "goloop",Short: "Goloop CLI",}
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.StringVarP(&cfg.FilePath, "config", "c", "", "Parsing configuration file")
	rootPFlags.StringVarP(&cliSocket, "node_sock", "s", "",
		"Node Command Line Interface socket path(default:[node_dir]/cli.sock)")
	rootPFlags.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	rootPFlags.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")
	vc.BindPFlags(rootPFlags)

	serverCmd := &cobra.Command{Use: "server", Short: "Server management"}
	serverFlags := serverCmd.PersistentFlags()
	serverFlags.StringVar(&flagCfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	serverFlags.StringVar(&flagCfg.P2PListenAddr, "p2p_listen", "", "Listen ip-port of P2P")
	serverFlags.StringVar(&flagCfg.RPCAddr, "rpc_addr", ":9080", "Listen ip-port of JSON-RPC")
	serverFlags.BoolVar(&flagCfg.RPCDump, "rpc_dump", false, "JSON-RPC Request, Response Dump flag")
	serverFlags.StringVar(&eeSocket, "ee_socket", "", "Execution engine socket path")
	serverFlags.StringVar(&keyStoreFile, "key_store", "", "KeyStore file for wallet")
	serverFlags.StringVar(&keyStoreSecret, "key_secret", "", "Secret(password) file for KeyStore")
	serverFlags.StringVar(&flagCfg.KeyStorePass, "key_password", "", "Password for the KeyStore file")
	serverFlags.IntVar(&flagCfg.EEInstances, "ee_instances", 1, "Number of execution engines")
	serverFlags.StringVar(&nodeDir, "node_dir", "",
		"Node data directory(default:[configuration file path]/.chain/[ADDRESS])")
	serverFlags.StringVar(&flagCfg.LogLevel, "log_level", "debug", "Global log level (trace,debug,info,warn,error,fatal,panic)")
	serverFlags.StringVar(&flagCfg.ConsoleLevel, "console_level", "trace", "Console log level (trace,debug,info,warn,error,fatal,panic)")
	vc.BindPFlags(serverFlags)

	saveCmd := &cobra.Command{
		Use:   "save [file]",
		Short: "Save configuration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := makeSureKeyStore(&cfg); err != nil {
				log.Panic(err)
			}
			if saveKeyStore != "" {
				if err := os.MkdirAll(path.Dir(saveKeyStore), 0700); err != nil {
					log.Panicf("Fail to create directory %s err=%+v", saveKeyStore, err)
				}
				ks := bytes.NewBuffer(nil)
				if err := json.Indent(ks, cfg.KeyStoreData, "", "  "); err != nil {
					log.Panicf("Fail to indenting key data err=%+v", err)
				}
				if err := ioutil.WriteFile(saveKeyStore, ks.Bytes(), 0600); err != nil {
					log.Panicf("Fail to save key store to the file=%s err=%+v", saveKeyStore, err)
				}
			}
			if keyStoreSecret != "" {
				cfg.KeyStorePass = ""
			}

			if _, err := log.ParseLevel(cfg.LogLevel); err != nil {
				log.Panicf("Invalid log_level=%s", cfg.LogLevel)
			}

			if _, err := log.ParseLevel(cfg.ConsoleLevel); err != nil {
				log.Panicf("Invalid console_level=%s", cfg.ConsoleLevel)
			}

			saveFilePath := args[0]
			if err := saveJsonFile(saveFilePath, &cfg); err != nil {
				log.Panicf("Fail to save JSON %s err=%+v", saveFilePath, err)
			}
		},
	}
	saveCmd.Flags().StringVar(&saveKeyStore, "save_key_store", "", "KeyStore File path to save")

	serverCmd.AddCommand(saveCmd)

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start server",
	}
	startFlags := startCmd.Flags()
	var modLevels map[string]string
	startFlags.StringToStringVar(&modLevels, "mod_level", nil, "Set log level for specific module (<mod>=<level>,...)")
	startFlags.MarkHidden("mod_level")

	startCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := makeSureKeyStore(&cfg); err != nil {
			log.Panic(err)
		}

		w, err := wallet.NewFromPrivateKey(cfg.priK)
		if err != nil {
			log.Panicf("Fail to create wallet err=%+v", err)
		}

		logger := log.WithFields(log.Fields{
			log.FieldKeyWallet: hex.EncodeToString(w.Address().ID()),
		})
		log.SetGlobalLogger(logger)
		stdlog.SetOutput(logger.WriterLevel(log.WarnLevel))

		if lv, err := log.ParseLevel(cfg.LogLevel); err != nil {
			log.Panicf("Invalid log_level=%s", cfg.LogLevel)
		} else {
			logger.SetLevel(lv)
		}
		if lv, err := log.ParseLevel(cfg.ConsoleLevel); err != nil {
			log.Panicf("Invalid console_level=%s", cfg.ConsoleLevel)
		} else {
			logger.SetConsoleLevel(lv)
		}
		if modLevels != nil {
			for mod, lvStr := range modLevels {
				if lv, err := log.ParseLevel(lvStr); err != nil {
					log.Panicf("Invalid mod_level mod=%s level=%s", mod, lvStr)
				} else {
					logger.SetModuleLevel(mod, lv)
				}
			}
		}

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

		n := node.NewNode(w, &cfg.NodeConfig, logger)
		n.Start()
	}
	serverCmd.AddCommand(startCmd)

	chainCmd := NewChainCmd(&cfg)
	systemCmd := NewSystemCmd(&cfg)
	statsCmd := NewStatsCmd(&cfg)
	rootCmd.AddCommand(serverCmd, chainCmd, systemCmd, statsCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print goloop version",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("goloop version", version, build)
		},
	})

	genMdCmd := NewGenerateMarkdownCommand()
	genMdCmd.Hidden = true
	rootCmd.AddCommand(genMdCmd)
	rootCmd.SilenceUsage = true
	err := rootCmd.Execute()
	if err != nil {
		if restErr, ok := err.(*node.RestError); ok {
			response := restErr.Response()
			if len(response) > 0 {
				rootCmd.Println(response)
			}
		}
		os.Exit(1)
	}
}

// make sure that cfg.KeyStoreData always has valid value to let them
// be stored with --save_key_store option even though the key is
// provided by cfg.Key value.
func makeSureKeyStore(cfg *GoLoopConfig) error {
	if cfg.priK == nil {
		cfg.priK, _ = crypto.GenerateKeyPair()
		log.Println("Generated KeyPair", common.NewAccountAddressFromPublicKey(cfg.priK.PublicKey()).String())
	}
	if len(cfg.KeyStorePass) == 0 {
		cfg.KeyStorePass = DefaultKeyStorePass
	}

	if ks, err := wallet.EncryptKeyAsKeyStore(cfg.priK, []byte(cfg.KeyStorePass)); err != nil {
		return fmt.Errorf("fail to encrypt private key err=%+v", err)
	} else {
		cfg.KeyStoreData = ks
	}
	return nil
}

func saveJsonFile(filePath string, v interface{}) error {
	if err := os.MkdirAll(path.Dir(filePath), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	f.Close()
	log.Println("Save configuration to", filePath)
	return nil
}
