package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"

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
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			l := make([]*node.ChainView, 0)
			resp, err := hc.Get(node.UrlChain, &l)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(l)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	})
	joinCmd := &cobra.Command{
		Use:   "join NID",
		Short: "Join chain",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			var err error
			var NID int64
			if NID, err = strconv.ParseInt(args[0], 16, 64); err != nil {
				fmt.Println("cannot parse NID", err)
				return
			}
			joinChainParam.NID = int(NID)
			var resp *http.Response

			if len(genesisZip) > 0 {
				resp, err = hc.PostWithFile(node.UrlChain, &joinChainParam, "genesisZip", genesisZip)
			} else if len(genesisPath) > 0 {
				buf := bytes.NewBuffer(nil)
				err = chain.WriteGenesisStorageFromPath(buf, genesisPath)
				if err != nil {
					fmt.Println(err)
					return
				}
				resp, err = hc.PostWithReader(node.UrlChain, &joinChainParam, "genesisZip", buf)
			} else {
				fmt.Println("There is no genesis")
				return
			}

			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	joinCmd.Flags().StringVar(&genesisZip, "genesis", "", "Genesis storage path")
	joinCmd.Flags().StringVar(&genesisPath, "genesis_template", "", "Genesis template directory or file")
	joinCmd.Flags().StringVar(&joinChainParam.SeedAddr, "seed", "", "Ip-port of Seed")
	joinCmd.Flags().UintVar(&joinChainParam.Role, "role", 3, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	joinCmd.Flags().StringVar(&joinChainParam.DBType, "db_type", "goleveldb", "Name of database system(*badgerdb, goleveldb, boltdb, mapdb)")
	joinCmd.Flags().IntVar(&joinChainParam.ConcurrencyLevel, "concurrency", 1, "Maximum number of executors to use for concurrency")

	leaveCmd := &cobra.Command{
		Use:                   "leave NID",
		Short:                 "Leave chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			resp, err := hc.Delete(node.UrlChain + "/" + args[0])
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	rootCmd.AddCommand(joinCmd, leaveCmd)
	inspectCmd := &cobra.Command{
		Use:                   "inspect NID",
		Short:                 "Inspect chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			format := cmd.Flag("format").Value.String()
			var v interface{}
			params := &url.Values{}
			if format == "" {
				v = new(node.ChainInspectView)
			}else{
				v = new(string)
				params.Add("format", format)
			}
			resp, err := hc.Get(node.UrlChain+"/"+args[0], v, params)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			if format == "" {
				s, err := JsonIntend(v)
				if err != nil {
					fmt.Println(err, resp)
					return
				}
				fmt.Println(s)
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
		},
	}
	inspectCmd.Flags().StringP("format", "f", "", "Format the output using the given Go template")
	rootCmd.AddCommand(inspectCmd)
	startCmd := &cobra.Command{
		Use:                   "start NID",
		Short:                 "Chain start",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			resp, err := hc.Post(node.UrlChain + "/" + args[0] + "/start")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	rootCmd.AddCommand(startCmd)
	stopCmd := &cobra.Command{
		Use:                   "stop NID",
		Short:                 "Chain stop",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			resp, err := hc.Post(node.UrlChain + "/" + args[0] + "/stop")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	rootCmd.AddCommand(stopCmd)
	resetCmd := &cobra.Command{
		Use:                   "reset NID",
		Short:                 "Chain data reset",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			resp, err := hc.Post(node.UrlChain + "/" + args[0] + "/reset")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	rootCmd.AddCommand(resetCmd)
	verifyCmd := &cobra.Command{
		Use:                   "verify NID",
		Short:                 "Chain data verify",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			resp, err := hc.Post(node.UrlChain + "/" + args[0] + "/verify")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
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
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			format := cmd.Flag("format").Value.String()
			var v interface{}
			params := &url.Values{}
			if format == "" {
				v = new(node.SystemView)
			}else{
				v = new(string)
				params.Add("format", format)
			}
			resp, err := hc.Get(node.UrlSystem, &v, params)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			if format == "" {
				s, err := JsonIntend(v)
				if err != nil {
					fmt.Println(err, resp)
					return
				}
				fmt.Println(s)
			} else {
				s := v.(*string)
				fmt.Println(*s)
			}
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
		Run: func(cmd *cobra.Command, args []string) {
			hc := GetUnixDomainSockHttpClient(cfg)
			v := node.StatsView{}
			params := &url.Values{}
			params.Add("interval", fmt.Sprint(intervalSec))

			var resp *http.Response
			var err error
			if noStream {
				params.Add("stream", "false")
				resp, err = hc.Get(node.UrlStats, &v, params)
				cmd.Println(v.Timestamp)
				table := StatsViewToTable(&v, 50)
				cmd.Println(table)
			} else {
				g, guiTermCh := NewCui()
				resp, err = hc.Stream(node.UrlStats, nil, &v, UpdateCuiByStatsViewStream(g), guiTermCh, params)
				TermGui(g, guiTermCh)
			}
			if err != nil {
				if noStream && err == io.EOF {
					//ignore EOF error
					err = nil
				} else {
					fmt.Println(err, resp)
				}
			}
		},
	}
	rootCmd.Flags().BoolVar(&noStream, "no-stream", false, "Only pull the first metric-statistics")
	rootCmd.Flags().IntVar(&intervalSec, "interval", 1, "Pull interval")
	return rootCmd
}
