package chain

import (
	"encoding/json"
	"path"
	"path/filepath"
	"strconv"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	ConfigDefaultNormalTxPoolSize = 5000
	ConfigDefaultPatchTxPoolSize  = 1000
	ConfigDefaultMaxBlockTxBytes  = 1024 * 1024
)

const (
	NodeCacheNone    = "none"
	NodeCacheSmall   = "small"
	NodeCacheLarge   = "large"
	NodeCacheDefault = NodeCacheNone
)

var NodeCacheOptions = [...]string{
	NodeCacheNone, NodeCacheSmall, NodeCacheLarge,
}

type Config struct {
	// fixed
	NID    int    `json:"nid"`
	DBType string `json:"db_type"`

	Platform string `json:"platform,omitempty"`

	// static
	SeedAddr         string `json:"seed_addr"`
	Role             uint   `json:"role"`
	ConcurrencyLevel int    `json:"concurrency_level,omitempty"`
	NormalTxPoolSize int    `json:"normal_tx_pool,omitempty"`
	PatchTxPoolSize  int    `json:"patch_tx_pool,omitempty"`
	MaxBlockTxBytes  int    `json:"max_block_tx_bytes,omitempty"`
	NodeCache        string `json:"node_cache,omitempty"`
	AutoStart        bool   `json:"auto_start,omitempty"`

	// runtime
	Channel        string `json:"channel"`
	SecureSuites   string `json:"secureSuites"`
	SecureAeads    string `json:"secureAeads"`
	DefWaitTimeout int64  `json:"waitTimeout"`
	MaxWaitTimeout int64  `json:"maxTimeout"`

	GenesisStorage module.GenesisStorage `json:"-"`
	Genesis        json.RawMessage       `json:"genesis"`

	BaseDir  string `json:"chain_dir"`
	FilePath string `json:"-"` // absolute path

	NIDForP2P bool `json:"-"`
}

func (c *Config) ResolveAbsolute(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if c.FilePath == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	return filepath.Clean(path.Join(filepath.Dir(c.FilePath), targetPath))
}

func (c *Config) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base := filepath.Dir(c.FilePath)
	base, _ = filepath.Abs(base)
	r, _ := filepath.Rel(base, absPath)
	return r
}

func (c *Config) CID() int {
	if cid, err := c.GenesisStorage.CID(); err == nil {
		return cid
	}
	hash := crypto.SHA3Sum256(c.GenesisStorage.Genesis())
	return int(hash[0])<<16 | int(hash[1])<<8 | int(hash[2])
}

func (c *Config) AbsBaseDir() string {
	return c.ResolveAbsolute(c.BaseDir)
}

func (c *Config) NetID() int {
	if c.NIDForP2P {
		return c.NID
	} else {
		return c.CID()
	}
}

func (c *Config) GetChannel() string {
	return GetChannel(c.Channel, c.NID)
}

func GetChannel(channel string, nid int) string {
	if channel == "" {
		return strconv.FormatInt(int64(nid), 16)
	}
	return channel
}
