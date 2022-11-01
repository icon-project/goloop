package server

import (
	"encoding/base64"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type BTPRequest struct {
	Height    common.HexInt64 `json:"height"`
	NetworkId common.HexInt64 `json:"networkID"`
	ProofFlag bool            `json:"proofFlag"`
	bn        BTPNotification
}

type BTPNotification struct {
	Header common.HexBytes `json:"header"`
	Proof  string          `json:"proof,omitempty"`
}

func (wm *wsSessionManager) RunBtpSession(ctx echo.Context) error {
	var br BTPRequest
	wss, err := wm.initSession(ctx, &br)
	if err != nil {
		return err
	}
	defer wm.StopSession(wss)

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

	block, err := bm.GetLastBlock()
	nw, err := sm.BTPNetworkFromResult(block.Result(), br.NetworkId.Value)
	if err != nil {
		wm.logger.Infof("not found nid=%d height=%d, err:%+v\n", br.NetworkId.Value, h, err)
		return nil
	}

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
			if nw.StartHeight()+1 <= h {
				chain, ok := ctx.Get("chain").(module.Chain)
				if chain == nil || !ok {
					wm.logger.Infof("err:%+v\n", err)
					break loop
				}

				cs := chain.Consensus()
				nw, err := sm.BTPNetworkFromResult(blk.Result(), br.NetworkId.Value)
				if !nw.Open() {
					wm.logger.Infof("network is closed (height=%d, err:%+v)\n", h, err)
					_ = wss.response(int(jsonrpc.ErrorCodeInvalidParams),
						fmt.Sprintf("network is closed ( height(%d) , networkId(%d)", h, br.NetworkId))
					break loop
				}

				var flag uint
				if br.ProofFlag == true && blk.Height() != nw.StartHeight()+1 {
					flag = module.FlagBTPBlockHeader | module.FlagBTPBlockProof
				} else {
					flag = module.FlagBTPBlockHeader
				}

				btpBlock, proof, err := cs.GetBTPBlockHeaderAndProof(blk, br.NetworkId.Value, flag)
				if err == nil {
					br.bn.Header = btpBlock.HeaderBytes()
					if flag == module.FlagBTPBlockHeader|module.FlagBTPBlockProof {
						br.bn.Proof = base64.StdEncoding.EncodeToString(proof)
					}

					if err = wss.WriteJSON(&br.bn); err != nil {
						wm.logger.Infof("fail to write json BtpNotification err:%+v\n", err)
						break loop
					}
				}
			}
		}
		h++
	}
	wm.logger.Warnf("%+v\n", err)
	return nil
}
