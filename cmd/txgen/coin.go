package main

import (
	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
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
	NID          int64
	WalletCount  int
	GodWallet    module.Wallet
	wallets      []module.Wallet
	NoWaitResult bool
	TxIndex      int64
	TxCount      int64
	TxPool       TxPool
}

func (m *CoinTransferMaker) Prepare(client *Client) error {
	m.TxPool.Base = v3.TransactionParam{
		Version:   jsonrpc.HexIntFromInt64(3),
		Value:     jsonrpc.HexIntFromInt64(10),
		NetworkID: jsonrpc.HexIntFromInt64(m.NID),
		StepLimit: jsonrpc.HexIntFromInt64(stepLimitForCoinTransfer),
	}
	m.TxPool.Init()

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

	if !m.NoWaitResult {
		for i := 0; i < m.WalletCount; i++ {
			if txr, err := client.GetTxResult(tids[i], time.Second*3); err != nil {
				return err
			} else {
				if txr.Status.Value != 1 {
					return errors.Errorf("TransactionFails:failure=%+v", txr.Failure)
				}
			}
		}
	}
	m.wallets = wallets
	return nil
}

func (m *CoinTransferMaker) MakeOne() (interface{}, error) {
	if m.TxCount > 0 {
		if m.TxIndex >= m.TxCount {
			return nil, ErrEndOfTransaction
		}
		m.TxIndex += 1
	}
	walletNumber := len(m.wallets)
	fromIdx := rand.Intn(walletNumber)
	toIdx := (fromIdx + rand.Intn(walletNumber-1)) % walletNumber
	fromWallet := m.wallets[fromIdx]
	toWallet := m.wallets[toIdx]

	tx := m.TxPool.Get()
	tx.FromAddress = jsonrpc.Address(fromWallet.Address().String())
	tx.ToAddress = jsonrpc.Address(toWallet.Address().String())
	tx.Timestamp = jsonrpc.HexInt(TimeStampNow())
	if err := client.SignTransaction(fromWallet, tx); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}

func (m *CoinTransferMaker) Dispose(tx interface{}) {
	m.TxPool.Put(tx)
}

func makeCoinTransfer(nid int64, from module.Wallet, to module.Address, value *big.Int) (interface{}, error) {
	tx := &v3.TransactionParam{
		Version:     "0x3",
		FromAddress: jsonrpc.Address(from.Address().String()),
		ToAddress:   jsonrpc.Address(to.String()),
		NetworkID:   jsonrpc.HexIntFromInt64(nid),
		Value:       jsonrpc.HexIntFromBigInt(value),
		StepLimit:   jsonrpc.HexIntFromInt64(stepLimitForCoinTransfer),
		Timestamp:   jsonrpc.HexInt(TimeStampNow()),
	}
	if err := client.SignTransaction(from, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
