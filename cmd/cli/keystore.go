package cli

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"syscall"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/common/wallet"
)

func readPassword(prompt string) ([]byte, error) {
	fmt.Printf(prompt)
	pb, err := term.ReadPassword(syscall.Stdin)
	fmt.Printf("\n")
	return pb, err
}

func getPasswordFromFlags(prompt string, interactive *bool, secret, pass *string) []byte {
	var pb []byte
	var err error

	if *interactive {
		if pb, err = readPassword(prompt); err != nil {
			log.Panicf("Fail to read password err=%+v", err)
		}
	} else if *secret != "" {
		if pb, err = os.ReadFile(*secret); err != nil {
			log.Panicf("fail to open KeySecret err=%+v", err)
		}
	} else {
		pb = []byte(*pass)
	}

	return pb
}

func newKeystoreGenCmd(c string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   c,
		Short: "Generate keystore",
	}
	flags := cmd.PersistentFlags()
	out := flags.StringP("out", "o", "keystore.json", "Output file path")
	interactive := flags.BoolP("interactive", "i", false, "Interactive mode for password input")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		pb := getPasswordFromFlags("Password: ", interactive, secret, pass)
		w := wallet.New()
		ks, err := wallet.KeyStoreFromWallet(w, pb)
		if err != nil {
			log.Panicf("Fail to generate keystore err=%+v", err)
		}
		if err := os.WriteFile(*out, ks, 0600); err != nil {
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
	interactive := flags.BoolP("interactive", "i", false, "Interactive mode for password input")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			if kb, err := os.ReadFile(arg); err != nil {
				log.Panicf("fail to open keystore file err=%+v", err)
			} else {
				pb := getPasswordFromFlags("Password: ", interactive, secret, pass)
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
	interactive := flags.BoolP("interactive", "i", false, "Interactive mode for password input")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the old keystore")
	out := flags.StringP("out", "o", "keystore_new.json", "Output file path")
	npass := flags.StringP("newpassword", "n", "gochain", "Password for the new keystore")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if kb, err := os.ReadFile(*keystorePath); err != nil {
			log.Panicf("fail to open keystore file err=%+v", err)
		} else {
			pb := getPasswordFromFlags("Old Password: ", interactive, secret, pass)
			w, err := wallet.NewFromKeyStore(kb, pb)
			if err != nil {
				log.Panicf("Fail to decrypt KeyStore err=%+v", err)
			}

			var npb []byte
			if *interactive {
				// Read a new password from standard input under interactive mode
				if npb, err = readPassword("New Password: "); err != nil {
					log.Panicf("Fail to read password err=%+v", err)
				}
			} else {
				npb = []byte(*npass)
			}

			ks, err := wallet.KeyStoreFromWallet(w, npb)
			if err != nil {
				log.Panicf("Fail to generate keystore err=%+v", err)
			}
			if err := os.WriteFile(*out, ks, 0600); err != nil {
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
	interactive := flags.BoolP("interactive", "i", false, "Interactive mode for password input")
	secret := flags.StringP("secret", "s", "", "KeySecret file path")
	pass := flags.StringP("password", "p", "gochain", "Password for the keystore")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if kb, err := os.ReadFile(*keystorePath); err != nil {
			log.Panicf("fail to open keystore file err=%+v", err)
		} else {
			pb := getPasswordFromFlags("Password: ", interactive, secret, pass)
			w, err := wallet.NewFromKeyStore(kb, pb)
			if err != nil {
				log.Panicf("Fail to decrypt KeyStore err=%+v", err)
			}
			fmt.Println("0x" + hex.EncodeToString(w.PublicKey()))
		}
	}
	return cmd
}
