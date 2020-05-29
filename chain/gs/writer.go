package gs

import (
	"archive/zip"
	"encoding/hex"
	"io"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type genesisStorageWriter struct {
	zw   *zip.Writer
	data map[string]bool
}

func (g *genesisStorageWriter) WriteGenesis(gtx []byte) error {
	f, err := g.zw.Create(GenesisFileName)
	if err != nil {
		return err
	}
	_, err = f.Write(gtx)
	return err
}

func (g *genesisStorageWriter) WriteData(value []byte) ([]byte, error) {
	hv := crypto.SHA3Sum256(value)
	ks := hex.EncodeToString(hv)
	if _, ok := g.data[ks]; ok {
		return hv, nil
	} else {
		g.data[ks] = true
	}
	f, err := g.zw.Create(ks)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(value); err != nil {
		return nil, err
	}
	return hv, nil
}

func (g *genesisStorageWriter) Close() error {
	if err := g.zw.Flush(); err != nil {
		return err
	}
	return g.zw.Close()
}

func NewGenesisStorageWriter(w io.Writer) module.GenesisStorageWriter {
	return &genesisStorageWriter{
		zw:   zip.NewWriter(w),
		data: make(map[string]bool),
	}
}
