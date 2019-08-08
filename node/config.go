package node

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	ChainConfigFileName     = "config.json"
	ChainGenesisZipFileName = "genesis.zip"
)

type StaticConfig struct {
	// static
	CliSocket         string `json:"node_sock"` // relative path
	P2PAddr           string `json:"p2p"`
	P2PListenAddr     string `json:"p2p_listen"`
	RPCAddr           string `json:"rpc_addr"`
	RPCDump           bool   `json:"rpc_dump"`
	EESocket          string `json:"ee_socket"`

	BaseDir  string `json:"node_dir"`
	FilePath string `json:"-"` // absolute path

	// build info
	BuildVersion string `json:"-"`
	BuildTags    string `json:"-"`
}

func (c *StaticConfig) ResolveAbsolute(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if c.FilePath == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	return filepath.Clean(path.Join(filepath.Dir(c.FilePath), targetPath))
}

func (c *StaticConfig) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base := filepath.Dir(c.FilePath)
	base, _ = filepath.Abs(base)
	r, _ := filepath.Rel(base, absPath)
	return r
}

func (c *StaticConfig) FillEmpty(addr module.Address) {
	if c.BaseDir == "" {
		c.BaseDir = path.Join(".", ".chain", addr.String())
	}
	if c.CliSocket == "" {
		c.CliSocket = path.Join(c.BaseDir, "cli.sock")
	}
	if c.EESocket == "" {
		c.EESocket = path.Join(c.BaseDir, "ee.sock")
	}
}

const (
	DefaultEEInstances = 1
)

type RuntimeConfig struct {
	EEInstances       int    `json:"ee_instances"`
	RPCDefaultChannel string `json:"rpc_default_channel"`

	FilePath string `json:"-"` // absolute path
}

func (c *RuntimeConfig) load() error {
	log.Println("load ", c.FilePath)
	if _, err := os.Stat(c.FilePath); err != nil {
		return err
	}
	b, err := ioutil.ReadFile(c.FilePath)
	if err != nil {
		log.Printf("%T %+v", err, err)
		return err
	}
	if err = json.Unmarshal(b, c); err != nil {
		return err
	}
	return nil
}
func (c *RuntimeConfig) save() error {
	log.Println("save ", c.FilePath)
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(c.FilePath, b, 0644); err != nil {
		return err
	}
	return err
}

func loadRuntimeConfig(baseDir string) (*RuntimeConfig, error) {
	cfg := &RuntimeConfig{
		FilePath: path.Join(baseDir, "rconfig.json"),
	}
	if err := cfg.load(); err != nil {
		if os.IsNotExist(err) {
			//save default
			cfg.EEInstances = DefaultEEInstances
			if err = cfg.save(); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}
	return cfg, nil
}
