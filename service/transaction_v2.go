package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var version2FixedFee = big.NewInt(10 * PETA)
var version2StepPrice = big.NewInt(10 * GIGA)
var version2StepUsed = big.NewInt(1000000)

type transactionV2 struct {
	*transactionV3JSON
	hash []byte
}

func (tx *transactionV2) Version() int {
	return module.TransactionVersion2
}

func (tx *transactionV2) Verify() error {
	if tx.Fee.Int.Cmp(version2FixedFee) != 0 {
		return ErrInvalidFeeValue
	}

	if err := tx.updateTxHash(); err != nil {
		return err
	}

	if !bytes.Equal(tx.txHash, tx.Tx_Hash) {
		return ErrInvalidHashValue
	}

	if err := tx.transactionV3JSON.verifySignature(); err != nil {
		return err
	}

	return nil
}

func (tx *transactionV2) PreValidate(wc WorldContext, update bool) error {
	trans := new(big.Int)
	trans.Set(&tx.Value.Int)
	trans.Add(trans, &tx.Fee.Int)

	as1 := wc.GetAccountState(tx.From.ID())
	balance1 := as1.GetBalance()
	if balance1.Cmp(trans) < 0 {
		return ErrNotEnoughBalance
	}

	if configOnCheckingTimestamp == true {
		tsdiff := wc.BlockTimeStamp() - tx.TimeStamp.Value
		if tsdiff < int64(-5*time.Minute/time.Microsecond) ||
			tsdiff > int64(5*time.Minute/time.Microsecond) {
			return ErrTimeOut
		}
	}

	if update {
		as2 := wc.GetAccountState(tx.To.ID())
		balance2 := as2.GetBalance()
		balance2.Add(balance2, &tx.Value.Int)
		balance1.Sub(balance1, trans)
		as1.SetBalance(balance1)
		as2.SetBalance(balance2)
	}
	return nil
}

func (tx *transactionV2) GetHandler(cm ContractManager) (TransactionHandler, error) {
	return tx, nil
}

func (tx *transactionV2) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{string(tx.From.ID()), AccountWriteLock},
		{string(tx.To.ID()), AccountWriteLock},
	}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (tx *transactionV2) Execute(wc WorldContext) (Receipt, error) {
	r := NewReceipt(&tx.To)
	var trans big.Int

	trans.Add(&tx.Value.Int, version2FixedFee)

	as1 := wc.GetAccountState(tx.From.ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(&trans) < 0 {
		stepUsed := version2StepUsed
		stepPrice := new(big.Int)

		log.Printf("TX2 Fail balance=%s value=%s fee=%s",
			bal1.String(), tx.Value.Int.String(), tx.Fee.Int.String())

		if bal1.Cmp(&tx.Value.Int) < 0 {
			r.SetResult(module.StatusOutOfBalance, stepUsed, stepPrice, nil)
		} else {
			r.SetResult(module.StatusNotPayable, stepUsed, stepPrice, nil)
		}
		return r, nil
	}

	bal1.Sub(bal1, &trans)
	as1.SetBalance(bal1)

	as2 := wc.GetAccountState(tx.To.ID())
	bal2 := as2.GetBalance()
	bal2.Add(bal2, &tx.Value.Int)
	as2.SetBalance(bal2)

	r.SetResult(module.StatusSuccess, version2StepUsed, version2StepPrice, nil)
	return r, nil
}

func (tx *transactionV2) Dispose() {
}

func (tx *transactionV2) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *transactionV2) Bytes() []byte {
	return tx.raw
}

func (tx *transactionV2) Hash() []byte {
	if tx.hash == nil {
		tx.hash = crypto.SHA3Sum256(tx.Bytes())
	}
	return tx.hash
}

func (tx *transactionV2) Nonce() *big.Int {
	if nonce := tx.transactionV3JSON.Nonce; nonce != nil {
		return &nonce.Int
	}
	return nil
}

func (tx *transactionV2) ToJSON(version int) (interface{}, error) {
	if version == 2 {
		var jso map[string]interface{}
		if err := json.Unmarshal(tx.raw, &jso); err != nil {
			return nil, err
		}
		return jso, nil
	} else {
		return nil, errors.New("InvalidVersion:" + strconv.Itoa(version))
	}
}

func (tx *transactionV2) MarshalJSON() ([]byte, error) {
	return tx.raw, nil
}
