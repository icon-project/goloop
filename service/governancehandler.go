package service

import "github.com/pkg/errors"

type GovCallHandler struct {
	*CallHandler
}

func (h *GovCallHandler) ExecuteAsync(wc WorldContext) error {
	// skip to check if governance is active
	h.as = wc.GetAccountState(h.th.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.as.NextContract())
	if err != nil {
		return err
	}

	info := h.as.APIInfo()
	paramObj, err := info.ConvertParamsToTypedObj(h.method, h.params)
	if err != nil {
		return err
	}

	err = h.conn.Invoke(h, path, false, h.th.from, h.th.to, h.th.value,
		h.th.stepLimit, h.method, paramObj)
	if err != nil {
		return err
	}

	return nil
}
