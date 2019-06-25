package service

import (
	"github.com/icon-project/goloop/module"
)

func Inspect(c module.Chain) map[string]interface{} {
	mgr := c.ServiceManager().(*manager)
	m := make(map[string]interface{})
	m["normalTxPool"] = inspectTxPool(mgr.normalTxPool)
	m["patchTxPool"] = inspectTxPool(mgr.patchTxPool)
	return m
}

func inspectTxPool(p *TransactionPool) map[string]interface{} {
	m := make(map[string]interface{})
	m["size"] = p.Size()
	m["used"] = p.Used()
	return m
}
