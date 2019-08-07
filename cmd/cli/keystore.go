package cli

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/spf13/cobra"
)

func newKeystoreGenCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Generate keystore",
	}
	flags := cmd.PersistentFlags()
	out := flags.StringP("out", "o", "keystore.json", "Output file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		w := wallet.New()
		ks, err := wallet.KeyStoreFromWallet(w, []byte(*pass))
		if err != nil {
			log.Panicf("Fail to generate keystore err=%+v", err)
		}
		if err := ioutil.WriteFile(*out, ks, 0600); err != nil {
			log.Panicf("Fail to write keystore err=%+v", err)
		}
		fmt.Printf("%s ==> %s\n",
			w.Address().String(), *out)
	}
	return cmd
}

func NewKeystoreCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Keystore manipulation"}
	cmd.AddCommand(newKeystoreGenCmd("gen"))
	return cmd
}
