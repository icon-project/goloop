package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

const (
	stepLimitForTokenTransfer      = 100000
	initialTokenBalanceOfUser      = 10 * 1000 * 1000
	initialCoinBalanceOfTokenOwner = 10 * 1000 * 1000
	tokenForTransfer               = 10

	timeoutForTokenTransfer = 5 * time.Second
)

type TokenTransferMaker struct {
	NID         int64
	WalletCount int
	SourcePath  string
	Method      string
	GOD         module.Wallet
	Last        int64

	owner    module.Wallet
	wallets  []module.Wallet
	contract module.Address
	index    int64
}

var (
	tokenOwnerInitialBalance = big.NewInt(initialCoinBalanceOfTokenOwner)
	tokenInitialBalance      = big.NewInt(initialTokenBalanceOfUser)
	tokenValueForTransfer    = big.NewInt(tokenForTransfer)
)

func (m *TokenTransferMaker) Prepare(client *Client) error {
	m.owner = wallet.New()

	tr, err := makeCoinTransfer(m.NID, m.GOD, m.owner.Address(), tokenOwnerInitialBalance)
	if err != nil {
		return err
	}
	if r, err := client.SendTxAndGetResult(tr, timeoutForCoinTransfer); err != nil {
		return err
	} else {
		if r.Status.Value != 1 {
			return errors.Errorf("FailToFundingOwner(failre=%+v)", r.Failure)
		}
	}

	deploy, err := makeDeploy(m.NID, m.owner, m.SourcePath,
		map[string]interface{}{
			"_name":          "MySampleToken",
			"_symbol":        "MST",
			"_decimals":      fmt.Sprintf("0x%x", 18),
			"_initialSupply": fmt.Sprintf("0x%x", 1000),
		})
	if err != nil {
		return err
	}
	r, err := client.SendTxAndGetResult(deploy, timeoutForDeploy)
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
		r, err := client.GetTxResult(tid, timeoutForTokenTransfer)
		if err != nil {
			return err
		}
		if r.Status.Value != 1 {
			return errors.Errorf("Fail to transfer initial balance %+v", r.Failure)
		}
	}
	return nil
}

func (m *TokenTransferMaker) MakeOne() (interface{}, error) {
	index := atomic.AddInt64(&m.index, 1)
	if m.Last != 0 && index > m.Last {
		return nil, ErrEndOfTransaction
	}
	fromIndex := rand.Intn(m.WalletCount)
	toIndex := (fromIndex + rand.Intn(m.WalletCount-1)) % m.WalletCount
	from := m.wallets[fromIndex]
	to := m.wallets[toIndex]

	return makeTokenTransfer(m.NID, m.contract, m.Method, from, to.Address(), tokenValueForTransfer)
}

func makeTokenTransfer(nid int64, contract module.Address, method string, from module.Wallet, to module.Address, value *big.Int) (interface{}, error) {
	tx := map[string]interface{}{
		"version":   "0x3",
		"from":      from.Address(),
		"to":        contract,
		"nid":       fmt.Sprintf("0x%x", nid),
		"stepLimit": fmt.Sprintf("0x%x", stepLimitForTokenTransfer),
		"timestamp": TimeStampNow(),
		"dataType":  "call",
		"data": map[string]interface{}{
			"method": method,
			"params": map[string]interface{}{
				"_to":    to,
				"_value": fmt.Sprintf("0x%x", value),
				"_data":  fmt.Sprintf("0x%s", hex.EncodeToString([]byte("Hello"))),
			},
		},
	}
	if err := SignTx(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
