package network

import (
	"github.com/icon-project/goloop/common/crypto"
)

type Config struct {
	ListenAddress string
	PrivateKey    *crypto.PrivateKey
	PublicKey     *crypto.PublicKey
}

const (
	DEF_LISTEN_ADDRESS = "127.0.0.1:8080"
)

var (
	c *Config
)

func GetConfig() *Config {
	if c == nil {
		//TODO Read from file or DB
		priK, pubK := crypto.GenerateKeyPair()
		c = &Config{
			ListenAddress: DEF_LISTEN_ADDRESS,
			PrivateKey:    priK,
			PublicKey:     pubK,
		}

	}
	return c
}
