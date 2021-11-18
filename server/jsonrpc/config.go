package jsonrpc

import (
	"path"
	"path/filepath"
)

type Config struct {

	// JSON RPC
	LimitOfBatch int `json:"limit_of_batch"`

	BaseDir  string `json:"chain_dir"`
	FilePath string `json:"-"` // absolute path
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

func (c *Config) AbsBaseDir() string {
	return c.ResolveAbsolute(c.BaseDir)
}
