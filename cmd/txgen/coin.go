package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

const (
	stepLimitForCoinTransfer = 1000
	initialCoinBalance       = 10 * 1000 * 1000

	timeoutForCoinTransfer = 5 * time.Second
)

type CoinTransferMaker struct {
	NID         int64
	WalletCount int
	GodWallet   module.Wallet
	wallets     []module.Wallet
}

func (m *CoinTransferMaker) Prepare(client *Client) error {
	wallets := make([]module.Wallet, m.WalletCount)
	tids := make([]string, m.WalletCount)
	for i := 0; i < m.WalletCount; i++ {
		ac := wallet.New()
		wallets[i] = ac

		tx, err := makeCoinTransfer(m.NID, m.GodWallet, ac.Address(), big.NewInt(initialCoinBalance))
		if err != nil {
			return err
		}

		if tid, err := client.SendTx(tx); err != nil {
			return err
		} else {
			tids[i] = tid
		}
	}

	for i := 0; i < m.WalletCount; i++ {
		if txr, err := client.GetTxResult(tids[i], time.Second*3); err != nil {
			return err
		} else {
			if txr.Status.Value != 1 {
				return errors.Errorf("TransactionFails:failure=%+v", txr.Failure)
			}
		}
	}
	m.wallets = wallets
	return nil
}

func (m *CoinTransferMaker) MakeOne() (interface{}, error) {
	walletNumber := len(m.wallets)
	fromIdx := rand.Intn(walletNumber)
	toIdx := (fromIdx + rand.Intn(walletNumber-1)) % walletNumber
	fromWallet := m.wallets[fromIdx]
	toWallet := m.wallets[toIdx]

	return makeCoinTransfer(m.NID, fromWallet, toWallet.Address(), big.NewInt(10))
}

func makeCoinTransfer(nid int64, from module.Wallet, to module.Address, value *big.Int) (interface{}, error) {
	tx := map[string]interface{}{
		"version":   "0x3",
		"from":      from.Address(),
		"to":        to,
		"value":     fmt.Sprintf("0x%x", value),
		"nid":       fmt.Sprintf("0x%x", nid),
		"stepLimit": fmt.Sprintf("0x%x", stepLimitForCoinTransfer),
		"timestamp": TimeStampNow(),
	}
	if err := SignTx(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
