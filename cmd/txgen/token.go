package main

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

const (
	stepsForTokenTransfer = 1000
	initialTokenBalance   = 10 * 1000 * 1000
)

type TokenTransferMaker struct {
	NID         int64
	WalletCount int
	SourcePath  string
	Method      string

	owner    module.Wallet
	wallets  []module.Wallet
	contract module.Address
}

var (
	tokenInitialBalance = big.NewInt(initialTokenBalance)
	tokenTransferAmount = big.NewInt(10)
)

func (m *TokenTransferMaker) Prepare(client *Client) error {
	m.owner = wallet.New()

	deploy, err := makeDeploy(m.NID, m.owner, m.SourcePath,
		map[string]interface{}{
			"_initialSupply": fmt.Sprintf("0x%x", 1000),
			"_decimals":      fmt.Sprintf("0x%x", 18),
		})
	if err != nil {
		return err
	}
	r, err := client.SendTxAndGetResult(deploy, time.Second*3)
	if err != nil {
		js, _ := json.MarshalIndent(deploy, "", "  ")
		log.Printf("Transaction FAIL : tx=%s", js)
		return err
	}
	if r.Status.Value != 1 {
		return errors.New("DeployFailed")
	}
	m.contract = r.SCOREAddress

	m.wallets = make([]module.Wallet, m.WalletCount)
	tids := make([]string, m.WalletCount)
	for i := 0; i < m.WalletCount; i++ {
		m.wallets[i] = wallet.New()
		tx, err := makeTokenTransfer(m.NID, m.contract,
			m.Method, m.owner, m.wallets[i].Address(), tokenInitialBalance)
		if err != nil {
			return err
		}
		if tid, err := client.SendTx(tx); err != nil {
			return err
		} else {
			tids[i] = tid
		}
	}

	for _, tid := range tids {
		r, err := client.GetTxResult(tid, time.Second*5)
		if err != nil {
			return err
		}
		if r.Status.Value != 1 {
			return errors.Errorf("Fail to transfer initial balance")
		}
	}
	return nil
}

func (m *TokenTransferMaker) MakeOne() (interface{}, error) {
	fromIndex := rand.Intn(m.WalletCount)
	toIndex := (fromIndex + rand.Intn(m.WalletCount-1)) % m.WalletCount
	from := m.wallets[fromIndex]
	to := m.wallets[toIndex]

	return makeTokenTransfer(m.NID, m.contract, m.Method, from, to.Address(), tokenTransferAmount)
}

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
		"stepLimit": fmt.Sprintf("0x%x", stepsForTokenTransfer),
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

func makeTokenTransfer(nid int64, contract module.Address, method string, from module.Wallet, to module.Address, value *big.Int) (interface{}, error) {
	tx := map[string]interface{}{
		"version":   "0x3",
		"from":      from.Address(),
		"to":        contract,
		"nid":       fmt.Sprintf("0x%x", nid),
		"stepLimit": fmt.Sprintf("0x%x", stepsForCoinTransfer),
		"timestamp": TimeStampNow(),
		"dataType":  "call",
		"data": map[string]interface{}{
			"method": method,
			"params": map[string]interface{}{
				"_to":    to,
				"_value": fmt.Sprintf("0x%x", value),
			},
		},
	}
	if err := SignTx(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
