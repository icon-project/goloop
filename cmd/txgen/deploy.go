package main

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/icon-project/goloop/module"
)

const (
	stepLimitForDeploy = 100000

	timeoutForDeploy = 5 * time.Second
)

func addDirectoryToZip(zipWriter *zip.Writer, base, uri string) error {
	p := path.Join(base, uri)
	entries, err := ioutil.ReadDir(p)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			err = addDirectoryToZip(zipWriter, base, path.Join(uri, entry.Name()))
			if err != nil {
				return err
			}
		} else {
			fd, err := os.Open(path.Join(p, entry.Name()))
			if err != nil {
				return errors.WithStack(err)
			}

			info, err := fd.Stat()
			if err != nil {
				fd.Close()
				return errors.WithStack(err)
			}

			hdr, err := zip.FileInfoHeader(info)
			if err != nil {
				fd.Close()
				return errors.WithStack(err)
			}
			hdr.Name = path.Join(uri, entry.Name())
			hdr.Method = zip.Deflate
			writer, err := zipWriter.CreateHeader(hdr)
			_, err = io.Copy(writer, fd)
			fd.Close()
		}
	}
	return nil
}

func zipDirectory(fd io.Writer, p string) error {
	zfd := zip.NewWriter(fd)
	err := addDirectoryToZip(zfd, p, "")
	if err != nil {
		return err
	}
	return zfd.Close()
}

func makeDeploy(nid int64, from module.Wallet, src string, params interface{}) (interface{}, error) {
	buf := bytes.NewBuffer(nil)
	if err := zipDirectory(buf, src); err != nil {
		return nil, err
	}
	content := "0x" + hex.EncodeToString(buf.Bytes())

	tx := map[string]interface{}{
		"version":   "0x3",
		"from":      from.Address(),
		"to":        "cx0000000000000000000000000000000000000000",
		"nid":       fmt.Sprintf("0x%x", nid),
		"stepLimit": fmt.Sprintf("0x%x", stepLimitForDeploy),
		"timestamp": TimeStampNow(),
		"dataType":  "deploy",
		"data": map[string]interface{}{
			"contentType": "application/zip",
			"content":     content,
			"params":      params,
		},
	}
	if err := SignTx(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
