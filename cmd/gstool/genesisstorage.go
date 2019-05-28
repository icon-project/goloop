package main

import (
	"fmt"
	"log"
	"os"

	"github.com/icon-project/goloop/chain"
	"github.com/spf13/cobra"
)

func newGStorageGenCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Create genesis storage from the template"}
	flags := cmd.PersistentFlags()
	out := flags.StringP("out", "o", "gs.zip", "Output file path")
	input := flags.StringP("input", "i", "genesis.json", "Input file or directory path")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Printf("Generating %s with %s\n", *out, *input)
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			log.Panicf("Fail to open %s for write err=%+v", *out, err)
		}
		defer f.Close()
		if err := chain.WriteGenesisStorageFromPath(f, *input); err != nil {
			log.Panicf("Fail to write genesis storage err=%+v", err)
		}
	}
	return cmd
}

func NewGStorageCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Genesis storage manipulation"}
	cmd.AddCommand(newGStorageGenCmd("gen"))
	return cmd
}
