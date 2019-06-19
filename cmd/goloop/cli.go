package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gosuri/uitable"
	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/node"
)

var (
	genesisZip, genesisPath string
	joinChainParam          node.JoinChainParam
)

func JsonIntend(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", nil
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, b, "", "  ")
	if err != nil {
		return "", nil
	}
	return string(buf.Bytes()), nil
}

func GetUnixDomainSockHttpClient(cfg *GoLoopConfig) *node.UnixDomainSockHttpClient {
	if cfg.CliSocket == "" {
		if cfg.priK == nil {
			log.Panicf("fail to using default CliSocket")
		}
		addr := common.NewAccountAddressFromPublicKey(cfg.priK.PublicKey())
		cfg.FillEmpty(addr)
	}
	cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
	return node.NewUnixDomainSockHttpClient(cliSocket)
}

func NewChainCmd(cfg *GoLoopConfig) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "chain",
		Short: "Manage chains",
		Args:  cobra.MinimumNArgs(1),
	}
	rootCmd.DisableFlagsInUseLine = true
	rootCmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List chains",
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			l := make([]*node.ChainView, 0)
			reqUrl := node.UrlChain
			resp, err := hc.Get(reqUrl, &l)
			if err != nil {
				return fmt.Errorf("failed GET %s resp=%+v, err=%+v", reqUrl, resp, err)
			}
			s, err := JsonIntend(l)
			if err != nil {
				return fmt.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
			}
			fmt.Println(s)
			return nil
		},
	})
	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "Join chain",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			var err error
			var v string
			var resp *http.Response
			reqUrl := node.UrlChain
			if len(genesisZip) > 0 {
				file, err2 := os.Open(genesisZip)
				if err2 != nil {
					return fmt.Errorf("fail to open %s err=%+v", genesisZip, err2)
				}
				gs, err2 := chain.NewGenesisStorageFromFile(file)
				if err2 != nil {
					return fmt.Errorf("fail to parse %s err=%+v", genesisZip, err2)
				}
				nid, err2 := gs.NID()
				if err2 != nil {
					return fmt.Errorf("fail to get NID for %s err=%+v", genesisZip, err2)
				}
				joinChainParam.NID.Value = int32(nid)
				resp, err = hc.PostWithFile(reqUrl, &joinChainParam, "genesisZip", genesisZip, &v)
			} else if len(genesisPath) > 0 {
				buf := bytes.NewBuffer(nil)
				err = chain.WriteGenesisStorageFromPath(buf, genesisPath)
				if err != nil {
					return fmt.Errorf("failed WriteGenesisStorage err=%+v", err)
				}
				gs, err := chain.NewGenesisStorage(buf.Bytes())
				if err != nil {
					return fmt.Errorf("fail to parse %s err=%+v", genesisZip, err)
				}
				var nid int
				nid, err = gs.NID()
				if err != nil {
					return fmt.Errorf("fail to get NID for %s err=%+v", genesisZip, err)
				}
				joinChainParam.NID.Value = int32(nid)
				resp, err = hc.PostWithReader(reqUrl, &joinChainParam, "genesisZip", buf, &v)
			} else {
				return fmt.Errorf("required flag --genesis or --genesis_template")
			}
			if err != nil {
				return fmt.Errorf("failed POST %s param=%+v, resp=%+v, err=%+v", reqUrl, joinChainParam, resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	joinCmd.Flags().StringVar(&genesisZip, "genesis", "", "Genesis storage path")
	joinCmd.Flags().StringVar(&genesisPath, "genesis_template", "", "Genesis template directory or file")
	joinCmd.Flags().StringVar(&joinChainParam.SeedAddr, "seed", "", "Ip-port of Seed")
	joinCmd.Flags().UintVar(&joinChainParam.Role, "role", 3, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	joinCmd.Flags().StringVar(&joinChainParam.DBType, "db_type", "goleveldb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	joinCmd.Flags().IntVar(&joinChainParam.ConcurrencyLevel, "concurrency", 1, "Maximum number of executors to use for concurrency")
	joinCmd.Flags().StringVar(&joinChainParam.Channel, "channel", "", "Channel")
	joinCmd.Flags().StringVar(&joinChainParam.SecureSuites, "secure_suites", "none,tls,ecdhe",
		"Supported Secure suites with order (none,tls,ecdhe) - Comma separated string")
	joinCmd.Flags().StringVar(&joinChainParam.SecureAeads, "secure_aeads", "chacha,aes128,aes256",
		"Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string")

	leaveCmd := &cobra.Command{
		Use:                   "leave NID",
		Short:                 "Leave chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			reqUrl := node.UrlChain + "/" + args[0]
			var v string
			resp, err := hc.Delete(reqUrl, &v)
			if err != nil {
				return fmt.Errorf("failed DELETE %s resp=%+v, err=%+v", node.UrlChain+"/"+args[0], resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(joinCmd, leaveCmd)
	inspectCmd := &cobra.Command{
		Use:                   "inspect NID",
		Short:                 "Inspect chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
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
			resp, err := hc.Get(reqUrl, v, params)
			if err != nil {
				return fmt.Errorf("failed GET %s param=%+v, resp=%+v, err=%+v", reqUrl, params, resp, err)
			}
			if format == "" {
				s, err := JsonIntend(v)
				if err != nil {
					return fmt.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
				fmt.Println(s)
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
			return nil
		},
	}
	inspectCmd.Flags().StringP("format", "f", "", "Format the output using the given Go template")
	rootCmd.AddCommand(inspectCmd)
	startCmd := &cobra.Command{
		Use:                   "start NID",
		Short:                 "Chain start",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			reqUrl := node.UrlChain + "/" + args[0] + "/start"
			var v string
			resp, err := hc.Post(reqUrl, &v)
			if err != nil {
				return fmt.Errorf("failed POST %s resp=%+v, err=%+v", reqUrl, resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(startCmd)
	stopCmd := &cobra.Command{
		Use:                   "stop NID",
		Short:                 "Chain stop",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			reqUrl := node.UrlChain + "/" + args[0] + "/stop"
			var v string
			resp, err := hc.Post(reqUrl, &v)
			if err != nil {
				return fmt.Errorf("failed POST %s resp=%+v, err=%+v", reqUrl, resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(stopCmd)
	resetCmd := &cobra.Command{
		Use:                   "reset NID",
		Short:                 "Chain data reset",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			reqUrl := node.UrlChain + "/" + args[0] + "/reset"
			var v string
			resp, err := hc.Post(reqUrl, &v)
			if err != nil {
				return fmt.Errorf("failed POST %s resp=%+v, err=%+v", reqUrl, resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(resetCmd)
	verifyCmd := &cobra.Command{
		Use:                   "verify NID",
		Short:                 "Chain data verify",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			reqUrl := node.UrlChain + "/" + args[0] + "/verify"
			var v string
			resp, err := hc.Post(reqUrl, &v)
			if err != nil {
				return fmt.Errorf("failed POST %s resp=%+v, err=%+v", reqUrl, resp, err)
			}
			fmt.Println(v)
			return nil
		},
	}
	rootCmd.AddCommand(verifyCmd)
	return rootCmd
}

func NewSystemCmd(cfg *GoLoopConfig) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "system",
		Short:                 "System info",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			format := cmd.Flag("format").Value.String()
			var v interface{}
			params := &url.Values{}
			if format == "" {
				v = new(node.SystemView)
			} else {
				v = new(string)
				params.Add("format", format)
			}
			reqUrl := node.UrlSystem
			resp, err := hc.Get(node.UrlSystem, v, params)
			if err != nil {
				return fmt.Errorf("failed GET %s param=%+v, resp=%+v, err=%+v", reqUrl, params, resp, err)
			}
			if format == "" {
				s, err := JsonIntend(v)
				if err != nil {
					return fmt.Errorf("failed JsonIntend resp=%+v, err=%+v", resp, err)
				}
				fmt.Println(s)
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
			return nil
		},
	}
	rootCmd.Flags().StringP("format", "f", "", "Format the output using the given Go template")
	return rootCmd
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

func NewStatsCmd(cfg *GoLoopConfig) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "stats",
		Short:                 "Display a live streams of chains metric-statistics",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			hc := GetUnixDomainSockHttpClient(cfg)
			v := node.StatsView{}
			params := &url.Values{}
			params.Add("interval", fmt.Sprint(intervalSec))

			var resp *http.Response
			reqUrl := node.UrlStats
			var err error
			if noStream {
				params.Add("stream", "false")
				resp, err = hc.Get(reqUrl, &v, params)
				if err != nil {
					return fmt.Errorf("failed GET %s param=%+v, resp=%+v, err=%+v", reqUrl, params, resp, err)
				}
				fmt.Println(v.Timestamp)
				table := StatsViewToTable(&v, 50)
				fmt.Println(table)
			} else {
				g, guiTermCh := NewCui()
				resp, err = hc.Stream(reqUrl, nil, &v, UpdateCuiByStatsViewStream(g), guiTermCh, params)
				if err != nil && err != io.EOF {
					return fmt.Errorf("failed Stream %s param=%+v, resp=%+v, err=%+v", reqUrl, params, resp, err)
				}
				TermGui(g, guiTermCh)
			}
			return nil
		},
	}
	rootCmd.Flags().BoolVar(&noStream, "no-stream", false, "Only pull the first metric-statistics")
	rootCmd.Flags().IntVar(&intervalSec, "interval", 1, "Pull interval")
	return rootCmd
}
