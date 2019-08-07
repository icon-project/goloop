package main

import (
	"os"

	"github.com/icon-project/goloop/cmd/cli"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{Use: os.Args[0]}
	cmd.AddCommand(cli.NewGStorageCmd("gs"))
	cmd.AddCommand(cli.NewGenesisCmd("gn"))
	cmd.AddCommand(cli.NewKeystoreCmd("ks"))
	cmd.Execute()
}
