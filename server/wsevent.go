package server

import (
	"bytes"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/txresult"
)

type EventRequest struct {
	EventFilter
	Height common.HexInt64 `json:"height"`
	Logs   common.HexInt32 `json:"logs,omitempty""`
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
	Hash   common.HexBytes   `json:"hash"`
	Height common.HexInt64   `json:"height"`
	Index  common.HexInt32   `json:"index"`
	Events []common.HexInt32 `json:"events"`
	Logs   []module.EventLog `json:"logs,omitempty"`
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

	bm := wss.chain.BlockManager()
	sm := wss.chain.ServiceManager()
	if bm == nil || sm == nil {
		_ = wss.response(int(jsonrpc.ErrorCodeServer), "Stopped")
		return nil
	}

	h := er.Height.Value
	if gh := wss.chain.GenesisStorage().Height(); gh > h {
		_ = wss.response(int(jsonrpc.ErrorCodeInvalidParams),
			fmt.Sprintf("given height(%d) is lower than genesis height(%d)", h, gh))
		return nil
	}

	_ = wss.response(0, "")

	ech := make(chan error)
	go readLoop(wss.c, ech)

	var bch <-chan module.Block

loop:
	for {
		bch, err = bm.WaitForBlock(h)
		if err != nil {
			break loop
		}
		select {
		case err = <-ech:
			break loop
		case blk, ok := <-bch:
			if !ok {
				break loop
			}
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
				if es, el, err := er.matchWithLogs(r, er.Logs.Value != 0); err == nil && len(es) > 0 {
					var en EventNotification
					en.Height.Value = h
					en.Hash = blk.ID()
					en.Index.Value = index
					en.Events = es
					en.Logs = el
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
	f.numOfArgs = len(f.Indexed) + len(f.Data)
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

// bytesEqual check equality of byte slice.
// But it doesn't assume nil as empty bytes.
func bytesEqual(b1 []byte, b2 []byte) bool {
	if b1 == nil && b2 == nil {
		return true
	}
	if b1 == nil || b2 == nil {
		return false
	}
	return bytes.Equal(b1, b2)
}

func (f *EventFilter) matchWithLogs(r module.Receipt, includeLogs bool) ([]common.HexInt32, []module.EventLog, error) {
	var indexes []common.HexInt32
	var logs []module.EventLog
	if err := f.filterFunc(r, func(idx int, log module.EventLog) {
		indexes = append(indexes, common.HexInt32{Value: int32(idx)})
		if includeLogs {
			logs = append(logs, log)
		}
	}); err != nil {
		return nil, nil, err
	}
	return indexes, logs, nil
}

func (f *EventFilter) match(r module.Receipt) ([]common.HexInt32, bool) {
	eventIndexes := make([]common.HexInt32, 0)
	if err := f.filterFunc(r, func(idx int, log module.EventLog) {
		eventIndexes = append(eventIndexes, common.HexInt32{int32(idx)})
	}); err != nil {
		return []common.HexInt32{}, false
	} else {
		return eventIndexes, len(eventIndexes) > 0
	}
	return eventIndexes, false
}

func (f *EventFilter) filterFunc(r module.Receipt, v func(idx int, log module.EventLog)) error {
	if r.LogsBloom().Contain(f.lb) {
	loop:
		for it, idx := r.EventLogIterator(), 0; it.Has(); _, idx = it.Next(), idx+1 {
			el, err := it.Get()
			if err != nil {
				return err
			}

			if bytes.Equal([]byte(f.Signature), el.Indexed()[0]) {
				if f.Addr != nil && !el.Address().Equal(f.Addr) {
					continue loop
				}
				if f.numOfArgs > 0 {
					if (len(el.Indexed()) + len(el.Data())) <= f.numOfArgs {
						continue loop
					}

					for i, arg := range f.indexedBSs {
						if arg != nil && !bytesEqual(arg, el.Indexed()[i+1]) {
							continue loop
						}
					}
					for i, arg := range f.dataBSs {
						if arg != nil && !bytesEqual(arg, el.Data()[i]) {
							continue loop
						}
					}
				}
				v(idx, el)
			}
		}
	}
	return nil
}
