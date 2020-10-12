package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/icon-project/goloop/chain/gs"
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
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		if err != nil {
			log.Panicf("Fail to open %s for write err=%+v", *out, err)
		}
		defer f.Close()
		if err := gs.WriteFromPath(f, *input); err != nil {
			log.Panicf("Fail to write genesis storage err=%+v", err)
		}
	}
	return cmd
}

func newGStorageInfoCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s genesis_storage.zip", c),
		Short: "Show genesis storage information",
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	nidOnly := flags.BoolP("nid_only", "n", false, "Showing network ID only")
	cidOnly := flags.BoolP("cid_only", "c", false, "Showing chain ID only")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			f, err := os.OpenFile(arg, os.O_RDONLY, 0)
			if err != nil {
				log.Panicf("Fail to open file=%s err=%+v", arg, err)
			}
			gs, err := gs.NewFromFile(f)
			if *cidOnly {
				cid, err := gs.CID()
				if err != nil {
					log.Panic(err)
				}
				fmt.Printf("%#x\n", cid)
			} else if *nidOnly {
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
				cid, err := gs.CID()
				if err != nil {
					log.Panicf("Fail to get CID for file=%s err=%+v", arg, err)
				}
				height := gs.Height()
				fmt.Printf("File       : %s\nNetwork ID : %#x (%[2]d)\nChain   ID : %#x (%[3]d)\nHeight     : %d\nGenesis TX\n%s\n",
					arg, nid, cid, height, buf.Bytes())
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
