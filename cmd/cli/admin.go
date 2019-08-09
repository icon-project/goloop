package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gosuri/uitable"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/node"
	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminPersistentPreRunE(vc *viper.Viper, adminClient *node.UnixDomainSockHttpClient) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		nodeSock := vc.GetString("node_sock")
		cfgFilePath := vc.GetString("config")
		if nodeSock == "" && cfgFilePath != "" {
			cfg := &ServerConfig{}
			cfg.FilePath,_ = filepath.Abs(cfgFilePath)
			if err := MergeWithViper(vc, cfg); err != nil {
				return err
			}
			if cfg.CliSocket == "" {
				if cfg.priK == nil {
					return errors.Errorf("not exists keyStore on config %s", cfgFilePath)
				}
				addr := common.NewAccountAddressFromPublicKey(cfg.priK.PublicKey())
				cfg.FillEmpty(addr)
			}

			nodeSock = cfg.ResolveAbsolute(cfg.CliSocket)
			vc.Set("node_sock", nodeSock)
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
	pFlags.String("key_secret", "", "Secret(password) file for KeyStore")
	pFlags.String("key_password", "", "Password for the KeyStore file")
	MarkAnnotationCustom(pFlags, "node_sock")
	MarkAnnotationHidden(pFlags, "node_dir","config","key_store","key_secret","key_password")
}

func NewChainCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "chain", "Manage chains")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	rootCmd.AddCommand(&cobra.Command{
		Use:                   "ls",
		Short:                 "List chains",
		Args:                  cobra.ExactArgs(0),
		DisableFlagsInUseLine: true,
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
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			genesisZip, _ := fs.GetString("genesis")
			genesisPath, _ := fs.GetString("genesis_template")
			param := &node.ChainConfig{}
			param.SeedAddr, _ = fs.GetString("seed")
			param.Role, _ = fs.GetUint("role")
			param.DBType, _ = fs.GetString("db_type")
			param.ConcurrencyLevel, _ = fs.GetInt("concurrency")
			param.NormalTxPoolSize, _ = fs.GetInt("normal_tx_pool")
			param.PatchTxPoolSize, _ = fs.GetInt("patch_tx_pool")
			param.MaxBlockTxBytes, _ = fs.GetInt("max_block_tx_bytes")
			param.Channel, _ = fs.GetString("channel")
			param.SecureSuites, _ = fs.GetString("secure_suites")
			param.SecureAeads, _ = fs.GetString("secure_aeads")

			var err error
			var v string
			reqUrl := node.UrlChain
			if len(genesisZip) > 0 {
				file, err2 := os.Open(genesisZip)
				if err2 != nil {
					return errors.Errorf("fail to open %s err=%+v", genesisZip, err2)
				}
				gs, err2 := gs.NewFromFile(file)
				if err2 != nil {
					return errors.Errorf("fail to parse %s err=%+v", genesisZip, err2)
				}
				if _, err2 := gs.NID(); err2 != nil {
					return errors.Errorf("fail to get NID for %s err=%+v", genesisZip, err2)
				}
				_, err = adminClient.PostWithFile(reqUrl, param, "genesisZip", genesisZip, &v)
			} else if len(genesisPath) > 0 {
				buf := bytes.NewBuffer(nil)
				err2 := gs.WriteFromPath(buf, genesisPath)
				if err2 != nil {
					return errors.Errorf("failed WriteGenesisStorage err=%+v", err2)
				}
				gs, err2 := gs.New(buf.Bytes())
				if err2 != nil {
					return errors.Errorf("fail to parse genesis storage err=%+v", err2)
				}
				if _, err2 := gs.NID(); err2 != nil {
					return errors.Errorf("fail to get NID for %s err=%+v", genesisZip, err2)
				}
				_, err = adminClient.PostWithReader(reqUrl, param, "genesisZip", buf, &v)
			} else {
				return errors.Errorf("required flag --genesis or --genesis_template")
			}
			if err != nil {
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
	joinFlags.StringSlice("seed", nil, "Ip-port of Seed")
	joinFlags.Uint("role", 3, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	joinFlags.String("db_type", "goleveldb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	joinFlags.Int("concurrency", 1, "Maximum number of executors to use for concurrency")
	joinFlags.Int("normal_tx_pool", 0, "Size of normal transaction pool")
	joinFlags.Int("patch_tx_pool", 0, "Size of patch transaction pool")
	joinFlags.Int("max_block_tx_bytes", 0, "Max size of transactions in a block")
	joinFlags.String("channel", "", "Channel")
	joinFlags.String("secure_suites", "none,tls,ecdhe",
		"Supported Secure suites with order (none,tls,ecdhe) - Comma separated string")
	joinFlags.String("secure_aeads", "chacha,aes128,aes256",
		"Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string")

	leaveCmd := &cobra.Command{
		Use:                   "leave NID",
		Short:                 "Leave chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
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
		Use:   "inspect NID",
		Short: "Inspect chain",
		Args:  cobra.ExactArgs(1),
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
			Use:                   "start NID",
			Short:                 "Chain start",
			Args:                  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE:                  opFunc("start"),
		},
		&cobra.Command{
			Use:                   "stop NID",
			Short:                 "Chain stop",
			Args:                  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE:                  opFunc("stop"),
		},
		&cobra.Command{
			Use:                   "reset NID",
			Short:                 "Chain data reset",
			Args:                  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE:                  opFunc("reset"),
		},
		&cobra.Command{
			Use:                   "verify NID",
			Short:                 "Chain data verify",
			Args:                  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE:                  opFunc("verify"),
		})

	importCmd := &cobra.Command{
		Use:   "import NID",
		Short: "Start to import legacy database",
		Args:  cobra.ExactArgs(1),
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
	return rootCmd, vc
}

func NewSystemCmd(parentCmd *cobra.Command, parentVc *viper.Viper) (*cobra.Command, *viper.Viper) {
	var adminClient node.UnixDomainSockHttpClient
	rootCmd, vc := NewCommand(parentCmd, parentVc, "system", "System info")
	rootCmd.PersistentPreRunE = AdminPersistentPreRunE(vc, &adminClient)
	AddAdminRequiredFlags(rootCmd)
	BindPFlags(vc, rootCmd.PersistentFlags())

	infoCmd := &cobra.Command{
		Use:                   "info",
		Short:                 "Get system information",
		Args:                  cobra.ExactArgs(0),
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
		Use:                   "config KEY VALUE",
		Short:                 "Configure system",
		Args:                  cobra.ExactArgs(2),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			param := &node.ConfigureParam{
				Key: args[0],
				Value: args[1],
			}
			var v string
			if _, err := adminClient.PostWithJson(node.UrlSystem+"/configure", param, &v); err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(configCmd)

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
