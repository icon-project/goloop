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
	ProofFlag common.HexInt64 `json:"proofFlag"`
	bn        BTPNotification
}

type BTPNotification struct {
	Header common.HexBytes `json:"header"`
	Proof  string          `json:"proof"`
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
		case blk := <-bch:
			chain, ok := ctx.Get("chain").(module.Chain)
			if chain == nil || !ok {
				wm.logger.Infof("err:%+v\n", err)
				break loop
			}
			cs := chain.Consensus()
			btpBlock, _, err := cs.GetBTPBlockHeaderAndProof(blk, br.NetworkId.Value, module.FlagBTPBlockHeader)
			if err == nil {
				br.bn.Header = btpBlock.HeaderBytes()
				if br.ProofFlag.Value == module.FlagBTPBlockProof {
					_, proof, err := cs.GetBTPBlockHeaderAndProof(blk, br.NetworkId.Value, module.FlagBTPBlockProof)
					if err != nil {
						wm.logger.Infof("fail to get a BTP block proof for height=%d, err:%+v\n", h, err)
					}
					br.bn.Proof = base64.StdEncoding.EncodeToString(proof)
				}
				if err = wss.WriteJSON(&br.bn); err != nil {
					wm.logger.Infof("fail to write json BtpNotification err:%+v\n", err)
					break loop
				}
			}
		}
		h++
	}
	wm.logger.Warnf("%+v\n", err)
	return nil
}
