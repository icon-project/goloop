package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/common/wallet"
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

func newVerifyCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Verify keystore with the password",
	}
	flags := cmd.PersistentFlags()
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			var pb []byte
			if kb, err := ioutil.ReadFile(arg); err != nil {
				log.Panicf("fail to open keystore file err=%+v", err)
			} else {
				if *secret != "" {
					if pb, err = ioutil.ReadFile(*secret); err != nil {
						log.Panicf("fail to open KeySecret err=%+v", err)
					}
				} else {
					pb = []byte(*pass)
				}

				_, err := wallet.NewFromKeyStore(kb, pb)
				if err != nil {
					fmt.Printf("FAIL err=%v\n", err)
				} else {
					fmt.Println("SUCCESS")
				}
			}
		}
	}
	return cmd
}

func newReEncryptCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Re-encrypt keystore",
	}
	flags := cmd.PersistentFlags()
	keystorePath := flags.StringP("keystore", "k", "keystore.json", "Keystore file path")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the old keystore")
	out := flags.StringP("out", "o", "keystore_new.json", "Output file path")
	npass := flags.StringP("newpassword", "n", "gochain", "Password for the new keystore")

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
			ks, err := wallet.KeyStoreFromWallet(w, []byte(*npass))
			if err != nil {
				log.Panicf("Fail to generate keystore err=%+v", err)
			}
			if err := ioutil.WriteFile(*out, ks, 0600); err != nil {
				log.Panicf("Fail to write keystore err=%+v", err)
			}
			fmt.Printf("%s ==> %s\n",
				w.Address().String(), *out)
		}
	}
	return cmd
}

func NewKeystoreCmd(c string) *cobra.Command {
	cmd := &cobra.Command{Use: c, Short: "Keystore manipulation"}
	cmd.AddCommand(newKeystoreGenCmd("gen"))
	cmd.AddCommand(newVerifyCmd("verify"))
	cmd.AddCommand(publickeyFromKeyStore("pubkey"))
	cmd.AddCommand(newReEncryptCmd("encrypt"))
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

			// Check if the file starts with the UTF-8 BOM and strip it if present
			if bytes.HasPrefix(kb, []byte("\xef\xbb\xbf")) {
				kb = bytes.TrimPrefix(kb, []byte("\xef\xbb\xbf"))
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
