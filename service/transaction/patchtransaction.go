package transaction

import (
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

func NewPatchTransaction(p module.Patch, nid int, ts int64, w module.Wallet) (Transaction, error) {
	var v3tx transactionV3
	// fill data
	tx := &v3tx.transactionV3Data
	tx.Version.Value = module.TransactionVersion3
	tx.From.Set(w.Address())
	tx.To.SetTypeAndID(true, state.SystemID)
	tx.TimeStamp.Value = ts
	tx.NID = &common.HexInt64{Value: int64(nid)}
	dt := contract.DataTypePatch
	tx.DataType = &dt
	data := &contract.Patch{
		Type: p.Type(),
		Data: p.Data(),
	}
	js, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	tx.Data = js

	// sign
	sig, err := w.Sign(v3tx.TxHash())
	if err != nil {
		return nil, err
	}
	if err := tx.Signature.UnmarshalBinary(sig); err != nil {
		return nil, err
	}
	return &transaction{&v3tx}, nil
}
