package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{Use: os.Args[0]}
	cmd.AddCommand(NewGStorageCmd("gs"))
	cmd.AddCommand(NewGenesisCmd("gn"))
	cmd.AddCommand(NewKeystoreCmd("ks"))
	cmd.Execute()
}
