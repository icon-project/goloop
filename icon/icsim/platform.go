package icsim

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type platform struct {
	calculator iiss.CalculatorHolder
}

func (p *platform) NewBaseTransaction(wc WorldContext) (module.Transaction, error) {
	// calculate issued i-score and amount balance. No changes on world context.
	es := p.getExtensionState(wc, nil)
	if es == nil || !es.IsDecentralized() {
		return nil, nil
	}

	t := common.HexInt64{Value: wc.BlockTimeStamp()}
	v := common.HexUint16{Value: module.TransactionVersion3}
	prep, issue := iiss.GetIssueData(es)
	data := make(map[string]interface{})
	if prep != nil {
		data["prep"] = prep
	}
	if issue != nil {
		data["result"] = issue
	}
	mtx := map[string]interface{}{
		"timestamp": t,
		"version":   v,
		"dataType":  "base",
		"data":      data,
	}
	bs, err := json.Marshal(mtx)
	if err != nil {
		return nil, err
	}
	tx, err := transaction.NewTransactionFromJSON(bs)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (p *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot, logger log.Logger) {
	// Start background calculator if it's not started.
	p.calculator.Start(ess, logger)
}

func (p *platform) OnExecutionBegin(wc WorldContext, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	//if revision >= icmodule.Revision12 && revision < icmodule.RevisionFixRLPBug {
	//	// Set batch data root storing block batch data and tx batch data
	//	wc.(contract.Context).SetProperty(BatchKey, new(batchRoot).Init(nil))
	//}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	return es.OnExecutionBegin(wc)
}

func (p *platform) OnExecutionEnd(wc WorldContext, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	var totalFee *big.Int
	//if revision < icmodule.RevisionEnableIISS3 {
	//	// Use virtual fee instead of total fee for IISS 2.x
	//	totalFee = er.VirtualFee()
	//} else {
	//	totalFee = er.TotalFee()
	//}
	totalFee = icmodule.BigIntZero
	return es.OnExecutionEnd(wc, totalFee, p.calculator.Get())
}

func (p *platform) OnTransactionEnd(wc WorldContext, logger log.Logger, rct txresult.Receipt) error {
	success := rct.Status() == module.StatusSuccess
	// Apply stored tx batch data
	//if value := wc.(contract.Context).GetProperty(BatchKey); value != nil {
	//	root := value.(*batchRoot)
	//	root.handleTxBatch(success)
	//}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	return es.OnTransactionEnd(wc.BlockHeight(), success)
}

func (p *platform) getExtensionState(wc WorldContext, logger log.Logger) *iiss.ExtensionStateImpl {
	es := wc.GetExtensionState()
	if es == nil {
		return nil
	}
	esi, ok := es.(*iiss.ExtensionStateImpl)
	if !ok {
		return nil
	}
	if logger != nil {
		esi.SetLogger(icutils.NewIconLogger(logger))
	}
	return esi
}
