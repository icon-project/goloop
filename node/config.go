package node

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

const (
	ChainConfigFileName     = "config.json"
	ChainGenesisZipFileName = "genesis.zip"
)

type StaticConfig struct {
	// static
	CliSocket     string `json:"node_sock"` // relative path
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	RPCAddr       string `json:"rpc_addr"`
	RPCDump       bool   `json:"rpc_dump"`
	EESocket      string `json:"ee_socket"`
	Engines       string `json:"engines"`
	BackupDir     string `json:"backup_dir"`

	AuthSkipIfEmptyUsers bool `json:"auth_skip_if_empty_users,omitempty"`
	NIDForP2P            bool `json:"nid_for_p2p,omitempty"`

	BaseDir  string `json:"node_dir"`
	FilePath string `json:"-"` // absolute path

	// build info
	BuildVersion string `json:"-"`
	BuildTags    string `json:"-"`
}

func (c *StaticConfig) ResolveAbsolute(targetPath string) string {
	return ResolveAbsolute(c.FilePath, targetPath)
}

func (c *StaticConfig) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base, _ := filepath.Abs(filepath.Dir(c.FilePath))
	r, _ := filepath.Rel(base, absPath)
	return r
}

func ResolveAbsolute(baseFile, targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if baseFile == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	if !filepath.IsAbs(baseFile) {
		baseFile, _ = filepath.Abs(baseFile)
	}
	return filepath.Clean(path.Join(filepath.Dir(baseFile), targetPath))
}

func (c *StaticConfig) SetFilePath(path string) string {
	o := c.FilePath
	c.FilePath, _ = filepath.Abs(path)
	if c.BaseDir != "" {
		c.BaseDir = c.ResolveRelative(ResolveAbsolute(o, c.BaseDir))
	}
	if c.CliSocket != "" {
		c.CliSocket = c.ResolveRelative(ResolveAbsolute(o, c.CliSocket))
	}
	if c.EESocket != "" {
		c.EESocket = c.ResolveRelative(ResolveAbsolute(o, c.EESocket))
	}
	if c.BackupDir != "" {
		c.BackupDir = c.ResolveRelative(ResolveAbsolute(o, c.BackupDir))
	}
	return o
}

func (c *StaticConfig) FillEmpty(addr module.Address) {
	if c.BaseDir == "" {
		c.BaseDir = path.Join(".", ".chain", addr.String())
	}
	if c.BackupDir == "" {
		c.BackupDir = path.Join(c.BaseDir, "backup")
	}
	if c.CliSocket == "" {
		c.CliSocket = path.Join(c.BaseDir, "cli.sock")
	}
	if c.EESocket == "" {
		c.EESocket = path.Join(c.BaseDir, "ee.sock")
	}
}

func (c *StaticConfig) AbsBaseDir() string {
	return c.ResolveAbsolute(c.BaseDir)
}

const (
	DefaultEEInstances = 1
)

type RuntimeConfig struct {
	EEInstances       int    `json:"eeInstances"`
	RPCDefaultChannel string `json:"rpcDefaultChannel"`
	RPCIncludeDebug   bool   `json:"rpcIncludeDebug"`
	RPCRosetta        bool   `json:"rpcRosetta"`
	RPCBatchLimit     int    `json:"rpcBatchLimit"`

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
		EEInstances:   DefaultEEInstances,
		RPCBatchLimit: jsonrpc.DefaultBatchLimit,
		FilePath:      path.Join(baseDir, "rconfig.json"),
	}
	if err := cfg.load(); err != nil {
		if os.IsNotExist(err) {
			if err = cfg.save(); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}
	return cfg, nil
}
