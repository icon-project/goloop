package server

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type BlockRequest struct {
	Height       common.HexInt64 `json:"height"`
	EventFilters []*EventFilter  `json:"eventFilters,omitempty"`
	Logs         common.HexInt32 `json:"logs,omitempty"`
	bn           BlockNotification
}

type BlockNotification struct {
	Hash    common.HexBytes       `json:"hash"`
	Height  common.HexInt64       `json:"height"`
	Indexes [][]common.HexInt32   `json:"indexes,omitempty"`
	Events  [][][]common.HexInt32 `json:"events,omitempty"`
	Logs    [][][]module.EventLog `json:"logs,omitempty"`
}

func (wm *wsSessionManager) RunBlockSession(ctx echo.Context) error {
	var br BlockRequest
	wss, err := wm.initSession(ctx, &br)
	if err != nil {
		return err
	}
	defer wm.StopSession(wss)

	if err := br.Compile(); err != nil {
		_ = wss.response(int(jsonrpc.ErrorCodeInvalidParams), err.Error())
		return nil
	}

	bm := wss.chain.BlockManager()
	sm := wss.chain.ServiceManager()
	if bm == nil || sm == nil {
		_ = wss.response(int(jsonrpc.ErrorCodeServer), "Stopped")
		return nil
	}

	h := br.Height.Value
	if gh := wss.chain.GenesisStorage().Height(); gh > h {
		_ = wss.response(int(jsonrpc.ErrorCodeInvalidParams),
			fmt.Sprintf("given height(%d) is lower than genesis height(%d)", h, gh))
		return nil
	}

	_ = wss.response(0, "")

	ech := make(chan error, 1)
	wss.RunLoop(ech)

	var bch <-chan module.Block
	indexes := make([][]common.HexInt32, len(br.EventFilters))
	events := make([][][]common.HexInt32, len(br.EventFilters))
	eventLogs := make([][][]module.EventLog, len(br.EventFilters))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = make([]common.HexInt32, 0)
		events[i] = make([][]common.HexInt32, 0)
		eventLogs[i] = make([][]module.EventLog, 0)
	}
	var rl module.ReceiptList
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
			br.bn.Height = common.HexInt64{Value: h}
			br.bn.Hash = blk.ID()
			if rl != nil {
				rl = nil
			}
			if len(br.bn.Indexes) > 0 {
				br.bn.Indexes = indexes[:0]
				br.bn.Events = events[:0]
				for i := range indexes {
					indexes[i] = indexes[i][:0]
				}
				for i := range events {
					events[i] = events[i][:0]
				}
				if br.Logs.Value != 0 {
					br.bn.Logs = eventLogs[:0]
					for i := range eventLogs {
						eventLogs[i] = eventLogs[i][:0]
					}
				}
			}
			lb := blk.LogsBloom()
			for i, f := range br.EventFilters {
				if lb.Contain(f.lb) {
					if rl == nil {
						rl, err = sm.ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
						if err != nil {
							break loop
						}
					}
					index := int32(0)
					for rit := rl.Iterator(); rit.Has(); rit.Next() {
						r, err := rit.Get()
						if err != nil {
							break loop
						}
						if es, logs, err := f.MatchEvents(r, br.Logs.Value != 0); err == nil && len(es) > 0 {
							if len(br.bn.Indexes) < 1 {
								br.bn.Indexes = indexes[:]
								br.bn.Events = events[:]
								if br.Logs.Value != 0 {
									br.bn.Logs = eventLogs[:]
								}
							}
							br.bn.Indexes[i] = append(br.bn.Indexes[i], common.HexInt32{Value: index})
							br.bn.Events[i] = append(br.bn.Events[i], es)
							if br.Logs.Value != 0 {
								br.bn.Logs[i] = append(br.bn.Logs[i], logs)
							}
						}
						index++
					}
				}
			}
			if err = wss.WriteJSON(&br.bn); err != nil {
				wm.logger.Infof("fail to write json BlockNotification err:%+v\n", err)
				break loop
			}
		}
		h++
	}
	wm.logger.Warnf("%+v\n", err)
	return nil
}

func (r *BlockRequest) Compile() error {
	for i, f := range r.EventFilters {
		if f == nil {
			return fmt.Errorf("null filter idx:%d", i)
		}
		if err := f.Compile(); err != nil {
			return fmt.Errorf("fail to compile idx:%d, err:%v", i, err)
		}
	}
	return nil
}
