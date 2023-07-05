package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/cmd/cli"
)

var (
	version = "unknown"
	build   = "unknown"
)

type ErrorWithResponse interface {
	Error() string
	Response() string
}

func main() {
	rootCmd, rootVc := cli.NewCommand(nil, nil, "goloop", "Goloop CLI")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print goloop version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("goloop version", version, build)
		},
	})

	logoLines := []string{
		"  ____  ___  _     ___   ___  ____",
		" / ___|/ _ \\| |   / _ \\ / _ \\|  _ \\",
		"| |  _| | | | |  | | | | | | | |_) |",
		"| |_| | |_| | |__| |_| | |_| |  __/",
		" \\____|\\___/|_____\\___/ \\___/|_|",
	}

	cli.NewServerCmd(rootCmd, rootVc, version, build, logoLines)
	cli.NewChainCmd(rootCmd, rootVc)
	cli.NewSystemCmd(rootCmd, rootVc)
	cli.NewUserCmd(rootCmd, rootVc)
	cli.NewStatsCmd(rootCmd, rootVc)
	cli.NewRpcCmd(rootCmd, nil)
	cli.NewDebugCmd(rootCmd, nil)
	rootCmd.AddCommand(
		cli.NewGStorageCmd("gs"),
		cli.NewGenesisCmd("gn"),
		cli.NewKeystoreCmd("ks"))

	genMdCmd := cli.NewGenerateMarkdownCommand(rootCmd, nil)
	genMdCmd.Hidden = true

	rootCmd.SilenceUsage = true
	err := rootCmd.Execute()
	if err != nil {
		if responseError, ok := err.(ErrorWithResponse); ok {
			response := responseError.Response()
			if len(response) > 0 {
				rootCmd.Println(response)
			}
		}
		os.Exit(1)
	}
}
