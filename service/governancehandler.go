package service

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/pkg/errors"
)

type GovCallHandler struct {
	*CallHandler
}

func (h *GovCallHandler) ExecuteAsync(ctx Context) error {
	// Calculate steps
	if !h.ApplySteps(ctx, state.StepTypeContractCall, 1) {
		h.cc.OnResult(module.StatusOutOfBalance, h.stepLimit, nil, nil)
		return nil
	}

	// Prepare
	h.as = ctx.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return errors.New("FAIL: not a contract account")
	}

	ctx.SetContractInfo(&state.ContractInfo{Owner: h.as.ContractOwner()})

	h.cm = ctx.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	// Set up contract files
	c := h.as.ActiveContract()
	if c == nil {
		return errors.New("No active contract")
	}

	var err error
	h.lock.Lock()
	h.cs, err = ctx.ContractManager().PrepareContractStore(ctx, c)
	h.lock.Unlock()
	if err != nil {
		return err
	}

	path, err := h.cs.WaitResult()
	if err != nil {
		return nil
	}

	// Execute
	h.lock.Lock()
	if !h.disposed {
		if err = h.ensureParamObj(); err == nil {
			err = h.conn.Invoke(h, path, false, h.from, h.to, h.value,
				h.StepAvail(), h.method, h.paramObj)
		}
	}
	h.lock.Unlock()

	return err
}
