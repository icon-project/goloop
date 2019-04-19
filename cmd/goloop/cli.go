package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/chain"
)

var (
	genesisZip, genesisPath string
	joinChainParam          JoinChainParam
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			l := make([]*ChainView, 0)
			resp, err := hc.Get(UrlChain, &l)
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			var err error
			var NID int64
			if NID, err = strconv.ParseInt(args[0],16,64); err != nil {
				fmt.Println("cannot parse NID", err)
				return
			}
			joinChainParam.NID = int(NID)
			var resp *http.Response

			if len(genesisZip) > 0 {
				resp, err = hc.PostWithFile(UrlChain, &joinChainParam, "genesisZip", genesisZip)
			} else if len(genesisPath) > 0 {
				buf := bytes.NewBuffer(nil)
				err = chain.WriteGenesisStorageFromPath(buf, genesisPath)
				if err != nil {
					fmt.Println(err)
					return
				}
				resp, err = hc.PostWithReader(UrlChain, &joinChainParam, "genesisZip", buf)
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			resp, err := hc.Delete(UrlChain + "/" + args[0])
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			v := &ChainInspectView{}
			resp, err := hc.Get(UrlChain+"/"+args[0], v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	}
	rootCmd.AddCommand(inspectCmd)
	startCmd := &cobra.Command{
		Use:                   "start NID",
		Short:                 "Chain start",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/start")
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/stop")
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/reset")
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/verify")
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
			cliSocket := cfg.ResolveAbsolute(cfg.CliSocket)
			hc := NewUnixDomainSockHttpClient(cliSocket)
			v := &SystemView{}
			resp, err := hc.Get(UrlSystem, v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	}
	return rootCmd
}
