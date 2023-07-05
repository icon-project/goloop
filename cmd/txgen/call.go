package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

const (
	stepsForCall       = 1000
	initialUserBalance = 10 * 1000 * 1000
)

type CallMaker struct {
	NID int64

	SourcePath    string
	InstallParams map[string]string

	Method        string
	CallParams    map[string]string
	CallTemplates map[string]*template.Template
	Index, Last   int64
	GOD           module.Wallet

	owner    module.Wallet
	contract module.Address
}

var (
	callInitialBalance = big.NewInt(initialUserBalance)
)

func (m *CallMaker) Prepare(client *Client) error {
	m.owner = wallet.New()

	ts := make(map[string]*template.Template)
	for n, p := range m.CallParams {
		if tmpl, err := template.New(n).Parse(p); err != nil {
			return err
		} else {
			ts[n] = tmpl
		}
	}
	m.CallTemplates = ts

	tx1, err := makeCoinTransfer(m.NID, m.GOD, m.owner.Address(), callInitialBalance)
	if err != nil {
		return err
	}
	r, err := client.SendTxAndGetResult(tx1, time.Second*5)
	if err != nil {
		js, _ := json.MarshalIndent(tx1, "", "  ")
		log.Printf("Transaction FAIL : tx=%s", js)
		return err
	}
	if r.Status.Value != 1 {
		return errors.New("Coin Transfer FAIL")
	}

	if _, err := os.Stat(m.SourcePath); err == nil {
		deploy, err := makeDeploy(m.NID, m.owner, m.SourcePath,
			m.InstallParams)
		if err != nil {
			return err
		}

		r, err = client.SendTxAndGetResult(deploy, time.Second*3)
		if err != nil {
			js, _ := json.MarshalIndent(deploy, "", "  ")
			log.Printf("Transaction FAIL : tx=%s", js)
			return err
		}
		if r.Status.Value != 1 {
			return errors.New("DeployFailed")
		}
		m.contract = r.SCOREAddress
	} else {
		addr := new(common.Address)
		if err := addr.SetString(m.SourcePath); err != nil {
			return err
		}
		m.contract = addr
	}

	return nil
}

func (m *CallMaker) MakeOne() (interface{}, error) {
	index := atomic.AddInt64(&m.Index, 1) - 1
	if m.Last != 0 && index >= m.Last {
		return nil, ErrEndOfTransaction
	}
	context := map[string]interface{}{
		"owner": m.owner.Address(),
		"index": index,
	}
	params := make(map[string]string)
	buffer := bytes.NewBuffer(nil)
	for n, t := range m.CallTemplates {
		buffer.Reset()
		if err := t.Execute(buffer, context); err != nil {
			return nil, err
		}
		params[n] = buffer.String()
	}
	return makeCallTx(m.NID, m.owner, m.contract, m.Method, params)
}

func (m *CallMaker) Dispose(tx interface{}) {
	// do nothing
}

func makeCallTx(nid int64, from module.Wallet,
	contract module.Address, method string, params map[string]string,
) (interface{}, error) {
	tx := map[string]interface{}{
		"version":   "0x3",
		"from":      from.Address(),
		"to":        contract,
		"nid":       fmt.Sprintf("0x%x", nid),
		"stepLimit": fmt.Sprintf("0x%x", stepLimitForCoinTransfer),
		"timestamp": TimeStampNow(),
		"dataType":  "call",
		"data": map[string]interface{}{
			"method": method,
			"params": params,
		},
	}
	if err := SignTx(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
