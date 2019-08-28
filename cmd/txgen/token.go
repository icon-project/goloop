package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/pkg/errors"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
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
