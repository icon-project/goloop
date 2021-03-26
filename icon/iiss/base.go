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

// -build base

package iiss

import (
	"bytes"
	"encoding/json"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type baseDataJSON struct {
	PRep   json.RawMessage `json:"prep"`
	Result json.RawMessage `json:"result"`
}

func parseBaseData(data []byte) (*baseDataJSON, error) {
	if data == nil {
		return nil, nil
	}
	jso := new(baseDataJSON)
	jd := json.NewDecoder(bytes.NewBuffer(data))
	jd.DisallowUnknownFields()
	if err := jd.Decode(jso); err != nil {
		return nil, err
	}
	return jso, nil
}

type baseV3Data struct {
	Version   common.HexUint16 `json:"version"`
	From      *common.Address  `json:"from,omitempty"` // it should be nil
	TimeStamp common.HexInt64  `json:"timestamp"`
	DataType  string           `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

func (tx *baseV3Data) calcHash() ([]byte, error) {
	sha := bytes.NewBuffer(nil)
	sha.Write([]byte("icx_sendTransaction"))

	// data
	if tx.Data != nil {
		sha.Write([]byte(".data."))
		if len(tx.Data) > 0 {
			var obj interface{}
			if err := json.Unmarshal(tx.Data, &obj); err != nil {
				return nil, err
			}
			if bs, err := transaction.SerializeValue(obj); err != nil {
				return nil, err
			} else {
				sha.Write(bs)
			}
		}
	}

	// dataType
	sha.Write([]byte(".dataType."))
	sha.Write([]byte(tx.DataType))

	// timestamp
	sha.Write([]byte(".timestamp."))
	sha.Write([]byte(tx.TimeStamp.String()))

	// version
	sha.Write([]byte(".version."))
	sha.Write([]byte(tx.Version.String()))

	return crypto.SHA3Sum256(sha.Bytes()), nil
}

type baseV3 struct {
	baseV3Data

	id    []byte
	hash  []byte
	bytes []byte
}

func (tx *baseV3) Version() int {
	return module.TransactionVersion3
}

func (tx *baseV3) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	wc := ctx.GetFuture(lq)
	wc.WorldVirtualState().Ensure()

	return wc, nil
}

func (tx *baseV3) Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error) {
	info := ctx.TransactionInfo()
	if info == nil {
		return nil, errors.InvalidStateError.New("TransactionInfoUnavailable")
	}
	if info.Index != 0 {
		return nil, errors.CriticalFormatError.New("BaseMustBeTheFirst")
	}

	cc := contract.NewCallContext(ctx, ctx.GetStepLimit(state.StepLimitTypeInvoke), false)
	defer cc.Dispose()

	if err := handleConsensusInfo(cc); err != nil {
		return nil, err
	}

	if err := handleICXIssue(cc, tx.Data); err != nil {
		return nil, err
	}

	// Make a receipt
	r := txresult.NewReceipt(ctx.Database(), ctx.Revision(), tx.To())
	cc.GetEventLogs(r)
	r.SetResult(module.StatusSuccess, new(big.Int), new(big.Int), nil)
	return r, nil
}

func handleConsensusInfo(cc contract.CallContext) error {
	es := cc.GetExtensionState().(*ExtensionStateImpl)
	csi := cc.ConsensusInfo()
	if csi == nil {
		//return errors.CriticalUnknownError.Errorf("There is no consensus Info.")
		return nil
	}
	if !es.IsDecentralized() {
		return nil
	}
	// if PrepManager is not ready, it returns immediately
	if es.pm.GetPRepByNode(csi.Proposer()) == nil {
		return nil
	}

	// Convert node address to owner address
	owners, trueVoters, err := nodeToOwner(es.pm, csi)
	if err != nil {
		return err
	}

	// Make block produce Info for calculator
	if err = addBlockProduce(cc, trueVoters); err != nil {
		return err
	}

	// Update Block vote stats
	voted := csi.Voted()
	return updateBlockVoteStats(cc, owners, voted)
}

func nodeToOwner(pm *PRepManager, csi module.ConsensusInfo) ([]module.Address, []module.Address, error) {
	voters := csi.Voters()
	if voters == nil {
		return nil, nil, errors.Errorf("Voters are nil")
	}
	voted := csi.Voted()

	size := voters.Len()
	owners := make([]module.Address, size, size)
	trueVoters := make([]module.Address, 0)

	for i := 0; i < size; i += 1 {
		v, _ := voters.Get(i)
		owner := pm.GetOwnerByNode(v.Address())
		if owner == nil {
			return nil, nil, errors.Errorf("Owner not found: %s", v.Address())
		}
		owners[i] = owner
		if voted[i] {
			trueVoters = append(trueVoters, owner)
		}
	}

	return owners, trueVoters, nil
}

// addBlockProduce makes Block produce Info for calculator
// trueVoters contain the addresses which vote for block approval
func addBlockProduce(cc contract.CallContext, trueVoters []module.Address) error {
	es := cc.GetExtensionState().(*ExtensionStateImpl)
	csi := cc.ConsensusInfo()

	// if PRepManager is not ready, it returns immediately
	proposer := csi.Proposer()
	po := es.pm.GetOwnerByNode(proposer)
	term := es.State.GetTerm()

	return es.Front.AddBlockProduce(
		int(cc.BlockHeight()-term.StartHeight()),
		po,
		trueVoters,
	)
}

// updateBlockVoteStats updates validation state of each PRep and checks PReps for penalty
func updateBlockVoteStats(cc contract.CallContext, owners []module.Address, voted []bool) error {
	es := cc.GetExtensionState().(*ExtensionStateImpl)

	for i, owner := range owners {
		if err := es.UpdateBlockVoteStats(cc, owner, voted[i]); err != nil {
			return err
		}
		if !voted[i] {
			if err := es.handlePenalty(cc, owner); err != nil {
				return err
			}
		}
	}

	es.pm.Sort()
	return nil
}

func handleICXIssue(cc contract.CallContext, data []byte) error {
	// parse Issue Info. from TX data
	bd, err := parseBaseData(data)
	if err != nil {
		return scoreresult.InvalidParameterError.Wrap(err, "InvalidData")
	}

	var iPrep *IssuePRepJSON
	var iResult *IssueResultJSON
	if bd != nil {
		iPrep, err = parseIssuePRepData(bd.PRep)
		if err != nil {
			return scoreresult.InvalidParameterError.Wrap(err, "InvalidData")
		}
		iResult, err = parseIssueResultData(bd.Result)
		if err != nil {
			return scoreresult.InvalidParameterError.Wrap(err, "InvalidData")
		}
	}

	// get Issue result from state
	es := cc.GetExtensionState().(*ExtensionStateImpl)
	prep, result := GetIssueData(es)
	// there is no issue data
	if iPrep == nil && iResult == nil && prep == nil && result == nil {
		return nil
	}

	// check Issue result
	if (iPrep != nil && !iPrep.equal(prep)) || (iResult != nil && !iResult.equal(result)) {
		return scoreresult.InvalidParameterError.New("Invalid issue data")
	}

	// transfer issued ICX to treasury
	tr := cc.GetAccountState(cc.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Add(tb, result.Issue.Value()))

	// increase total supply
	if err = icutils.IncreaseTotalSupply(cc, result.Issue.Value()); err != nil {
		return err
	}

	// write Issue Info
	is, err := es.State.GetIssue()
	if err != nil {
		return scoreresult.InvalidContainerAccessError.Wrap(err, "Failed to get issue Info.")
	}
	issue := is.Clone()
	issue.TotalIssued.Add(issue.TotalIssued, result.GetTotalReward())
	if result.ByFee.Sign() != 0 {
		issue.PrevBlockFee.Sub(issue.PrevBlockFee, result.ByFee.Value())
	}
	if result.ByOverIssuedICX.Sign() != 0 {
		issue.OverIssued.Sub(issue.OverIssued, result.ByOverIssuedICX.Value())
	}
	if err = es.State.SetIssue(issue); err != nil {
		return scoreresult.InvalidContainerAccessError.Wrap(err, "Failed to set issue Info.")
	}

	// make event log
	if prep != nil {
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("PRepIssued(int,int,int,int)")},
			[][]byte{
				intconv.BigIntToBytes(prep.IRep.Value()),
				intconv.BigIntToBytes(prep.RRep.Value()),
				intconv.BigIntToBytes(prep.TotalDelegation.Value()),
				intconv.BigIntToBytes(prep.Value.Value()),
			},
		)
	}
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("ICXIssued(int,int,int,int)")},
		[][]byte{
			intconv.BigIntToBytes(result.ByFee.Value()),
			intconv.BigIntToBytes(result.ByOverIssuedICX.Value()),
			intconv.BigIntToBytes(result.Issue.Value()),
			intconv.BigIntToBytes(issue.OverIssued),
		},
	)
	term := es.State.GetTerm()
	if cc.BlockHeight() == term.StartHeight() {
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("TermStarted(int,int,int)")},
			[][]byte{
				intconv.Int64ToBytes(int64(term.Sequence())),
				intconv.Int64ToBytes(term.StartHeight()),
				intconv.Int64ToBytes(term.GetEndBlockHeight()),
			},
		)
	}
	return nil
}

func (tx *baseV3) Dispose() {
	//panic("implement me")
}

func (tx *baseV3) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *baseV3) ID() []byte {
	if tx.id == nil {
		if bs, err := tx.baseV3Data.calcHash(); err != nil {
			panic(err)
		} else {
			tx.id = bs
		}
	}
	return tx.id
}

func (tx *baseV3) From() module.Address {
	return state.SystemAddress
}

func (tx *baseV3) Bytes() []byte {
	if tx.bytes == nil {
		if bs, err := codec.BC.MarshalToBytes(&tx.baseV3Data); err != nil {
			panic(err)
		} else {
			tx.bytes = bs
		}
	}
	return tx.bytes
}

func (tx *baseV3) Hash() []byte {
	if tx.hash == nil {
		tx.hash = crypto.SHA3Sum256(tx.Bytes())
	}
	return tx.hash
}

func (tx *baseV3) Verify() error {
	return nil
}

func (tx *baseV3) ToJSON(version module.JSONVersion) (interface{}, error) {
	jso := map[string]interface{}{
		"version":   &tx.baseV3Data.Version,
		"timestamp": &tx.baseV3Data.TimeStamp,
		"dataType":  tx.baseV3Data.DataType,
		"data":      tx.baseV3Data.Data,
	}
	jso["txHash"] = common.HexBytes(tx.ID())
	return jso, nil
}

func (tx *baseV3) ValidateNetwork(nid int) bool {
	return true
}

func (tx *baseV3) PreValidate(wc state.WorldContext, update bool) error {
	return nil
}

func (tx *baseV3) GetHandler(cm contract.ContractManager) (transaction.Handler, error) {
	return tx, nil
}

func (tx *baseV3) Timestamp() int64 {
	return tx.baseV3Data.TimeStamp.Value
}

func (tx *baseV3) Nonce() *big.Int {
	return nil
}

func (tx *baseV3) To() module.Address {
	return state.SystemAddress
}

func (tx *baseV3) IsSkippable() bool {
	return false
}

func checkBaseV3JSON(jso map[string]interface{}) bool {
	if d, ok := jso["dataType"]; !ok || d != "base" {
		return false
	}
	if v, ok := jso["version"]; !ok || v != "0x3" {
		return false
	}
	return true
}

func parseBaseV3JSON(bs []byte, raw bool) (transaction.Transaction, error) {
	tx := new(baseV3)
	if err := json.Unmarshal(bs, &tx.baseV3Data); err != nil {
		return nil, transaction.InvalidFormat.Wrap(err, "InvalidJSON")
	}
	if tx.baseV3Data.From != nil {
		return nil, transaction.InvalidFormat.New("InvalidFromValue(NonNil)")
	}
	return tx, nil
}

type baseV3Header struct {
	Version common.HexUint16 `json:"version"`
	From    *common.Address  `json:"from"` // it should be nil
}

func checkBaseV3Bytes(bs []byte) bool {
	var vh baseV3Header
	if _, err := codec.BC.UnmarshalFromBytes(bs, &vh); err != nil {
		return false
	}
	return vh.From == nil
}

func parseBaseV3Bytes(bs []byte) (transaction.Transaction, error) {
	tx := new(baseV3)
	if _, err := codec.BC.UnmarshalFromBytes(bs, &tx.baseV3Data); err != nil {
		return nil, err
	}
	return tx, nil
}

func RegisterBaseTx() {
	transaction.RegisterFactory(&transaction.Factory{
		Priority:    15,
		CheckJSON:   checkBaseV3JSON,
		ParseJSON:   parseBaseV3JSON,
		CheckBinary: checkBaseV3Bytes,
		ParseBinary: parseBaseV3Bytes,
	})
}
