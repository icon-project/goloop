/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icon

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/icon/icconsensus"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/icnetwork"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/iccache"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/platform/basic"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type platform struct {
	calculator iiss.CalculatorHolder
	base       string
}

func (p *platform) NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	if err := blockv1.CheckAndApplyPatch(dbase); err != nil {
		return nil, err
	}
	return newContractManager(p, dbase, dir, logger)
}

func (p *platform) NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot {
	// TODO return valid ExtensionSnapshot(not nil) which can return valid ExtensionState.
	//  with that state, we may change state of extension.
	//  For initial state, the snapshot returns nil for Bytes() method.
	if len(raw) == 0 {
		return nil
	}
	dbase = iccache.AttachStateNodeCache(dbase)
	return iiss.NewExtensionSnapshot(dbase, raw)
}

func (p *platform) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	return iiss.NewExtensionSnapshotWithBuilder(builder, raw)
}

func (p *platform) ToRevision(value int) module.Revision {
	return icmodule.ValueToRevision(value)
}

func (p *platform) NewBaseTransaction(wc state.WorldContext) (module.Transaction, error) {
	// calculate issued i-score and amount balance. No changes on world context.
	es := p.getExtensionState(wc, nil)
	if es == nil || !es.IsDecentralized() {
		return nil, nil
	}

	t := common.HexInt64{Value: wc.BlockTimeStamp()}
	v := common.HexUint16{Value: module.TransactionVersion3}
	prep, issue := iiss.GetIssueData(es)
	data := make(map[string]interface{})
	if prep != nil {
		data["prep"] = prep
	}
	if issue != nil {
		data["result"] = issue
	}
	mtx := map[string]interface{}{
		"timestamp": t,
		"version":   v,
		"dataType":  "base",
		"data":      data,
	}
	bs, err := json.Marshal(mtx)
	if err != nil {
		return nil, err
	}
	tx, err := transaction.NewTransactionFromJSON(bs)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (p *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot, logger log.Logger) {
	// Start background calculator if it's not started.
	p.calculator.Start(ess, logger)
}

func checkBaseTX(txs module.TransactionList) bool {
	tx, err := txs.Get(0)
	if err == nil {
		return iiss.CheckBaseTX(tx)
	} else {
		return false
	}
}

func (p *platform) OnValidateTransactions(wc state.WorldContext, patches, txs module.TransactionList) error {
	es := p.getExtensionState(wc, nil)
	needBaseTX := es != nil && es.IsDecentralized()
	if hasBaseTX := checkBaseTX(txs); needBaseTX == hasBaseTX {
		return nil
	} else {
		if needBaseTX {
			return errors.IllegalArgumentError.New("NoBaseTransaction")
		} else {
			return errors.IllegalArgumentError.New("InvalidBaseTransaction")
		}
	}
}

func (p *platform) OnExecutionBegin(wc state.WorldContext, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	if revision >= icmodule.Revision12 && revision < icmodule.RevisionFixRLPBug {
		// Set batch data root storing block batch data and tx batch data
		wc.(contract.Context).SetProperty(BatchKey, new(batchRoot).Init(nil))
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	return es.OnExecutionBegin(iiss.NewWorldContext(wc, logger))
}

func (p *platform) OnExecutionEnd(wc state.WorldContext, er base.ExecutionResult, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	var totalFee *big.Int
	if revision < icmodule.RevisionEnableIISS3 {
		// Use virtual fee instead of total fee for IISS 2.x
		totalFee = er.VirtualFee()
	} else {
		totalFee = er.TotalFee()
	}

	txInfo := wc.TransactionInfo()
	txIndex := int(txInfo.Index)
	tlogger := trace.LoggerOf(logger)
	tlogger.OnTransactionStart(txIndex, nil)
	defer tlogger.OnTransactionEnd(txIndex, nil, nil, wc.Treasury(), wc.Revision(), nil)

	return es.OnExecutionEnd(iiss.NewWorldContext(wc, logger), totalFee, p.calculator.Get())
}

func (p *platform) OnTransactionEnd(wc state.WorldContext, logger log.Logger, rct txresult.Receipt) error {
	success := rct.Status() == module.StatusSuccess
	// Apply stored tx batch data
	if value := wc.(contract.Context).GetProperty(BatchKey); value != nil {
		root := value.(*batchRoot)
		root.handleTxBatch(success)
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	return es.OnTransactionEnd(wc.BlockHeight(), success)
}

func (p *platform) Term() {
	// Terminate
}

func (p *platform) DefaultBlockVersionFor(cid int) int {
	if cid == CIDForMainNet || cid == CIDForTestNet {
		return module.BlockVersion1
	}
	return basic.Platform.DefaultBlockVersionFor(cid)
}

func (p *platform) NewBlockHandlers(c base.Chain) []base.BlockHandler {
	if p.DefaultBlockVersionFor(c.CID()) != module.BlockVersion1 {
		return basic.Platform.NewBlockHandlers(c)
	}
	return []base.BlockHandler{
		blockv1.NewHandler(c),
		block.NewBlockV2Handler(c),
	}
}

func (p *platform) NewConsensus(c base.Chain, walDir string) (module.Consensus, error) {
	if p.DefaultBlockVersionFor(c.CID()) != module.BlockVersion1 {
		return basic.Platform.NewConsensus(c, walDir)
	}
	header, lastVotes, err := p.GetBlockV1Proof()
	if err != nil {
		return nil, err
	}
	cs, err := icconsensus.New(c, walDir, nil, nil, header, lastVotes, 0)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

func (t *platform) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return func(bytes []byte) module.CommitVoteSet {
		vs := consensus.NewCommitVoteSetFromBytes(bytes)
		if vs != nil {
			return vs
		}
		vl, _ := blockv0.NewBlockVotesFromBytes(bytes)
		return vl
	}
}

const (
	BlockV1ProofFile = "block_v1_proof.bin"
)

type BlockV1Proof struct {
	MerkleHeader *hexary.MerkleHeader
	LastVotes    *blockv0.BlockVoteList
}

func (p *platform) GetBlockV1Proof() (*hexary.MerkleHeader, *blockv0.BlockVoteList, error) {
	file := path.Join(p.base, BlockV1ProofFile)
	reader, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()
	bp := new(BlockV1Proof)
	if err := codec.BC.Unmarshal(reader, bp); err != nil {
		return nil, nil, err
	}
	return bp.MerkleHeader, bp.LastVotes, nil
}

func (p *platform) SetBlockV1Proof(root []byte, size int64, votes *blockv0.BlockVoteList) error {
	hdr := &BlockV1Proof{
		MerkleHeader: &hexary.MerkleHeader{
			RootHash: root,
			Leaves:   size,
		},
		LastVotes: votes,
	}
	bs, err := codec.BC.MarshalToBytes(hdr)
	if err != nil {
		return err
	}
	file := path.Join(p.base, BlockV1ProofFile)
	return ioutil.WriteFile(file, bs, os.FileMode(0500))
}

func NewPlatform(base string, cid int) (base.Platform, error) {
	return &platform{
		base: base,
	}, nil
}

func init() {
	iiss.RegisterBaseTx()
}

func (p *platform) getExtensionState(wc state.WorldContext, logger log.Logger) *iiss.ExtensionStateImpl {
	es := wc.GetExtensionState()
	if es == nil {
		return nil
	}
	esi, ok := es.(*iiss.ExtensionStateImpl)
	if !ok {
		return nil
	}
	if logger != nil {
		esi.SetLogger(icutils.NewIconLogger(logger))
	}
	return esi
}


func (p *platform) ShowDiff(ctx service.DiffContext, name string, e, r []byte) error {
	switch name {
	case service.DNExtension:
		return ShowExtensionDiff(ctx, e, r)
	default:
		return errors.IllegalArgumentError.Errorf("UnknownObject(name=%s)", name)
	}
}

func (p *platform) DoubleSignDecoder() module.DoubleSignDataDecoder {
	return consensus.DecodeDoubleSignData
}

func (p *platform) NewSeedState(wc state.WorldSnapshot) (module.SeedState, error) {
	return icnetwork.NewSeedState(wc)
}
