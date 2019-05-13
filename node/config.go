package node

import (
	"path"
	"path/filepath"

	"github.com/icon-project/goloop/module"
)

const (
	ChainConfigFileName     = "config.json"
	ChainGenesisZipFileName = "genesis.zip"
)

type NodeConfig struct {
	// static
	CliSocket     string `json:"node_sock"` // relative path
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	RPCAddr       string `json:"rpc_addr"`
	EESocket      string `json:"ee_socket"`
	EEInstances   int    `json:"ee_instances"`

	BaseDir  string `json:"node_dir"`
	FilePath string `json:"-"` // absolute path

	// build info
	BuildVersion string `json:"-"`
	BuildTags    string `json:"-"`
}

func (c *NodeConfig) ResolveAbsolute(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if c.FilePath == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	return filepath.Clean(path.Join(filepath.Dir(c.FilePath), targetPath))
}

func (c *NodeConfig) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base := filepath.Dir(c.FilePath)
	base, _ = filepath.Abs(base)
	r, _ := filepath.Rel(base, absPath)
	return r
}

func (c *NodeConfig) FillEmpty(addr module.Address) {
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
