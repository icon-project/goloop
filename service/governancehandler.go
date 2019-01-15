package service

import (
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type GovCallHandler struct {
	*CallHandler
}

func (h *GovCallHandler) ExecuteAsync(wc WorldContext) error {
	// Calculate steps
	if !h.ApplySteps(wc, StepTypeContractCall, 1) {
		h.cc.OnResult(module.StatusNotPayable, h.stepLimit, nil, nil)
		return nil
	}

	// Prepare
	h.as = wc.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return errors.New("FAIL: not a contract account")
	}

	wc.SetContractInfo(&ContractInfo{Owner: h.as.ContractOwner()})

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	// Set up contract files
	c := h.as.ActiveContract()
	if c == nil {
		return errors.New("No active contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)

	// Execute
	select {
	case r := <-ch:
		if r.Error != nil {
			return r.Error
		}

		var err error
		if err = h.ensureParamObj(); err == nil {
			err = h.conn.Invoke(h, r.Path, false, h.from, h.to,
				h.value, h.StepAvail(), h.method, h.paramObj)
		}
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				if r.Error == nil {
					var err error
					if err = h.ensureParamObj(); err == nil {
						if err = h.conn.Invoke(h, r.Path, false,
							h.from, h.to, h.value, h.StepAvail(),
							h.method, h.paramObj); err == nil {
							return
						}
					}
				}
				h.cc.OnResult(module.StatusSystemError, h.stepLimit, nil, nil)
			}
		}()
	}
	return nil
}
