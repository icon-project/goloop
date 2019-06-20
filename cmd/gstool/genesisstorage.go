package main

import (
	"bytes"
	"encoding/json"
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

func newGStorageInfoCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Show genesis storage information",
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	nidOnly := flags.BoolP("nid_only", "n", false, "Showing network ID only")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			f, err := os.OpenFile(arg, os.O_RDONLY, 0)
			if err != nil {
				log.Panicf("Fail to open file=%s err=%+v", arg, err)
			}
			gs, err := chain.NewGenesisStorageFromFile(f)
			if *nidOnly {
				nid, err := gs.NID()
				if err != nil {
					log.Panic(err)
				}
				fmt.Printf("%#x\n", nid)
			} else {
				buf := bytes.NewBuffer(nil)
				err = json.Indent(buf, gs.Genesis(), "", "    ")
				if err != nil {
					log.Panicf("Fail to indent genesis err=%+v\n%s",
						err, gs.Genesis())
				}
				nid, err := gs.NID()
				if err != nil {
					log.Panicf("Fail to get NID for file=%s err=%+v", arg, err)
				}
				fmt.Printf("File       : %s\nNetwork ID : %#x (%[2]d)\nGenesis TX\n%s\n",
					arg, nid, buf.Bytes())
			}
		}
	}
	return cmd
}

func NewGStorageCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Genesis storage manipulation"}
	cmd.AddCommand(newGStorageGenCmd("gen"))
	cmd.AddCommand(newGStorageInfoCmd("info"))
	return cmd
}
