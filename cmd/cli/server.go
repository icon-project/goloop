package cli

import (
	"encoding/hex"
	"encoding/json"
	stdlog "log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/node"
)

type ServerConfig struct {
	node.StaticConfig

	KeyStoreData  json.RawMessage `json:"key_store,omitempty"`
	KeyStorePass  string          `json:"key_password,omitempty"`
	isPresentPass bool

	KeyPlugin     string            `json:"key_plugin,omitempty"`
	KeyPlgOptions map[string]string `json:"key_plugin_options,omitempty"`

	Wallet module.Wallet `json:"-"`

	LogLevel     string               `json:"log_level"`
	ConsoleLevel string               `json:"console_level"`
	LogForwarder *log.ForwarderConfig `json:"log_forwarder,omitempty"`
	LogWriter    *log.WriterConfig    `json:"log_writer,omitempty"`
}

func (cfg *ServerConfig) GetAddress() module.Address {
	if cfg.Wallet != nil {
		return cfg.Wallet.Address()
	}
	if len(cfg.KeyStoreData) > 0 {
		if addr, err := wallet.ReadAddressFromKeyStore(cfg.KeyStoreData); err == nil {
			return addr
		}
	}
	return nil
}

func (cfg *ServerConfig) MakesureWallet(gen bool) error {
	if cfg.Wallet != nil {
		return nil
	}
	if cfg.KeyPlugin != "" {
		options := make(map[string]string)
		for k, v := range cfg.KeyPlgOptions {
			options[k] = v
		}
		if _, ok := options["password"]; !ok {
			options["password"] = cfg.KeyStorePass
		}
		if w, err := wallet.OpenPlugin(cfg.KeyPlugin, options); err != nil {
			return err
		} else {
			cfg.Wallet = w
			return nil
		}
	}

	var privateKey *crypto.PrivateKey
	if len(cfg.KeyStoreData) > 0 {
		pass := cfg.KeyStorePass
		if pass == "" {
			pass = DefaultKeyStorePass
		}
		if pk, err := wallet.DecryptKeyStore(cfg.KeyStoreData, []byte(pass)); err != nil {
			return errors.Errorf("fail to decrypt KeyStore err=%+v", err)
		} else {
			privateKey = pk
		}
	}

	if privateKey == nil {
		if !gen {
			return errors.New("Fail to restore KeyStore")
		}
		privateKey, _ = crypto.GenerateKeyPair()
		if len(cfg.KeyStorePass) == 0 {
			cfg.KeyStorePass = DefaultKeyStorePass
		}
	}
	// make sure that cfg.KeyStoreData always has valid value to let them
	// be stored with --save_key_store option even though the key is
	// provided by cfg.Key value.
	if ks, err := wallet.EncryptKeyAsKeyStore(privateKey, []byte(cfg.KeyStorePass)); err != nil {
		return errors.Errorf("fail to encrypt private key err=%+v", err)
	} else {
		cfg.KeyStoreData = ks
	}

	if w, err := wallet.NewFromPrivateKey(privateKey); err != nil {
		return err
	} else {
		cfg.Wallet = w
	}
	return nil
}

func (cfg *ServerConfig) SetFilePath(path string) string {
	o := cfg.StaticConfig.SetFilePath(path)
	if cfg.LogWriter != nil && cfg.LogWriter.Filename != "" {
		cfg.LogWriter.Filename = cfg.ResolveRelative(node.ResolveAbsolute(o, cfg.LogWriter.Filename))
	}
	return o
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
		if err := MergeWithViper(vc, cfg); err != nil {
			return err
		}
		if err := cfg.MakesureWallet(true); err != nil {
			return err
		}
		return nil
	}
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.String("p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	rootPFlags.String("p2p_listen", "", "Listen ip-port of P2P")
	rootPFlags.String("rpc_addr", ":9080", "Listen ip-port of JSON-RPC")
	rootPFlags.Bool("rpc_dump", false, "JSON-RPC Request, Response Dump flag")
	rootPFlags.String("ee_socket", "", "Execution engine socket path")
	rootPFlags.String("key_password", "", "Password for the KeyStore file")
	rootPFlags.String("log_level", "debug", "Global log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("console_level", "trace", "Console log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("node_dir", "",
		"Node data directory (default: [configuration file path]/.chain/[ADDRESS])")
	rootPFlags.StringP("node_sock", "s", "",
		"Node Command Line Interface socket path (default: [node_dir]/cli.sock)")
	rootPFlags.String("backup_dir", "",
		"Node backup directory (default: [node_dir]/backup")
	rootPFlags.StringP("config", "c", "", "Parsing configuration file")
	//
	rootPFlags.String("key_store", "", "KeyStore file for wallet")
	rootPFlags.String("key_secret", "", "Secret (password) file for KeyStore")
	rootPFlags.String("key_plugin", "", "KeyPlugin file for wallet")
	rootPFlags.StringToString("key_plugin_options", nil, "KeyPlugin options")
	//
	rootPFlags.String("log_forwarder_vendor", "", "LogForwarder vendor (fluentd,logstash)")
	rootPFlags.String("log_forwarder_address", "", "LogForwarder address")
	rootPFlags.String("log_forwarder_level", "info", "LogForwarder level")
	rootPFlags.String("log_forwarder_name", "", "LogForwarder name")
	rootPFlags.StringToString("log_forwarder_options", nil, "LogForwarder options, comma-separated 'key=value'")
	rootPFlags.String("engines", "python", "Execution engines, comma-separated (python,java)")

	rootPFlags.String("log_writer_filename", "", "Log filename (rotated files resides in same directory)")
	rootPFlags.Int("log_writer_maxsize", 100, "Maximum log file size in MiB")
	rootPFlags.Int("log_writer_maxage", 0, "Maximum age of log file in day")
	rootPFlags.Int("log_writer_maxbackups", 0, "Maximum number of backups")
	rootPFlags.Bool("log_writer_localtime", false, "Use localtime on rotated log file instead of UTC")
	rootPFlags.Bool("log_writer_compress", false, "Use gzip on rotated log file")

	BindPFlags(vc, rootCmd.PersistentFlags())

	saveCmd := &cobra.Command{
		Use:   "save [file]",
		Short: "Save configuration",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		PreRun: func(cmd *cobra.Command, args []string) {
			if cfg.isPresentPass {
				cfg.KeyStorePass = ""
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			saveFilePath := args[0]
			cfg.SetFilePath(saveFilePath)
			if err := JsonPrettySaveFile(saveFilePath, 0644, cfg); err != nil {
				return err
			}
			stdlog.Println("Save configuration to", saveFilePath)

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
			logger := log.WithFields(log.Fields{
				log.FieldKeyWallet: hex.EncodeToString(cfg.GetAddress().ID()),
			})
			log.SetGlobalLogger(logger)
			stdlog.SetOutput(logger.WriterLevel(log.WarnLevel))
			if cfg.LogWriter != nil {
				var lwCfg log.WriterConfig
				lwCfg = *cfg.LogWriter
				lwCfg.Filename = cfg.ResolveAbsolute(lwCfg.Filename)
				writer, err := log.NewWriter(&lwCfg)
				if err != nil {
					log.Panicf("Fail to make writer err=%+v", err)
				}
				err = logger.SetFileWriter(writer)
				if err != nil {
					log.Panicf("Fail to set file logger err=%+v", err)
				}
			}

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

			if cfg.LogForwarder != nil {
				if err := log.AddForwarder(cfg.LogForwarder); err != nil {
					log.Fatalf("Invalid log_forwarder err:%+v", err)
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

			if blockProfile := vc.GetString("blockprofile"); blockProfile != "" {
				rate := vc.GetInt("blockprofilerate")
				if err := StartBlockProfile(blockProfile, rate); err != nil {
					log.Fatalf(err.Error())
				}
			}

			for _, l := range logoLines {
				log.Println(l)
			}
			log.Printf("Version : %s", version)
			log.Printf("Build   : %s", build)

			n := node.NewNode(cfg.Wallet, &cfg.StaticConfig, logger)
			n.Start()
			return nil
		},
	}
	rootCmd.AddCommand(startCmd)
	startFlags := startCmd.Flags()
	startFlags.StringToString("mod_level", nil, "Set console log level for specific module ('mod'='level',...)")
	startFlags.String("cpuprofile", "", "CPU Profiling data file")
	startFlags.String("memprofile", "", "Memory Profiling data file")
	startFlags.String("blockprofile", "", "Block Profiling data file")
	startFlags.Int("blockprofilerate", 1, "Block Profiling rate in ns")
	startFlags.Bool("auth_skip_if_empty_users", false, "Skip admin API authentication if empty users")
	startFlags.Bool("nid_for_p2p", false, "Use NID instead of CID for p2p network")
	startFlags.MarkHidden("mod_level")
	startFlags.MarkHidden("auth_skip_if_empty_users")
	startFlags.MarkHidden("nid_for_p2p")

	BindPFlags(vc, startFlags)

	return rootCmd, vc
}

func MergeWithViper(vc *viper.Viper, cfg *ServerConfig) error {
	if vc.GetString("key_secret") != "" || vc.GetString("key_password") != "" {
		cfg.isPresentPass = true
	}
	cfgFilePath := vc.GetString("config")
	//relative path from flag, env
	nodeDir := vc.GetString("node_dir")
	cliSocket := vc.GetString("node_sock")
	eeSocket := vc.GetString("ee_socket")
	backupDir := vc.GetString("backup_dir")
	lwFilename := vc.GetString("log_writer_filename")

	if cfgFilePath != "" {
		cfg.SetFilePath(cfgFilePath)
		f, err := os.Open(cfgFilePath)
		if err != nil {
			return errors.Errorf("fail to open config file=%s err=%+v", cfg.FilePath, err)
		}
		vc.SetConfigType("json")
		err = vc.ReadConfig(f)
		if err != nil {
			return errors.Errorf("fail to read config file=%s err=%+v", cfg.FilePath, err)
		}
		if lfVc := vc.Sub("log_forwarder"); lfVc != nil {
			m := make(map[string]interface{})
			for _, k := range lfVc.AllKeys() {
				m["log_forwarder_"+k] = lfVc.Get(k)
			}
			if err := vc.MergeConfigMap(m); err != nil {
				return errors.Errorf("fail to merge config file=%s err=%+v", cfg.FilePath, err)
			}
		}
		if lfVc := vc.Sub("log_writer"); lfVc != nil {
			m := make(map[string]interface{})
			for _, k := range lfVc.AllKeys() {
				m["log_writer_"+k] = lfVc.Get(k)
			}
			if err := vc.MergeConfigMap(m); err != nil {
				return errors.Errorf("fail to merge config file=%s err=%+v", cfg.FilePath, err)
			}
		}
	}

	if err := vc.Unmarshal(cfg, ViperDecodeOptJson); err != nil {
		return errors.Errorf("fail to unmarshall server config from env err=%+v", err)
	}
	if err := vc.Unmarshal(&cfg.StaticConfig, ViperDecodeOptJson); err != nil {
		return errors.Errorf("fail to unmarshall node config from env err=%+v", err)
	}

	var lfOpts map[string]interface{}
	switch v := vc.Get("log_forwarder_options").(type) {
	case string:
		if m, err := stringToStringConv(v); err != nil {
			return errors.Errorf("fail to stringToStringConv config from env err=%+v", err)
		} else {
			lfOpts = m
		}
	case map[string]interface{}:
		lfOpts = v
	}
	if cfg.LogForwarder != nil && cfg.LogForwarder.Options != nil {
		for k, v := range lfOpts {
			cfg.LogForwarder.Options[k] = v
		}
		lfOpts = cfg.LogForwarder.Options
	}
	lfCfg := &log.ForwarderConfig{
		Vendor:  vc.GetString("log_forwarder_vendor"),
		Address: vc.GetString("log_forwarder_address"),
		Level:   vc.GetString("log_forwarder_level"),
		Name:    vc.GetString("log_forwarder_name"),
		Options: lfOpts,
	}
	if lfCfg.Vendor != "" {
		cfg.LogForwarder = lfCfg
	}

	lwCfg := &log.WriterConfig{
		Filename:   vc.GetString("log_writer_filename"),
		MaxSize:    vc.GetInt("log_writer_maxsize"),
		MaxAge:     vc.GetInt("log_writer_maxage"),
		MaxBackups: vc.GetInt("log_writer_maxbackups"),
		LocalTime:  vc.GetBool("log_writer_localtime"),
		Compress:   vc.GetBool("log_writer_compress"),
	}
	if len(lwFilename) > 0 {
		lwCfg.Filename = cfg.ResolveRelative(lwFilename)
	}
	if len(lwCfg.Filename) > 0 {
		cfg.LogWriter = lwCfg
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
	if backupDir != "" {
		cfg.BackupDir = cfg.ResolveRelative(backupDir)
	}

	//config.KeyStorePass
	//overwrite env.KeyStorePass
	//overwrite flag.KeyStorePass
	//overwrite read(env.KeyStoreSecret)
	//overwrite read(flag.KeyStoreSecret)
	if keyStoreSecret := vc.GetString("key_secret"); keyStoreSecret != "" {
		if ksp, err := os.ReadFile(keyStoreSecret); err != nil {
			return errors.Errorf("fail to open KeySecret file=%s err=%+v", keyStoreSecret, err)
		} else {
			cfg.KeyStorePass = string(ksp)
		}
	}

	return nil
}
