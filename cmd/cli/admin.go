package cli

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/node"
)

func ReadFile(name string) ([]byte, error) {
	if name == "-" {
		if bs, err := ioutil.ReadAll(os.Stdin); err != nil {
			return nil, errors.Wrap(err, "Fail to read stdin")
		} else {
			return bs, nil
		}
	} else {
		if bs, err := ioutil.ReadFile(name); err != nil {
			return nil, errors.Wrapf(err, "Fail to read file=%s", name)
		} else {
			return bs, nil
		}
	}
}

func ReadParam(param string) ([]byte, error) {
	if strings.HasPrefix(param, "@") {
		return ReadFile(param[1:])
	} else {
		return []byte(param), nil
	}
}

func AdminPersistentPreRunE(vc *viper.Viper, adminClient *node.UnixDomainSockHttpClient) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		nodeSock := vc.GetString("node_sock")
		if nodeSock == "" {
			cfg := &ServerConfig{}
			if err := MergeWithViper(vc, cfg); err != nil {
				return err
			}
			if cfg.FilePath != "" {
				if cfg.CliSocket == "" {
					if addr := cfg.GetAddress(); addr != nil {
						cfg.FillEmpty(addr)
					} else {
						return errors.New("unable to decide node directory")
					}
				}
				nodeSock = cfg.ResolveAbsolute(cfg.CliSocket)
				vc.Set("node_sock", nodeSock)
			}
		}
		if err := ValidateFlagsWithViper(vc, cmd.Flags()); err != nil {
			return err
		}
		*adminClient = *node.NewUnixDomainSockHttpClient(nodeSock)
		return nil
	}
}

func AddAdminRequiredFlags(c *cobra.Command) {
	pFlags := c.PersistentFlags()
	pFlags.String("node_dir", "",
		"Node data directory(default:[configuration file path]/.chain/[ADDRESS])")
	pFlags.StringP("node_sock", "s", "",
		"Node Command Line Interface socket path(default:[node_dir]/cli.sock)")
	pFlags.StringP("config", "c", "", "Parsing configuration file")
	pFlags.String("key_store", "", "KeyStore file for wallet")
	MarkAnnotationCustom(pFlags, "node_sock")
}

func NewChainCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "chain", "Manage chains")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List chains",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			l := make([]*node.ChainView, 0)
			reqUrl := node.UrlChain
			resp, err := adminClient.Get(reqUrl, &l)
			if err != nil {
				return err
			}
			if err = JsonPrettyPrintln(os.Stdout, l); err != nil {
				return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
			}
			return nil
		},
	})
	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "Join chain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			genesisZip, _ := fs.GetString("genesis")
			genesisPath, _ := fs.GetString("genesis_template")
			param := &node.ChainConfig{}
			param.SeedAddr, _ = fs.GetString("seed")
			param.Role, _ = fs.GetUint("role")
			param.DBType, _ = fs.GetString("db_type")
			param.Platform, _ = fs.GetString("platform")
			param.ConcurrencyLevel, _ = fs.GetInt("concurrency")
			param.NormalTxPoolSize, _ = fs.GetInt("normal_tx_pool")
			param.PatchTxPoolSize, _ = fs.GetInt("patch_tx_pool")
			param.MaxBlockTxBytes, _ = fs.GetInt("max_block_tx_bytes")
			param.NodeCache, _ = fs.GetString("node_cache")
			param.Channel, _ = fs.GetString("channel")
			param.SecureSuites, _ = fs.GetString("secure_suites")
			param.SecureAeads, _ = fs.GetString("secure_aeads")
			param.DefWaitTimeout, _ = fs.GetInt64("default_wait_timeout")
			param.MaxWaitTimeout, _ = fs.GetInt64("max_wait_timeout")
			param.TxTimeout, _ = fs.GetInt64("tx_timeout")
			param.AutoStart, _ = fs.GetBool("auto_start")
			if fs.Changed("children_limit") {
				childrenLimit, _ := fs.GetInt("children_limit")
				param.ChildrenLimit = &childrenLimit
			}
			if fs.Changed("nephews_limit") {
				nephewsLimit, _ := fs.GetInt("nephews_limit")
				param.NephewsLimit = &nephewsLimit
			}
			param.ValidateTxOnSend, _ = fs.GetBool("validate_tx_on_send")

			var buf *bytes.Buffer
			if len(genesisZip) > 0 {
				b, err := ReadFile(genesisZip)
				if err != nil {
					return err
				}
				buf = bytes.NewBuffer(b)
			} else if len(genesisPath) > 0 {
				buf = bytes.NewBuffer(nil)
				if err := gs.WriteFromPath(buf, genesisPath); err != nil {
					return errors.Errorf("failed WriteGenesisStorage err=%+v", err)
				}
			} else {
				return errors.Errorf("required flag --genesis or --genesis_template")
			}

			if genesisStorage, err := gs.New(buf.Bytes()); err != nil {
				return errors.Errorf("fail to parse genesis storage err=%+v", err)
			} else if _, err = genesisStorage.NID(); err != nil {
				return errors.Errorf("fail to get NID for %s err=%+v", genesisZip, err)
			}

			var v string
			reqUrl := node.UrlChain
			if _, err := adminClient.PostWithReader(reqUrl, param, "genesisZip", buf, &v); err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(joinCmd)
	joinFlags := joinCmd.Flags()
	joinFlags.String("genesis", "", "Genesis storage path")
	joinFlags.String("genesis_template", "", "Genesis template directory or file")
	joinFlags.String("seed", "", "List of trust-seed ip-port, Comma separated string")
	joinFlags.Uint("role", 3, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	joinFlags.String("db_type", "goleveldb", "Name of database system("+strings.Join(db.RegisteredBackendTypes(), ", ")+")")
	joinFlags.String("platform", "", "Name of service platform")
	joinFlags.Int("concurrency", 1, "Maximum number of executors to be used for concurrency")
	joinFlags.Int("normal_tx_pool", 0, "Size of normal transaction pool")
	joinFlags.Int("patch_tx_pool", 0, "Size of patch transaction pool")
	joinFlags.Int("max_block_tx_bytes", 0, "Max size of transactions in a block")
	joinFlags.String("node_cache", chain.NodeCacheDefault, "Node cache (none,small,large)")
	joinFlags.String("channel", "", "Channel")
	joinFlags.String("secure_suites", "none,tls,ecdhe",
		"Supported Secure suites with order (none,tls,ecdhe) - Comma separated string")
	joinFlags.String("secure_aeads", "chacha,aes128,aes256",
		"Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string")
	joinFlags.Int64("default_wait_timeout", 0, "Default wait timeout in milli-second (0: disable)")
	joinFlags.Int64("max_wait_timeout", 0, "Max wait timeout in milli-second (0: uses same value of default_wait_timeout)")
	joinFlags.Int64("tx_timeout", 0, "Transaction timeout in milli-second (0: uses system default value)")
	joinFlags.Bool("auto_start", false, "Auto start")
	joinFlags.Int("children_limit", -1, "Maximum number of child connections (-1: uses system default value)")
	joinFlags.Int("nephews_limit", -1, "Maximum number of nephew connections (-1: uses system default value)")
	joinFlags.Bool("validate_tx_on_send", false, "Validate transaction on send")

	leaveCmd := &cobra.Command{
		Use:   "leave CID",
		Short: "Leave chain",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlChain + "/" + args[0]
			var v string
			_, err := adminClient.Delete(reqUrl, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(leaveCmd)

	inspectCmd := &cobra.Command{
		Use:   "inspect CID",
		Short: "Inspect chain",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cmd.Flag("format").Value.String()
			var v interface{}
			params := &url.Values{}
			if format == "" {
				v = new(node.ChainInspectView)
			} else {
				v = new(string)
				params.Add("format", format)
			}
			if informal, err := cmd.Flags().GetBool("informal"); informal && err == nil {
				params.Add("informal", strconv.FormatBool(informal))
			}
			reqUrl := node.UrlChain + "/" + args[0]
			resp, err := adminClient.Get(reqUrl, v, params)
			if err != nil {
				return err
			}
			if format == "" {
				if err = JsonPrettyPrintln(os.Stdout, v); err != nil {
					return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
				return nil
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
			return nil
		},
	}
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringP("format", "f", "", "Format the output using the given Go template")
	inspectCmd.Flags().Bool("informal", false, "Inspect with informal data")

	opFunc := func(op string) func(cmd *cobra.Command, args []string) error {
		return func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlChain + "/" + args[0] + "/" + op
			var v string
			_, err := adminClient.Post(reqUrl, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		}
	}
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "start CID",
			Short: "Chain start",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE:  opFunc("start"),
		},
		&cobra.Command{
			Use:   "stop CID",
			Short: "Chain stop",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE:  opFunc("stop"),
		},
		&cobra.Command{
			Use:   "verify CID",
			Short: "Chain data verify",
			Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
			RunE:  opFunc("verify"),
		})

	resetCmd := &cobra.Command{
		Use:   "reset CID",
		Short: "Chain data reset",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &node.ChainResetParam{}
			var err error
			fs := cmd.Flags()
			if param.Height, err = fs.GetInt64("height"); err != nil {
				return err
			}
			blockHash := cmd.Flag("block_hash").Value.String()
			if len(blockHash) > 0 {
				if len(blockHash) >= 2 && blockHash[:2] == "0x" {
					blockHash = blockHash[2:]
				}
				if param.BlockHash, err = hex.DecodeString(blockHash); err != nil {
					return err
				}
			}
			if param.Height < 0 {
				return fmt.Errorf("height should be zero or positive value")
			} else if param.Height == 0 && len(blockHash) > 0 {
				return fmt.Errorf("block_hash should be empty value")
			} else if param.Height > 0 && len(blockHash) == 0 {
				return fmt.Errorf("block_hash required")
			}

			var v string
			reqUrl := node.UrlChain + "/" + args[0] + "/reset"
			if _, err = adminClient.PostWithJson(reqUrl, param, &v); err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(resetCmd)
	resetFlags := resetCmd.Flags()
	resetFlags.Int64("height", 0, "Block Height")
	resetFlags.String("block_hash", "", "Hash of the block at the given height, If height is zero, shall be empty")

	importCmd := &cobra.Command{
		Use:   "import CID",
		Short: "Start to import legacy database",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			param := &node.ChainImportParam{}
			param.DBPath, _ = fs.GetString("db_path")
			param.Height, _ = fs.GetInt64("height")

			var v string
			reqUrl := node.UrlChain + "/" + args[0] + "/import"
			_, err := adminClient.PostWithJson(reqUrl, param, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(importCmd)
	importFlags := importCmd.Flags()
	importFlags.String("db_path", "", "Database path")
	importFlags.Int64("height", 0, "Block Height")
	MarkAnnotationRequired(importFlags, "db_path", "height")

	pruneCmd := &cobra.Command{
		Use:   "prune CID",
		Short: "Start to prune the database based on the height",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			param := &node.ChainPruneParam{}
			param.DBType, _ = fs.GetString("db_type")
			param.Height, _ = fs.GetInt64("height")

			var v string
			reqUrl := node.UrlChain + "/" + args[0] + "/prune"
			_, err := adminClient.PostWithJson(reqUrl, param, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(pruneCmd)
	pruneFlags := pruneCmd.Flags()
	pruneFlags.String("db_type", "", "Database type(default:original database type)")
	pruneFlags.Int64("height", 0, "Block Height")
	MarkAnnotationRequired(pruneFlags, "height")

	backupCmd := &cobra.Command{
		Use:   "backup CID",
		Short: "Start to backup the channel",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			manual, _ := fs.GetBool("manual")
			param := &node.ChainBackupParam{
				Manual: manual,
			}
			var v string
			reqUrl := node.UrlChain + "/" + args[0] + "/backup"
			_, err := adminClient.PostWithJson(reqUrl, param, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(backupCmd)
	backupFlags := backupCmd.Flags()
	backupFlags.Bool("manual", false, "Manual backup mode (just release database)")

	genesisCmd := &cobra.Command{
		Use:   "genesis CID FILE",
		Short: "Download chain genesis file",
		Args:  ArgsWithDefaultErrorFunc(OrArgs(cobra.ExactArgs(1), cobra.ExactArgs(2))),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlChain + "/" + args[0] + "/genesis"
			resp, err := adminClient.Get(reqUrl, nil)
			if err != nil {
				return err
			}
			b, fileName, dErr := node.FileDownload(resp)
			if dErr != nil {
				return err
			}
			if len(args) == 2 {
				fileName = args[1]
			}
			err = ioutil.WriteFile(fileName, b, 0644)
			if err != nil {
				return fmt.Errorf("fail to write file err:%+v", err)
			}
			fmt.Println(fileName)
			return nil
		},
	}
	rootCmd.AddCommand(genesisCmd)

	configCmd := &cobra.Command{
		Use:   "config CID KEY VALUE",
		Short: "Configure chain",
		Args:  ArgsWithDefaultErrorFunc(OrArgs(cobra.ExactArgs(1), cobra.ExactArgs(2), cobra.ExactArgs(3))),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlChain + "/" + args[0] + "/configure"
			if len(args) == 1 {
				v := &node.ChainConfig{}
				resp, err := adminClient.Get(reqUrl, v)
				if err != nil {
					return err
				}
				if err = JsonPrettyPrintln(os.Stdout, v); err != nil {
					return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
			} else {
				param := &node.ConfigureParam{
					Key: args[1],
				}
				if len(args) == 2 {
					fs := cmd.Flags()
					param.Value, _ = fs.GetString("value")
					if len(param.Value) == 0 {
						return errors.Errorf("to configure value as empty string, use the third arg with \"\" or ''")
					}
				} else {
					param.Value = args[2]
				}
				var v string
				reqUrl := node.UrlChain + "/" + args[0] + "/configure"
				_, err := adminClient.PostWithJson(reqUrl, param, &v)
				if err != nil {
					return err
				}
				fmt.Println(v)
			}
			return nil
		},
	}
	rootCmd.AddCommand(configCmd)
	configFlags := configCmd.Flags()
	configFlags.String("value", "", "use if value starts with '-'.\n"+
		"(if the third arg is used, this flag will be ignored)")

	rootCmd.Use = "chain TASK CID [PARAM]"
	rootCmd.Args = ArgsWithDefaultErrorFunc(cobra.RangeArgs(2, 3))
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		reqUrl := node.UrlChain + "/" + args[1] + "/" + args[0]
		var param json.RawMessage
		if len(args) == 3 {
			if bs, err := ReadParam(args[2]); err != nil {
				return err
			} else {
				param = bs
			}
		} else {
			param = []byte("{}")
		}
		var v string
		_, err := adminClient.PostWithJson(reqUrl, param, &v)
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil
	}
	return rootCmd, vc
}

func NewSystemCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "system", "System info")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Get system information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cmd.Flag("format").Value.String()
			var v interface{}
			params := &url.Values{}
			if format == "" {
				v = new(node.SystemView)
			} else {
				v = new(string)
				params.Add("format", format)
			}
			resp, err := adminClient.Get(node.UrlSystem, v, params)
			if err != nil {
				return err
			}
			if format == "" {
				if err = JsonPrettyPrintln(os.Stdout, v); err != nil {
					return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
				return nil
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
			return nil
		},
	}
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringP("format", "f", "", "Format the output using the given Go template")

	configCmd := &cobra.Command{
		Use:   "config KEY VALUE",
		Short: "Configure system",
		Args:  ArgsWithDefaultErrorFunc(OrArgs(cobra.ExactArgs(0), cobra.ExactArgs(2))),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlSystem + "/configure"
			if len(args) == 0 {
				v := &node.RuntimeConfig{}
				resp, err := adminClient.Get(reqUrl, v)
				if err != nil {
					return err
				}
				if err = JsonPrettyPrintln(os.Stdout, v); err != nil {
					return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
			} else {
				param := &node.ConfigureParam{
					Key:   args[0],
					Value: args[1],
				}
				var v string
				if _, err := adminClient.PostWithJson(reqUrl, param, &v); err != nil {
					return err
				}
				fmt.Println(v)
			}
			return nil
		},
	}
	rootCmd.AddCommand(configCmd)

	NewBackupCmd(rootCmd, &adminClient)
	NewRestoreCmd(rootCmd, &adminClient)

	return rootCmd, vc
}

func NewBackupCmd(parent *cobra.Command, client *node.UnixDomainSockHttpClient) {
	rootCmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage stored backups",
	}
	parent.AddCommand(rootCmd)

	listCmd := &cobra.Command{
		Use:   "ls",
		Short: "List current backups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := client.Get(node.UrlSystem+"/backup", nil)
			if err != nil {
				return err
			}
			return JsonPrettyCopyAndClose(os.Stdout, resp.Body)
		},
	}
	rootCmd.AddCommand(listCmd)
}

func NewRestoreCmd(parent *cobra.Command, client *node.UnixDomainSockHttpClient) {
	rootCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore chain from a backup",
	}
	parent.AddCommand(rootCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get restore status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := client.Get(node.UrlSystem+"/restore", nil)
			if err != nil {
				return err
			}
			return JsonPrettyCopyAndClose(os.Stdout, resp.Body)
		},
	}
	rootCmd.AddCommand(statusCmd)

	startCmd := &cobra.Command{
		Use:   "start [NAME]",
		Short: "Start to restore the specified backup",
		Args:  cobra.RangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var params node.RestoreBackupParam
			params.Name = args[0]
			params.Overwrite, _ = cmd.PersistentFlags().GetBool("overwrite")
			var v string
			_, err := client.PostWithJson(node.UrlSystem+"/restore", &params, &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	startFlags := startCmd.PersistentFlags()
	startFlags.Bool("overwrite", false, "Overwrite existing chain")
	rootCmd.AddCommand(startCmd)

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop current restoring job",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var v string
			_, err := client.Delete(node.UrlSystem+"/restore", &v)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(stopCmd)
}

func NewUserCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "user", "User management")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			l := make([]string, 0)
			reqUrl := node.UrlUser
			resp, err := adminClient.Get(reqUrl, &l)
			if err != nil {
				return err
			}
			if err = JsonPrettyPrintln(os.Stdout, l); err != nil {
				return errors.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
			}
			return nil
		},
	}, &cobra.Command{
		Use:   "add ADDRESS",
		Short: "Add user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlUser
			param := &struct {
				Id string `json:"id"`
			}{Id: args[0]}
			addr := &common.Address{}
			if err := addr.SetString(param.Id); err != nil {
				return errors.Wrap(err, "invalid Address format")
			}
			var v string
			if _, err := adminClient.PostWithJson(reqUrl, param, &v); err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}, &cobra.Command{
		Use:   "rm ADDRESS",
		Short: "Remove user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqUrl := node.UrlUser + "/" + args[0]
			var v string
			if _, err := adminClient.Delete(reqUrl, &v); err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	})
	return rootCmd, vc
}

const (
	TableCellDisplayNil = "-"
)

var (
	noStream    bool
	intervalSec int
)

func UpdateCuiByStatsViewStream(g *gocui.Gui) node.StreamCallbackFunc {
	return func(respPtr interface{}) error {
		sv := respPtr.(*node.StatsView)
		cuiView, err := g.View("main")
		if err != nil {
			return err
		}
		cuiView.Clear()
		if _, err := fmt.Fprintln(cuiView, sv.Timestamp); err != nil {
			return err
		}
		maxX, _ := cuiView.Size()
		table := StatsViewToTable(sv, uint(maxX))
		if _, err := fmt.Fprint(cuiView, table); err != nil {
			return err
		}

		g.Update(CuiNilUserEvtFunc)
		return nil
	}
}

func StatsViewToTable(v *node.StatsView, maxColWidth uint) *uitable.Table {
	thAlias := []interface{}{
		"Chain",
		"Height",
		"Duration(ms)",
		"TxRequest",
		"TxDrop",
		"TxProcess",
		"TxCommit(ms)",
		"TxFinalize(ms)",
	}
	th := []interface{}{
		"nid",
		"consensus_height",
		"consensus_height_duration",
		"txpool_add_cnt",
		"txpool_drop_cnt",
		"txpool_remove_cnt",
		"txlatency_commit",
		"txlatency_finalize",
	}

	table := uitable.New()
	table.MaxColWidth = maxColWidth
	if len(v.Chains) > 0 {
		table.AddRow(thAlias...)
		for _, c := range v.Chains {
			td := make([]interface{}, 0)
			for _, h := range th {
				tdv := c[h.(string)]
				tds := fmt.Sprint(tdv)
				if tdv == nil {
					tds = TableCellDisplayNil
				}
				td = append(td, tds)
			}
			table.AddRow(td...)
		}
	} else {
		table.AddRow("there is no chain")
	}
	return table
}

func NewStatsCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "stats", "Display a live streams of chains metric-statistics")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	rootPFlags := rootCmd.PersistentFlags()
	rootPFlags.BoolVar(&noStream, "no-stream", false, "Only pull the first metric-statistics")
	rootPFlags.IntVar(&intervalSec, "interval", 1, "Pull interval")
	BindPFlags(vc, rootCmd.PersistentFlags())

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		v := node.StatsView{}
		params := &url.Values{}
		params.Add("interval", fmt.Sprint(intervalSec))

		reqUrl := node.UrlStats
		var err error
		if noStream, _ := cmd.Flags().GetBool("no-stream"); noStream {
			params.Add("stream", "false")
			_, err = adminClient.Get(reqUrl, &v, params)
			if err != nil {
				return err
			}
			fmt.Println(v.Timestamp)
			table := StatsViewToTable(&v, 50)
			fmt.Println(table)
		} else {
			g, guiTermCh := NewCui()
			defer TermGui(g, guiTermCh)
			_, err = adminClient.Stream(reqUrl, nil, &v, UpdateCuiByStatsViewStream(g), guiTermCh, params)
			if err != nil && err != io.EOF {
				return err
			}
		}
		return nil
	}
	return rootCmd, vc
}
