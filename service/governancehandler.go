package service

import (
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type GovCallHandler struct {
	*CallHandler
}

func (h *GovCallHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.ActiveContract()
	if c == nil {
		return errors.New("No active contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)

	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}

		info := h.as.APIInfo()
		paramObj, err := info.ConvertParamsToTypedObj(h.method, h.params)
		if err != nil {
			return err
		}
		err = h.conn.Invoke(h, r.path, false, h.from, h.to,
			h.value, h.stepLimit, h.method, paramObj)
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				if r.err == nil {
					info := h.as.APIInfo()
					if paramObj, err := info.ConvertParamsToTypedObj(h.method, h.params); err == nil {
						if err = h.conn.Invoke(h, r.path, false, h.from, h.to, h.value, h.stepLimit, h.method, paramObj); err == nil {
							return
						}
					}
				}
				h.OnResult(module.StatusSystemError, h.stepLimit, nil)
			}
		}()
	}
	return nil
}
