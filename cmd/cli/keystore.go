package cli

import (
	"encoding/hex"
	"fmt"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
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
	cmd.AddCommand(publickeyFromKeyStore("pubkey"))
	return cmd
}

func publickeyFromKeyStore(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Generate publickey from keystore",
	}
	flags := cmd.PersistentFlags()
	keystorePath := flags.StringP("keystore", "k", "keystore.json", "Keystore file path")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		var pb []byte
		if kb, err := ioutil.ReadFile(*keystorePath); err != nil {
			log.Panicf("fail to open keystore file err=%+v", err)
		} else {
			if *secret != "" {
				if pb, err = ioutil.ReadFile(*secret); err != nil {
					log.Panicf("fail to open KeySecret err=%+v", err)
				}
			} else {
				pb = []byte(*pass)
			}

			w, err := wallet.NewFromKeyStore(kb, pb)
			if err != nil {
				log.Panicf("Fail to decrypt KeyStore err=%+v", err)
			}
			fmt.Println("0x" + hex.EncodeToString(w.PublicKey()))
		}
	}
	return cmd
}
