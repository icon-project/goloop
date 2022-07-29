package imports

import (
	"encoding/json"
	"reflect"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/legacy"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
)

type ImportCallback interface {
	OnError(err error)
	OnEnd(errCh <-chan error)
}

type managerForImport struct {
	module.TransitionManager // to force overriding of TransitionManager methods
	module.ServiceManager
	bdb        *legacy.LoopChainDB
	lastHeight int64
	cb         ImportCallback
}

func NewServiceManagerForImport(chain module.Chain, nm module.NetworkManager,
	eem eeproxy.Manager, plt base.Platform, contractDir string, lcDBDir string,
	height int64, cb ImportCallback,
) (module.ServiceManager, module.Timestamper, error) {
	manager, err := service.NewManager(chain, nm, eem, plt, contractDir)
	if err != nil {
		return nil, nil, err
	}
	bdb, err := legacy.OpenDatabase(lcDBDir, lcDBDir)
	if err != nil {
		return nil, nil, err
	}
	blk, err := bdb.GetLastBlock()
	if err != nil {
		return nil, nil, err
	}
	if blk.Height() < height {
		return nil, nil, errors.Errorf("last height in data source : %d",
			blk.Height())
	}
	m := &managerForImport{
		ServiceManager: manager,
		bdb:            bdb,
		lastHeight:     height,
		cb:             cb,
	}
	return m, m, nil
}

func (m *managerForImport) GetVoteTimestamp(h, ts int64) int64 {
	if h >= m.lastHeight {
		return ts
	}
	blk, err := m.bdb.GetBlockByHeight(int(h + 1))
	if err != nil {
		m.cb.OnError(err)
		return ts
	}
	return blk.Timestamp()
}

func (m *managerForImport) GetBlockTimestamp(h, ts int64) int64 {
	if h == 1 {
		blk, err := m.bdb.GetBlockByHeight(int(h))
		if err != nil {
			m.cb.OnError(err)
			return ts
		}
		ts = blk.Timestamp()
	}
	return ts
}

func unwrap(tr module.Transition) module.Transition {
	return tr.(*transitionForImport).Transition
}

func (m *managerForImport) ProposeTransition(
	parent module.Transition,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
) (module.Transition, error) {
	if bi.Height() > m.lastHeight {
		err := errors.Errorf("height:%d > lastHeight:%d\n", bi.Height(), m.lastHeight)
		return nil, err
	}
	blk, err := m.bdb.GetBlockByHeight(int(bi.Height()))
	if err != nil {
		m.cb.OnError(err)
		return nil, err
	}
	if bi.Height() == 1 {
		bi = common.NewBlockInfo(1, blk.Timestamp())
	}
	txl := blk.NormalTransactions()
	var txs []module.Transaction
	for it := txl.Iterator(); it.Has(); it.Next() {
		tx, _, _ := it.Get()
		txs = append(txs, tx)
	}
	txl2 := m.ServiceManager.TransactionListFromSlice(txs, module.BlockVersion2)
	otr, err := m.ServiceManager.CreateTransition(unwrap(parent), txl2, bi, csi, true)
	if err != nil {
		return nil, err
	}
	return &transitionForImport{
		Transition: otr,
		m:          m,
		bi:         bi,
		errCh:      make(chan error, 1),
	}, nil
}

func (m *managerForImport) CreateInitialTransition(result []byte, nextValidators module.ValidatorList) (module.Transition, error) {
	otr, err := m.ServiceManager.CreateInitialTransition(result, nextValidators)
	if err != nil {
		return nil, err
	}
	return &transitionForImport{
		Transition: otr,
		m:          m,
		errCh:      make(chan error, 1),
	}, nil
}

func (m *managerForImport) CreateTransition(
	parent module.Transition,
	txs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	validated bool,
) (module.Transition, error) {
	otr, err := m.ServiceManager.CreateTransition(unwrap(parent), txs, bi, csi, true)
	if err != nil {
		return nil, err
	}
	return &transitionForImport{
		Transition: otr,
		m:          m,
		bi:         bi,
		errCh:      make(chan error, 1),
	}, nil
}

func (m *managerForImport) GetPatches(parent module.Transition, bi module.BlockInfo) module.TransactionList {
	return m.ServiceManager.GetPatches(unwrap(parent), bi)
}

func (m *managerForImport) WaitForTransaction(parent module.Transition, bi module.BlockInfo, cb func()) bool {
	return m.ServiceManager.WaitForTransaction(unwrap(parent), bi, cb)
}

func (m *managerForImport) PatchTransition(
	transition module.Transition,
	patches module.TransactionList,
	bi module.BlockInfo,
) module.Transition {
	otr := m.ServiceManager.PatchTransition(unwrap(transition), patches, bi)
	if otr == nil {
		return nil
	}
	return &transitionForImport{
		Transition: otr,
		m:          m,
		bi:         transition.(*transitionForImport).bi,
		errCh:      make(chan error),
	}
}

func (m *managerForImport) CreateSyncTransition(transition module.Transition, result []byte, vlHash []byte, noBuffer bool) module.Transition {
	otr := m.ServiceManager.CreateSyncTransition(unwrap(transition), result, vlHash, noBuffer)
	if otr == nil {
		return nil
	}
	return &transitionForImport{
		Transition: otr,
		m:          m,
		bi:         transition.(*transitionForImport).bi,
		errCh:      make(chan error),
	}
}

func (m *managerForImport) Finalize(transition module.Transition, opt int) error {
	if opt&module.FinalizeNormalTransaction != 0 {
		tr := transition.(*transitionForImport)
		h := tr.bi.Height()
		if h >= m.lastHeight {
			cb := m.cb
			errCh := tr.errCh
			go func() {
				cb.OnEnd(errCh)
			}()
		}
	}
	return m.finalize(transition, opt)
}

func (m *managerForImport) finalize(transition module.Transition, opt int) error {
	return m.ServiceManager.Finalize(unwrap(transition), opt)
}

func (m *managerForImport) Term() {
	log.Infof("Term ServiceManager for Import\n")
	log.Must(m.bdb.Close())
	m.ServiceManager.Term()
}

type transitionForImport struct {
	module.Transition
	m        *managerForImport
	bi       module.BlockInfo
	errCh    chan error
	cb       module.TransitionCallback
	canceler func() bool
}

func (t *transitionForImport) OnValidate(tr module.Transition, e error) {
	if t.bi.Height() == 0 {
		t.cb.OnValidate(t, e)
		return
	}
	if e != nil {
		t.cb.OnValidate(t, e)
		return
	}
	blk, err := t.m.bdb.GetBlockByHeight(int(t.bi.Height()))
	if err != nil {
		t.m.cb.OnError(err)
		t.cb.OnValidate(t, err)
		t.canceler()
		return
	}
	txl := blk.NormalTransactions()
	var txs []module.Transaction
	for it := txl.Iterator(); it.Has(); it.Next() {
		tx, _, _ := it.Get()
		txs = append(txs, tx)
	}
	txl2 := t.m.ServiceManager.TransactionListFromSlice(txs, module.BlockVersion2)
	if txl2.Equal(t.NormalTransactions()) {
		t.cb.OnValidate(t, nil)
	} else {
		t.cb.OnValidate(t, errors.New("transaction list is different"))
		t.canceler()
	}
}

func preprocess(r module.Receipt) (map[string]interface{}, error) {
	j, err := r.ToJSON(3)
	if err != nil {
		return nil, err
	}
	jm := j.(map[string]interface{})
	delete(jm, "failure")
	delete(jm, "logsBloom")
	jbs, err := json.Marshal(jm)
	if err != nil {
		return nil, err
	}
	var jm2 map[string]interface{}
	err = json.Unmarshal(jbs, &jm2)
	if err != nil {
		return nil, err
	}
	return jm2, nil
}

func isAcceptableDiff(gc module.Receipt, lc module.Receipt) (bool, error) {
	gm, err := preprocess(gc)
	if err != nil {
		return false, err
	}
	lm, err := preprocess(lc)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(gm, lm), nil
}

func (t *transitionForImport) OnExecute(tr module.Transition, e error) {
	if t.bi.Height() == 0 {
		t.cb.OnExecute(t, e)
		t.errCh <- e
		return
	}
	if e != nil {
		t.cb.OnExecute(t, e)
		t.errCh <- e
		return
	}
	blk, err := t.m.bdb.GetBlockByHeight(int(t.bi.Height()))
	if err != nil {
		t.m.cb.OnError(err)
		t.cb.OnExecute(t, err)
		t.errCh <- err
		t.canceler()
		return
	}
	txl := blk.NormalTransactions()
	rl := tr.NormalReceipts()
	rit := rl.Iterator()
	for i := txl.Iterator(); i.Has(); i.Next() {
		tx, _, err := i.Get()
		if err != nil {
			t.m.cb.OnError(err)
			t.cb.OnExecute(t, err)
			t.errCh <- err
			t.canceler()
			return
		}
		rct, err := t.m.bdb.GetReceiptByTransaction(tx.ID())
		if err != nil {
			t.m.cb.OnError(err)
			t.cb.OnExecute(t, err)
			t.errCh <- err
			t.canceler()
			return
		}
		nrct, err := rit.Get()
		if err != nil {
			t.m.cb.OnError(err)
			t.cb.OnExecute(t, err)
			t.errCh <- err
			t.canceler()
			return
		}
		res, err := isAcceptableDiff(rct, nrct)
		if err != nil {
			t.m.cb.OnError(err)
			t.cb.OnExecute(t, err)
			t.errCh <- err
			t.canceler()
			return
		}
		if !res {
			rjsn, _ := rct.ToJSON(3)
			mrjsn := rjsn.(map[string]interface{})
			rjbs, _ := json.Marshal(mrjsn)
			nrjsn, _ := nrct.ToJSON(3)
			mnrjsn := nrjsn.(map[string]interface{})
			nrjbs, _ := json.Marshal(mnrjsn)
			err = errors.Errorf("cannot agree with receipt lc:%s gc:%s tx:%x", rjbs, nrjbs, tx.ID())
			t.m.cb.OnError(err)
			t.cb.OnExecute(t, err)
			t.errCh <- err
			t.canceler()
			return
		}
		rit.Next()
	}
	t.cb.OnExecute(t, nil)
	t.errCh <- nil
}

func (t *transitionForImport) Equal(transition module.Transition) bool {
	return t.Transition.Equal(unwrap(transition))
}

func (t *transitionForImport) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	t.cb = cb
	c, e := t.Transition.Execute(t)
	t.canceler = c
	return c, e
}

func (t *transitionForImport) ExecuteForTrace(ti module.TraceInfo) (canceler func() bool, err error) {
	return nil, errors.UnsupportedError.New("UnsupportedTrace(Importing)")
}
