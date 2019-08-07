package main

import (
	"fmt"
	"os"

	"github.com/icon-project/goloop/cmd/cli"
	"github.com/icon-project/goloop/node"
	"github.com/spf13/cobra"
)

var (
	version = "unknown"
	build   = "unknown"
)

func main() {
	rootCmd, rootVc := cli.NewCommand(nil, nil, "goloop", "Goloop CLI")
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print goloop version",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("goloop version", version, build)
		},
	})

	NewServerCmd(rootCmd, rootVc, version, build)
	cli.NewChainCmd(rootCmd, rootVc)
	cli.NewSystemCmd(rootCmd, rootVc)
	cli.NewStatsCmd(rootCmd, rootVc)
	cli.NewRpcCmd(rootCmd, nil)
	rootCmd.AddCommand(
		cli.NewGStorageCmd("gs"),
		cli.NewGenesisCmd("gn"),
		cli.NewKeystoreCmd("ks"))

	genMdCmd := cli.NewGenerateMarkdownCommand(rootCmd)
	genMdCmd.Hidden = true

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
