package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/spf13/cobra"
)

func main() {
	var keyStoreFile string
	var keyStorePass string
	var scorePath string
	var tps int
	var concurrent int
	var walletCount int
	var nid int64
	var methodName string
	var params map[string]string
	var installParams map[string]string
	var index, last int64
	var waitTimeout int64
	var noWaitResult bool

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s [urls]", os.Args[0]),
	}
	flags := cmd.PersistentFlags()
	flags.StringVarP(&keyStoreFile, "keystore", "k", "", "File path to keystore of base account (like GOD)")
	flags.StringVarP(&keyStorePass, "password", "p", "gochain", "Password for the keystore")
	flags.IntVarP(&tps, "tps", "t", 1000, "Max transaction per a second")
	flags.IntVarP(&concurrent, "concurrent", "c", 2, "Number of subroutines (threads)")
	flags.IntVarP(&walletCount, "wallets", "w", 1000, "Number of temporal wallets")
	flags.Int64VarP(&nid, "nid", "n", 1, "Network ID of URLs")
	flags.StringVarP(&scorePath, "score", "s", "", "Path to SCORE source directory")
	flags.StringVarP(&methodName, "method", "m", "transfer", "Method name to be used for transfer")
	flags.StringToStringVar(&params, "param", nil, "Parameters for the call")
	flags.StringToStringVar(&installParams, "installParam", make(map[string]string), "Install parameters")
	flags.Int64VarP(&index, "index", "i", 0, "Initial index value to be used for generating transaction")
	flags.Int64VarP(&last, "last", "l", 0, "Last index value to be used for generating transaction")
	flags.Int64Var(&waitTimeout, "wait", 0, "Wait for specified time (in ms) for each TX (enable to use sendAndWait)")
	flags.BoolVar(&noWaitResult, "nowaitresult", false, "No wait for result for confirm in COIN transfer")

	cmd.RunE = func(cmd *cobra.Command, urls []string) error {
		if len(urls) == 0 {
			urls = []string{"http://localhost:9080/api/v3"}
		}

		if keyStoreFile == "" {
			log.Panic("KeyStore for base account isn't specified")
		}

		ks, err := ioutil.ReadFile(keyStoreFile)
		if err != nil {
			log.Panicf("Fail to read KeyStore file=%s err=%+v", keyStoreFile, err)
		}

		godWallet, err := wallet.NewFromKeyStore(ks, []byte(keyStorePass))
		if err != nil {
			log.Panicf("Fail to decrypt KeyStore err=%+v", err)
		}

		var maker TransactionMaker
		if len(scorePath) > 0 && len(methodName) > 0 {
			maker = &CallMaker{
				NID:           nid,
				SourcePath:    scorePath,
				InstallParams: installParams,
				Method:        methodName,
				CallParams:    params,
				GOD:           godWallet,
				Index:         index,
				Last:          last,
			}
		} else if len(scorePath) > 0 {
			maker = &TokenTransferMaker{
				NID:         nid,
				WalletCount: walletCount,
				SourcePath:  scorePath,
				Method:      methodName,
				GOD:         godWallet,
				Last:        last,
			}
		} else {
			maker = &CoinTransferMaker{
				NID:          nid,
				WalletCount:  walletCount,
				GodWallet:    godWallet,
				NoWaitResult: noWaitResult,
				TxCount:      last,
			}
		}

		ctx := NewContext(concurrent, int64(tps), maker, waitTimeout)
		return ctx.Run(urls)
	}

	_ = cmd.Execute()
}
