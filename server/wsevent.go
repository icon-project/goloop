package server

import (
	"bytes"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/txresult"
)

type EventRequest struct {
	EventFilter
	Height  common.HexInt64 `json:"height"`
}

type EventFilter struct {
	Addr       *common.Address `json:"addr,omitempty"`
	Signature  string          `json:"event"`
	Indexed    []*string       `json:"indexed,omitempty"`
	Data       []*string       `json:"data,omitempty"`
	indexedBSs [][]byte
	dataBSs    [][]byte
	numOfArgs  int
	lb         module.LogsBloom
	indexes    []int
}

type EventNotification struct {
	Hash   common.HexBytes `json:"hash"`
	Height common.HexInt64 `json:"height"`
	Index  common.HexInt32 `json:"index"`
}

func (wm *wsSessionManager) RunEventSession(ctx echo.Context) error {
	var er EventRequest
	wss, err := wm.initSession(ctx, &er)
	if err != nil {
		return err
	}
	defer wm.StopSession(wss)

	if err := er.compile(); err != nil {
		_ = wss.response(int(jsonrpc.ErrorCodeInvalidParams), "bad event request parameter")
		return nil
	}
	_ = wss.response(0, "")

	ech := make(chan error)
	go readLoop(wss.c, ech)

	h := er.Height.Value
	var bch <-chan module.Block
	bm := wss.chain.BlockManager()
	sm := wss.chain.ServiceManager()

loop:
	for {
		bch, err = bm.WaitForBlock(h)
		if err != nil {
			break loop
		}
		select {
		case err = <-ech:
			break loop
		case blk := <-bch:
			if !blk.LogsBloom().Contain(er.lb) {
				h++
				continue loop
			}
			rl, err := sm.ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
			if err != nil {
				break loop
			}
			index := int32(0)
			for rit := rl.Iterator(); rit.Has(); rit.Next() {
				r, err := rit.Get()
				if err != nil {
					break loop
				}
				if er.match(r) {
					var en EventNotification
					en.Height.Value = h
					en.Hash = blk.ID()
					en.Index.Value = index
					if err := wss.WriteJSON(&en); err != nil {
						wm.logger.Infof("fail to write json EventNotification err:%+v\n", err)
						break loop
					}
				}
				index++
			}
		}
		h++
	}
	wm.logger.Warnf("%+v\n", err)
	return nil
}

func (f *EventFilter) compile() error {
	lb := txresult.NewLogsBloom(nil)
	if f.Addr != nil {
		lb.AddAddressOfLog(f.Addr)
	}
	f.numOfArgs = len(f.Indexed)+len(f.Data)
	name, pts := txresult.DecomposeEventSignature(f.Signature)
	if len(name) == 0 || pts == nil || len(pts) < f.numOfArgs {
		return errors.NewBase(errors.IllegalArgumentError, "bad event signature")
	}
	lb.AddIndexedOfLog(0, []byte(f.Signature))
	idx := 0
	f.indexedBSs = make([][]byte, len(f.Indexed))
	for i, arg := range f.Indexed {
		if arg != nil {
			bs, err := txresult.EventDataStringToBytesByType(pts[idx], string(*arg))
			if err != nil {
				return errors.NewBase(errors.IllegalArgumentError, "bad event data")
			}
			lb.AddIndexedOfLog(i+1, bs)
			f.indexedBSs[i] = bs
		}
		idx++
	}
	f.dataBSs = make([][]byte, len(f.Data))
	for i, arg := range f.Data {
		if arg != nil {
			bs, err := txresult.EventDataStringToBytesByType(pts[idx], string(*arg))
			if err != nil {
				return errors.NewBase(errors.IllegalArgumentError, "bad event data")
			}
			f.dataBSs[i] = bs
		}
		idx++
	}
	f.lb = lb
	return nil
}

func (f *EventFilter) match(r module.Receipt) bool {
	if r.LogsBloom().Contain(f.lb) {
	loop:
		for it := r.EventLogIterator(); it.Has(); it.Next() {
			if el, err := it.Get(); err == nil {
				if bytes.Equal([]byte(f.Signature), el.Indexed()[0]) {
					if f.Addr != nil && !el.Address().Equal(f.Addr) {
						continue loop
					}
					if f.numOfArgs > 0 {
						if (len(el.Indexed()) - 1 + len(el.Data())) <= f.numOfArgs {
							continue loop
						}

						for i, arg := range f.indexedBSs {
							if len(arg) > 0 && !bytes.Equal(arg, el.Indexed()[i+1]) {
								continue loop
							}
						}
						for i, arg := range f.dataBSs {
							if len(arg) > 0 && !bytes.Equal(arg, el.Data()[i]) {
								continue loop
							}
						}
					}
					return true
				}
			}
		}
	}
	return false
}
