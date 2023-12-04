package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/platform/basic"
)

func mustParseAddress(arg string) module.Address {
	addr := new(common.Address)
	if err := addr.SetString(arg); err == nil {
		return addr
	} else {
		data, err := os.ReadFile(arg)
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

var feeInfoByName = map[string]interface{}{
	"icon": map[string]interface{}{
		"stepPrice": "0x0",
		"stepLimit": map[string]interface{}{
			"invoke": "0x9502f900",
			"query":  "0x2faf080",
		},
		"stepCosts": map[string]interface{}{
			"schema":         "0x1",
			"default":        "0x186a0",
			"contractCall":   "0x61a8",
			"contractCreate": "0x3b9aca00",
			"contractUpdate": "0x3b9aca00",
			"contractSet":    "0x3a98",
			"delete":         "-0xf0",
			"deleteBase":     "0xc8",
			"get":            "0x19",
			"getBase":        "0xbb8",
			"input":          "0xc8",
			"log":            "0x64",
			"logBase":        "0x1388",
			"set":            "0x140",
			"setBase":        "0x2710",
			"apiCall":        "0x2710",
		},
	},
}

func getFeeInfoOf(name string) (interface{}, error) {
	switch name {
	case "none":
		return nil, nil
	default:
		if info, ok := feeInfoByName[name]; ok {
			return info, nil
		} else {
			return nil, fmt.Errorf("InvalidFeeName(name=%s)", name)
		}
	}
}

func getFeeNames() []string {
	names := make([]string, 0, len(feeInfoByName)+1)
	names = append(names, "none")
	for k, _ := range feeInfoByName {
		names = append(names, k)
	}
	return names
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
	configs := flags.StringToStringP("config", "c", nil, "Chain configuration")
	feeName := flags.String("fee", "none",
		fmt.Sprintf("Fee configuration (%s)", strings.Join(getFeeNames(), ",")))

	cmd.Run = func(cmd *cobra.Command, args []string) {
		var godAddr module.Address
		if *god != "" {
			godAddr = mustParseAddress(*god)
		}

		treasuryAddr := common.MustNewAddressFromString(*treasury)
		if treasuryAddr.IsContract() {
			log.Panicln("Treasury address shouldn't be contract")
		}

		supplyValue := new(common.HexInt)
		if _, ok := supplyValue.SetString(*supply, 0); !ok {
			log.Panicf("Total supply value=%s is invalid", *supply)
		}

		chainConfig := make(map[string]interface{})
		if basic.LatestRevision != basic.DefaultRevision {
			chainConfig["revision"] = &common.HexInt32{Value: basic.LatestRevision}
		}
		for k, v := range *configs {
			if len(v) == 0 {
				delete(chainConfig, k)
			} else {
				chainConfig[k] = v
			}
		}
		validators := make([]module.Address, len(args))
		for i, arg := range args {
			validators[i] = mustParseAddress(arg)
			if i == 0 && godAddr == nil {
				godAddr = validators[i]
			}
		}
		chainConfig["validatorList"] = validators

		if info, err := getFeeInfoOf(*feeName); err != nil {
			log.Panicf("Fail to get fee info err=%+v", err)
		} else {
			if info != nil {
				chainConfig["fee"] = info
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
			"chain":   chainConfig,
			"message": fmt.Sprintf("generated %s", time.Now()),
		}

		bs, err := json.MarshalIndent(genesis, "", "    ")
		if err != nil {
			log.Panicf("Fail to make genesis err=%+v", err)
		}
		if err := os.WriteFile(*out, bs, 0600); err != nil {
			log.Panicf("Fail to write genesis data to file %s err=%+v",
				*out, err)
		}
		fmt.Printf("Generate %s\n", *out)
	}
	return cmd
}

func newGenesisEditCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [genesis file]", c),
		Short: "Edit genesis transaction",
		Args:  ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
	}
	flags := cmd.PersistentFlags()
	god := flags.StringP("god", "g", "", "Address or keystore of GOD")
	validators := flags.StringSliceP("validator", "v", nil, "Address or keystore of Validator, [Validator...]")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		raw, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Fail to open file=%s err=%+v", filePath, err)
		}
		genesis := make(map[string]interface{})
		if err := json.Unmarshal(raw, &genesis); err != nil {
			log.Fatalf("Fail to unmarshall file=%s err=%+v", raw, err)
		}

		updated := false

		as, ok := genesis["accounts"].([]interface{})
		if !ok {
			log.Fatalf("Invalid genesis, must have 'accounts' array-node")
		}
		if *god != "" {
			godAddr := mustParseAddress(*god)
			found := false
			for i, ta := range as {
				a, ok := ta.(map[string]interface{})
				if !ok {
					log.Fatalf("Invalid genesis, parse fail %#v child[%d] of 'accounts' array-node", i, ta)
				}
				if a["name"] == "god" {
					a["address"] = godAddr
					found = true
					break
				}
			}
			if !found {
				log.Fatalf("Invalid genesis, must have 'god' node of 'accounts' array-node")
			}
			updated = true
		}

		c, ok := genesis["chain"].(map[string]interface{})
		if !ok {
			log.Fatalf("Invalid genesis, must have 'chain' node")
		}
		if len(*validators) > 0 {
			validatorAddrs := make([]module.Address, len(*validators))
			for i, validator := range *validators {
				validatorAddrs[i] = mustParseAddress(validator)
			}
			c["validatorList"] = validatorAddrs
			updated = true
		}

		if updated {
			bs, err := json.MarshalIndent(genesis, "", "    ")
			if err != nil {
				log.Panicf("Fail to make genesis err=%+v", err)
			}

			fi, _ := os.Stat(filePath)
			if err := os.WriteFile(filePath, bs, fi.Mode().Perm()); err != nil {
				log.Panicf("Fail to write genesis data to file %s err=%+v",
					filePath, err)
			}
			fmt.Printf("Updated %s\n", filePath)
		} else {
			fmt.Printf("Nothing to update %s\n", filePath)
		}

	}
	return cmd
}

func NewGenesisCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Genesis transaction manipulation"}
	cmd.AddCommand(newGenesisGenCmd("gen"))
	cmd.AddCommand(newGenesisEditCmd("edit"))
	return cmd
}
