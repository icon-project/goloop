package transaction

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var version2FixedFee = big.NewInt(10 * state.PETA)
var version2StepPrice = big.NewInt(10 * state.GIGA)
var version2StepUsed = big.NewInt(1000000)
var version2ZeroPrice = new(big.Int)

type transactionV2 struct {
	*transactionJSON
	hash   []byte
	txHash []byte
}

func (tx *transactionV2) updateTxHash() error {
	if tx.txHash == nil {
		h, err := tx.calcHash(Version2)
		if err != nil {
			return err
		}
		tx.txHash = h
	}
	return nil
}

func (tx *transactionV2) ID() []byte {
	if err := tx.updateTxHash(); err != nil {
		log.Debugf("Fail to calculate TxHash err=%+v", err)
		return nil
	}
	return tx.txHash
}

func (tx *transactionV2) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.txHash)
	if err != nil {
		return InvalidSignatureError.Wrap(err, "fail to recover public key")
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(&tx.transactionJSON.From) {
		return nil
	}
	return InvalidSignatureError.New("fail to verify signature")
}

func (tx *transactionJSON) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func (tx *transactionV2) From() module.Address {
	return &tx.transactionJSON.From
}

func (tx *transactionV2) Version() int {
	return module.TransactionVersion2
}

func (tx *transactionV2) Verify() error {
	// value >= 0
	if tx.Value != nil && tx.Value.Sign() < 0 {
		return InvalidTxValue.Errorf("InvalidTxValue(%s)", tx.Value.String())
	}

	// character level size of data element <= 512KB
	if n, err := countBytesOfCompactJSON(tx.Data); err != nil || n > txMaxDataSize {
		return InvalidTxValue.Errorf("InvalidTxData(%s)", tx.Value.String())
	}

	// fee == FixedFee
	if tx.Fee.Int.Cmp(version2FixedFee) != 0 {
		return InvalidTxValue.Errorf("InvalidFee(%s)", tx.Fee.String())
	}

	// check if it's EOA
	if tx.To().IsContract() {
		return InvalidTxValue.Errorf("NotEOA(%s)", tx.To().String())
	}

	// signature verification
	if err := tx.updateTxHash(); err != nil {
		return err
	}

	if !bytes.Equal(tx.txHash, tx.TxHashV2) {
		return InvalidTxValue.Errorf("InvalidHash(%x, %v)", tx.txHash, tx.TxHashV2.Bytes())
	}

	if err := tx.verifySignature(); err != nil {
		return err
	}

	return nil
}

func (tx *transactionV2) ValidateNetwork(nid int) bool {
	return true
}

func (tx *transactionV2) PreValidate(wc state.WorldContext, update bool) error {
	// balance >= (fee + value)
	trans := new(big.Int).Add(&tx.Value.Int, &tx.Fee.Int)
	as1 := wc.GetAccountState(tx.From().ID())
	balance1 := as1.GetBalance()
	if balance1.Cmp(trans) < 0 {
		return scoreresult.ErrOutOfBalance
	}

	// for cumulative balance check
	if update {
		as2 := wc.GetAccountState(tx.To().ID())
		balance2 := as2.GetBalance()
		as1.SetBalance(new(big.Int).Sub(balance1, trans))
		as2.SetBalance(new(big.Int).Add(balance2, &tx.Value.Int))
	}
	return nil
}

func (tx *transactionV2) GetHandler(cm contract.ContractManager) (Handler, error) {
	return tx, nil
}

func (tx *transactionV2) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{string(tx.From().ID()), state.AccountWriteLock},
		{string(tx.To().ID()), state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (tx *transactionV2) Execute(ctx contract.Context, wcs state.WorldSnapshot, estimate bool) (txresult.Receipt, error) {
	r := txresult.NewReceipt(ctx.Database(), ctx.Revision(), tx.To())
	amount := &tx.Value.Int
	trans := new(big.Int).Add(amount, version2FixedFee)
	as1 := ctx.GetAccountState(tx.From().ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(trans) < 0 {
		r.SetResult(module.StatusOutOfBalance, version2StepUsed, version2ZeroPrice, nil)
		return r, nil
	}

	as1.SetBalance(new(big.Int).Sub(bal1, trans))

	as2 := ctx.GetAccountState(tx.To().ID())
	bal2 := as2.GetBalance()
	as2.SetBalance(new(big.Int).Add(bal2, amount))

	r.SetResult(module.StatusSuccess, version2StepUsed, version2StepPrice, nil)
	traceLogger := ctx.GetTraceLogger(module.EPhaseTransaction)
	traceLogger.OnBalanceChange(module.Transfer, tx.From(), tx.To(), amount)
	return r, nil
}

func (tx *transactionV2) Dispose() {
}

func (tx *transactionV2) Query(wc state.WorldContext) (module.Status, interface{}) {
	log.Panicln("V2 transaction shouldn't be called by icx_call()")
	return module.StatusSuccess, nil
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
	if nonce := tx.transactionJSON.Nonce; nonce != nil {
		return &nonce.Int
	}
	return nil
}

func (tx *transactionV2) To() module.Address {
	return &tx.transactionV3Data.To
}

func (tx *transactionV2) ToJSON(version module.JSONVersion) (interface{}, error) {
	var jso map[string]interface{}
	if err := json.Unmarshal(tx.raw, &jso); err != nil {
		return nil, InvalidFormat.Errorf("Unmarshal FAILs(%s)", string(tx.raw))
	}
	return jso, nil
}

func (tx *transactionV2) MarshalJSON() ([]byte, error) {
	return tx.raw, nil
}

func (tx *transactionV2) IsSkippable() bool {
	return true
}

func checkV2(jso map[string]interface{}) bool {
	if _, ok := jso["version"]; ok {
		return false
	}
	return true
}

func parseV2(js []byte, raw bool) (Transaction, error) {
	jso, err := parseTransactionJSON(js)
	if err != nil {
		return nil, err
	}
	return &transactionV2{transactionJSON: jso}, nil
}

func init() {
	RegisterFactory(&Factory{
		Priority:  30,
		CheckJSON: checkV2,
		ParseJSON: parseV2,
	})
}
