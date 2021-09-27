package service

import (
	"github.com/icon-project/goloop/module"
)

func Inspect(c module.Chain, informal bool) map[string]interface{} {
	var mgr *manager
	if sm := c.ServiceManager(); sm == nil {
		return nil
	} else {
		if impl, ok := sm.(*manager); ok {
			mgr = impl
		} else {
			return nil
		}
	}
	m := make(map[string]interface{})
	m["normalTxPool"] = inspectTxPool(mgr.tm.normalTxPool)
	m["patchTxPool"] = inspectTxPool(mgr.tm.patchTxPool)
	m["resultCache"] = inspectResultCache(mgr.trc)
	return m
}

func inspectResultCache(tsc *transitionResultCache) map[string]interface{} {
	m := make(map[string]interface{})
	m["used"] = tsc.Count()
	m["size"] = tsc.MaxCount()
	m["bytes"] = tsc.TotalBytes()
	return m
}

func inspectTxPool(p *TransactionPool) map[string]interface{} {
	m := make(map[string]interface{})
	m["size"] = p.Size()
	m["used"] = p.Used()
	return m
}
