package cli

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	stdlog "log"
	"os"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/node"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	node.Config

	Key           []byte          `json:"key,omitempty"`
	KeyStoreData  json.RawMessage `json:"key_store"`
	KeyStorePass  string          `json:"key_password,omitempty"`
	isPresentPass bool
	priK          *crypto.PrivateKey

	LogLevel     string `json:"log_level"`
	ConsoleLevel string `json:"console_level"`
}

const (
	DefaultKeyStorePass = "gochain"
)

func NewServerCmd(parentCmd *cobra.Command, parentVc *viper.Viper, version, build string, logoLines []string) (*cobra.Command, *viper.Viper) {
	rootCmd, vc := NewCommand(parentCmd, parentVc, "server", "Server management")

	cfg := &ServerConfig{}
	cfg.BuildVersion = version
	cfg.BuildTags = build

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cfg.FilePath = vc.GetString("config")
		if err := MergeWithViper(vc, cfg); err != nil {
			return err
		}
		if cfg.priK == nil {
			cfg.priK, _ = crypto.GenerateKeyPair()
			log.Println("Generated KeyPair", common.NewAccountAddressFromPublicKey(cfg.priK.PublicKey()).String())
			if len(cfg.KeyStorePass) == 0 {
				cfg.KeyStorePass = DefaultKeyStorePass
			}
		}
		// make sure that cfg.KeyStoreData always has valid value to let them
		// be stored with --save_key_store option even though the key is
		// provided by cfg.Key value.
		if ks, err := wallet.EncryptKeyAsKeyStore(cfg.priK, []byte(cfg.KeyStorePass)); err != nil {
			return errors.Errorf("fail to encrypt private key err=%+v", err)
		} else {
			cfg.KeyStoreData = ks
		}
		return nil
	}
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.String("p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	rootPFlags.String("p2p_listen", "", "Listen ip-port of P2P")
	rootPFlags.String("rpc_addr", ":9080", "Listen ip-port of JSON-RPC")
	rootPFlags.Bool("rpc_dump", false, "JSON-RPC Request, Response Dump flag")
	rootPFlags.String("rpc_default_channel", "", "JSON-RPC Default Channel")
	rootPFlags.String("ee_socket", "", "Execution engine socket path")
	rootPFlags.String("key_password", "", "Password for the KeyStore file")
	rootPFlags.Int("ee_instances", 1, "Number of execution engines")
	rootPFlags.String("log_level", "debug", "Global log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("console_level", "trace", "Console log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("node_dir", "",
		"Node data directory(default:[configuration file path]/.chain/[ADDRESS])")
	rootPFlags.StringP("node_sock", "s", "",
		"Node Command Line Interface socket path(default:[node_dir]/cli.sock)")
	rootPFlags.StringP("config", "c", "", "Parsing configuration file")
	//
	rootPFlags.String("key_store", "", "KeyStore file for wallet")
	rootPFlags.String("key_secret", "", "Secret(password) file for KeyStore")
	BindPFlags(vc, rootCmd.PersistentFlags())

	saveCmd := &cobra.Command{
		Use:   "save [file]",
		Short: "Save configuration",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if cfg.isPresentPass {
				cfg.KeyStorePass = ""
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			saveFilePath := args[0]
			if err := JsonPrettySaveFile(saveFilePath, 0644, cfg); err != nil {
				return err
			}
			log.Println("Save configuration to", saveFilePath)

			if saveKeyStore, _ := cmd.Flags().GetString("save_key_store"); saveKeyStore != "" {
				if err := JsonPrettySaveFile(saveKeyStore, 0600, cfg.KeyStoreData); err != nil {
					return err
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().String("save_key_store", "", "KeyStore File path to save")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start server",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			modLevels, _ := cmd.Flags().GetStringToString("mod_level")
			for mod, lvStr := range modLevels {
				if lv, err := log.ParseLevel(lvStr); err != nil {
					log.Panicf("Invalid mod_level mod=%s level=%s", mod, lvStr)
				} else {
					logger.SetModuleLevel(mod, lv)
				}
			}

			if cpuProfile := vc.GetString("cpuprofile"); cpuProfile != "" {
				if err := StartCPUProfile(cpuProfile); err != nil {
					log.Fatalf(err.Error())
				}
			}
			if memProfile := vc.GetString("memprofile"); memProfile != "" {
				if err := StartMemoryProfile(memProfile); err != nil {
					log.Fatalf(err.Error())
				}
			}


			for _, l := range logoLines {
				log.Println(l)
			}
			log.Printf("Version : %s", version)
			log.Printf("Build   : %s", build)

			n := node.NewNode(w, &cfg.Config, logger)
			n.Start()
			return nil
		},
	}
	rootCmd.AddCommand(startCmd)
	startFlags := startCmd.Flags()
	startFlags.StringToString("mod_level", nil, "Set console log level for specific module ('mod'='level',...)")
	startFlags.String("cpuprofile", "", "CPU Profiling data file")
	startFlags.String("memprofile", "", "Memory Profiling data file")
	startFlags.MarkHidden("mod_level")
	BindPFlags(vc, rootCmd.PersistentFlags())

	return rootCmd, vc
}

func MergeWithViper(vc *viper.Viper, cfg *ServerConfig) error {
	if vc.GetString("key_secret") != "" || vc.GetString("key_password") != "" {
		cfg.isPresentPass = true
	}
	//relative path from flag, env
	nodeDir := vc.GetString("node_dir")
	cliSocket := vc.GetString("node_sock")
	eeSocket := vc.GetString("ee_socket")

	if cfg.FilePath != "" {
		log.Printf("Read config file=%s", cfg.FilePath)
		f, err := os.Open(cfg.FilePath)
		if err != nil {
			return errors.Errorf("fail to open config file=%s err=%+v", cfg.FilePath, err)
		}
		vc.SetConfigType("json")
		err = vc.ReadConfig(f)
		if err != nil {
			return errors.Errorf("fail to read config file=%s err=%+v", cfg.FilePath, err)
		}
	}

	if err := vc.Unmarshal(cfg, ViperDecodeOptJson); err != nil {
		return errors.Errorf("fail to unmarshall server config from env err=%+v", err)
	}
	if err := vc.Unmarshal(&cfg.Config, ViperDecodeOptJson); err != nil {
		return errors.Errorf("fail to unmarshall node config from env err=%+v", err)
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

	//config.KeyStorePass
	//overwrite env.KeyStorePass
	//overwrite flag.KeyStorePass
	//overwrite read(env.KeyStoreSecret)
	//overwrite read(flag.KeyStoreSecret)
	if keyStoreSecret := vc.GetString("key_secret"); keyStoreSecret != "" {
		if ksp, err := ioutil.ReadFile(keyStoreSecret); err != nil {
			return errors.Errorf("fail to open KeySecret file=%s err=%+v", keyStoreSecret, err)
		} else {
			cfg.KeyStorePass = strings.TrimSpace(string(ksp))
		}
	}

	//config.priK
	//crypto.GenerateKeyPair()
	//parse config.Key
	//overwrite config.KeyStoreData
	//overwrite read(env.KeyStore)
	//overwrite read(flag.KeyStore)
	var err error
	if len(cfg.KeyStoreData) > 0 {
		if cfg.KeyStorePass == "" {
			return errors.Errorf("there is no password information for the KeyStore")
		}
		if cfg.priK, err = wallet.DecryptKeyStore(cfg.KeyStoreData, []byte(cfg.KeyStorePass)); err != nil {
			return errors.Errorf("fail to decrypt KeyStore err=%+v", err)
		}
	} else if len(cfg.Key) > 0 {
		if cfg.priK, err = crypto.ParsePrivateKey(cfg.Key); err != nil {
			return errors.Errorf("illegal key data=[%x]", cfg.Key)
		}
	}
	return nil
}
