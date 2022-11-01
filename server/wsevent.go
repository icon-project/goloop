package server

import (
	"bytes"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/txresult"
)

type EventRequest struct {
	EventFilter
	Height common.HexInt64 `json:"height"`
	Logs   common.HexInt32 `json:"logs,omitempty""`

	Filters EventFilters `json:"eventFilters,omitempty"`
}

type EventFilters []*EventFilter

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

// FilteredByLogBloom returns applicable event filters.
// If there is no event filters, then it returns false along with filters.
func (fs EventFilters) FilteredByLogBloom(lb module.LogsBloom) (EventFilters, bool) {
	filters := make([]*EventFilter, len(fs))
	contained := false
	for idx, filter := range fs {
		if filter == nil {
			continue
		}
		if lb.Contain(filter.lb) {
			filters[idx] = filter
			contained = true
		}
	}
	return filters, contained
}

func (fs EventFilters) MatchEvents(r module.Receipt, includeLogs bool) ([]common.HexInt32, []module.EventLog, error) {
	var indexes []common.HexInt32
	var logs []module.EventLog
	if err := fs.filterEvents(r, func(fi, idx int, log module.EventLog) {
		indexes = append(indexes, common.HexInt32{Value: int32(idx)})
		if includeLogs {
			logs = append(logs, log)
		}
	}); err != nil {
		return nil, nil, err
	} else {
		return indexes, logs, nil
	}
}

func (fs EventFilters) filterEvents(r module.Receipt, v func(fi, idx int, log module.EventLog)) error {
	filters, contained := fs.FilteredByLogBloom(r.LogsBloom())
	if !contained {
		return nil
	}
	for it, idx := r.EventLogIterator(), 0; it.Has(); _, idx = it.Next(), idx+1 {
		el, err := it.Get()
		if err != nil {
			return err
		}
		for fi, f := range filters {
			if f == nil {
				continue
			}
			if f.MatchLog(el) {
				v(fi, idx, el)
				break
			}
		}
	}
	return nil
}

func (wm *wsSessionManager) RunEventSession(ctx echo.Context) error {
	var er EventRequest
	wss, err := wm.initSession(ctx, &er)
	if err != nil {
		return err
	}
	defer wm.StopSession(wss)

	filters, err := er.Compile()
	if err != nil {
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

	ech := make(chan error, 1)
	wss.RunLoop(ech)

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
			filters2, contained := filters.FilteredByLogBloom(blk.LogsBloom())
			if !contained {
				break
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
				if es, el, err := filters2.MatchEvents(r, er.Logs.Value != 0); err == nil && len(es) > 0 {
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

func (f *EventFilter) Compile() error {
	lb := txresult.NewLogsBloom(nil)
	if f.Addr != nil {
		lb.AddAddressOfLog(f.Addr)
	}
	f.numOfArgs = len(f.Indexed) + len(f.Data)
	name, pts := txresult.DecomposeEventSignature(f.Signature)
	if len(name) == 0 || pts == nil || len(pts) < f.numOfArgs {
		return errors.NewBase(errors.IllegalArgumentError, "bad event signature")
	}
	for idx, pt := range pts {
		dt := scoreapi.DataTypeOf(pt)
		if !dt.UsableForEvent() {
			return errors.IllegalArgumentError.Errorf("InvalidParameterType(idx=%d,type=%s)", idx, pt)
		}
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

func (f *EventFilter) MatchEvents(r module.Receipt, includeLogs bool) ([]common.HexInt32, []module.EventLog, error) {
	var indexes []common.HexInt32
	var logs []module.EventLog
	if err := f.filterEvents(r, func(idx int, log module.EventLog) {
		indexes = append(indexes, common.HexInt32{Value: int32(idx)})
		if includeLogs {
			logs = append(logs, log)
		}
	}); err != nil {
		return nil, nil, err
	}
	return indexes, logs, nil
}

func (f *EventFilter) MatchLog(el module.EventLog) bool {
	if bytes.Equal([]byte(f.Signature), el.Indexed()[0]) {
		if f.Addr != nil && !el.Address().Equal(f.Addr) {
			return false
		}
		if f.numOfArgs > 0 {
			if len(el.Indexed()) <= len(f.indexedBSs) {
				return false
			}
			if len(el.Data()) < len(f.dataBSs) {
				return false
			}

			for i, arg := range f.indexedBSs {
				if arg != nil && !bytesEqual(arg, el.Indexed()[i+1]) {
					return false
				}
			}
			for i, arg := range f.dataBSs {
				if arg != nil && !bytesEqual(arg, el.Data()[i]) {
					return false
				}
			}
		}
		return true
	} else {
		return false
	}
}

func (f *EventFilter) filterEvents(r module.Receipt, v func(idx int, log module.EventLog)) error {
	if r.LogsBloom().Contain(f.lb) {
		for it, idx := r.EventLogIterator(), 0; it.Has(); _, idx = it.Next(), idx+1 {
			el, err := it.Get()
			if err != nil {
				return err
			}

			if f.MatchLog(el) {
				v(idx, el)
			}
		}
	}
	return nil
}

func (f *EventRequest) Compile() (EventFilters, error) {
	var filters []*EventFilter
	if len(f.Filters) > 0 {
		if len(f.Signature) != 0 {
			return nil, errors.New("both eventFilters and event is used")
		}
		filters = f.Filters
	} else {
		filters = []*EventFilter{&f.EventFilter}
	}
	for idx, filter := range filters {
		if filter == nil {
			return nil, fmt.Errorf("invalid filter idx:%d", idx)
		}
		if err := filter.Compile(); err != nil {
			return nil, err
		}
	}
	return filters, nil
}
