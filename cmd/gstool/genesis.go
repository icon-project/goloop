package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/spf13/cobra"
)

func mustParseAddress(arg string) module.Address {
	addr := new(common.Address)
	if err := addr.SetString(arg); err == nil {
		return addr
	} else {
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Panicf("%s isn't address or keystore file", arg)
		}
		addr, err := wallet.ReadAddressFromKeyStore(data)
		if err != nil {
			log.Panicf("Fail to parse %s for KeyStore err=%+v", arg, err)
		}
		return addr
	}
}

func newGenesisGenCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [address or keystore...]", c),
		Short: "Generate genesis transaction",
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmd.PersistentFlags()
	out := flags.StringP("out", "o", "genesis.json", "Output file path")
	god := flags.StringP("god", "g", "", "Address or keystore of GOD")
	supply := flags.StringP("supply", "s", "0x2961fff8ca4a62327800000", "Total supply of the chain")
	treasury := flags.StringP("treasury", "t", "hx1000000000000000000000000000000000000000", "Treasury address")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		var godAddr module.Address
		if *god != "" {
			godAddr = mustParseAddress(*god)
		}

		treasuryAddr := common.NewAddressFromString(*treasury)
		if treasuryAddr.IsContract() {
			log.Panicln("Treasury address shouldn't be contract")
		}

		supplyValue := new(common.HexInt)
		if _, ok := supplyValue.SetString(*supply, 0); !ok {
			log.Panicf("Total supply value=%s is invalid", *supply)
		}

		validators := make([]module.Address, len(args))
		for i, arg := range args {
			validators[i] = mustParseAddress(arg)
			if i == 0 && godAddr == nil {
				godAddr = validators[i]
			}
		}

		genesis := map[string]interface{}{
			"accounts": []interface{}{
				map[string]interface{}{
					"name":    "god",
					"address": godAddr,
					"balance": supplyValue,
				},
				map[string]interface{}{
					"name":    "treasury",
					"address": treasuryAddr,
					"balance": "0x0",
				},
			},
			"chain": map[string]interface{}{
				"validatorList": validators,
			},
			"message": fmt.Sprintf("generated %s", time.Now()),
		}

		bs, err := json.MarshalIndent(genesis, "", "    ")
		if err != nil {
			log.Panicf("Fail to make genesis err=%+v", err)
		}
		if err := ioutil.WriteFile(*out, bs, 0700); err != nil {
			log.Panicf("Fail to write genesis data to file %s err=%+v",
				*out, err)
		}
		fmt.Printf("Generate %s\n", *out)
	}
	return cmd
}

func NewGenesisCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Genesis transaction manipulation"}
	cmd.AddCommand(newGenesisGenCmd("gen"))
	return cmd
}
