/*
 * Copyright 2021 ICON Foundation
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

package ictest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/test"
)

type BlockV0Generator struct {
	t              *testing.T
	node           *test.Node
	validators     []module.Wallet
	reps           *blockv0.RepsList
	validatorsInTx []module.Wallet
	repsInTx       *blockv0.RepsList
	last           blockv0.Block
	lastVotes      *blockv0.BlockVoteList
	txs            []blockv0.Transaction
	repsByHash     map[string]*blockv0.RepsList
	receiptByHash  map[string]module.Receipt
	blocks         []blockv0.Block
	lastTr         module.Transition
}

const defaultGenesis = `
{
  "version": "0.1a",
  "prev_block_hash": "",
  "merkle_tree_root_hash": "5aa2453a84ba2fb1e3394b9e3471f5dcebc6225fc311a97ca505728153b9d246",
  "time_stamp": 0,
  "confirmed_transaction_list": [
    {
      "accounts": [
        {
          "name": "god",
          "address": "hx54f7853dc6481b670caf69c5a27c7c8fe5be8269",
          "balance": "0x2961fff8ca4a62327800000"
        },
        {
          "name": "treasury",
          "address": "hx1000000000000000000000000000000000000000",
          "balance": "0x0"
        }
      ],
      "message": "A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \"to be\" but the fabric of the rhizome is the conjunction, \"and ... and ...and...\"This conjunction carries enough force to shake and uproot the verb \"to be.\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\n\n - Mille Plateaux, Gilles Deleuze \u0026 Felix Guattari\n\n\"Hyperconnect the world\""
    }
  ],
  "block_hash": "cf43b3fd45981431a0e64f79d07bfcf703e064b73b802c5f32834eec72142190",
  "height": 0,
  "peer_id": "",
  "signature": "",
  "next_leader": ""
}
`
func UseICONPlatform() test.FixtureOption {
	return test.UseConfig(&test.FixtureConfig{
		NewPlatform: func(ctx *test.NodeContext) base.Platform {
			return NewPlatform()
		},
	})
}

func NewBlockV0Generator(t *testing.T, genesis string) *BlockV0Generator {
	if len(genesis) == 0 {
		genesis = defaultGenesis
	}
	g := &BlockV0Generator{
		t:             t,
		repsByHash:    make(map[string]*blockv0.RepsList),
		receiptByHash: make(map[string]module.Receipt),
	}
	last, err := blockv0.ParseBlock([]byte(genesis), g)
	bs := last.NormalTransactions()[0].Bytes()
	g.node = test.NewNode(t, test.UseGenesis(string(bs)), UseICONPlatform())
	assert.NoError(t, err)
	g.last = last
	g.blocks = append(g.blocks, g.last)
	itr, err := g.node.SM.CreateInitialTransition(nil, nil)
	assert.NoError(t, err)
	g.lastTr = itr
	// pass validated=true for already committed transition
	g.executeAndAddReceipts(
		g.last.NormalTransactions(), g.last.Height(), g.last.Timestamp(), true,
	)
	return g
}

func (g *BlockV0Generator) executeAndAddReceiptsV0TXs(
	txs []blockv0.Transaction,
	height int64,
	ts int64,
) []byte {
	txs_ := make([]module.Transaction, len(txs))
	for i := range txs {
		txs_[i] = txs[i].Transaction
	}
	return g.executeAndAddReceipts(txs_, height, ts, false)
}

func (g *BlockV0Generator) executeAndAddReceipts(
	txs []module.Transaction,
	height int64,
	ts int64,
	validated bool,
) []byte {
	txl := transaction.NewTransactionListFromSlice(g.node.Chain.Database(), txs)
	tr, err := g.node.SM.CreateTransition(
		g.lastTr, txl, common.NewBlockInfo(height, ts), nil, validated,
	)
	assert.NoError(g.t, err)
	transitionExecute(g.t, tr)
	receipts := tr.NormalReceipts()
	for i, it := 0, receipts.Iterator(); it.Has(); i, _ = i+1, it.Next() {
		r, err := it.Get()
		assert.NoError(g.t, err)
		g.receiptByHash[string(txs[i].ID())] = r
	}
	g.lastTr = tr
	root := blockv0.CalcMerkleRootOfReceiptList(receipts, txl, height)
	return root
}

func (g *BlockV0Generator) Close() {
	g.node.Close()
}

type trCB struct {
	chn chan<- error
}

func (t trCB) OnValidate(transition module.Transition, err error) {
	t.chn <- err
}

func (t trCB) OnExecute(transition module.Transition, err error) {
	t.chn <- err
}

func transitionExecute(t *testing.T, tr module.Transition) {
	chn := make(chan error)
	_, err := tr.Execute(trCB{chn})
	assert.NoError(t, err)
	err = <-chn
	assert.NoError(t, err)
	err = <-chn
	assert.NoError(t, err)
}

func (g *BlockV0Generator) Validators() []module.Wallet {
	return g.validators
}

func (g *BlockV0Generator) ValidatorsInTx() []module.Wallet {
	return g.validatorsInTx
}

func (g *BlockV0Generator) AddTx(tx module.Transaction) {
	g.txs = append(g.txs, blockv0.Transaction{Transaction: tx})
}

func (g *BlockV0Generator) AddSetRandomValidatorsTx(n int) {
	validators := make([]module.Wallet, n)
	addresses := make([]module.Address, n)
	paddresses := make([]*common.Address, n)
	for i := range validators {
		validators[i] = wallet.New()
		addresses[i] = validators[i].Address().(module.Address)
		paddresses[i] = addresses[i].(*common.Address)
	}
	g.validatorsInTx = validators
	g.repsInTx = blockv0.NewRepsList(paddresses...)
	tx := test.NewTx().SetValidators(addresses...)
	g.AddTx(tx)
}

func (g *BlockV0Generator) AddSetNextBlockVersionTx(v int32) {
	tx := test.NewTx().SetNextBlockVersion(&v)
	g.AddTx(tx)
}

func (g *BlockV0Generator) GenerateNext(w module.Wallet) {
	var next blockv0.Block
	var err error
	if len(g.validators) == 0 {
		next, err = NewNextV01a(
			w,
			blockv0.BlockV01aJSON{
				Transactions: g.txs,
			},
			g.last,
		)
		g.executeAndAddReceiptsV0TXs(
			g.txs, g.last.Height()+1, g.last.Timestamp()+defaultDelta,
		)
	} else {
		next, err = NewNextV03(
			w,
			blockv0.BlockV03JSON{
				Transactions: g.txs,
				PrevVotes:    g.lastVotes,
				RepsHash:     g.reps.Hash(),
				ReceiptsHash: g.executeAndAddReceiptsV0TXs(
					g.txs, g.last.Height()+1, g.last.Timestamp()+defaultDelta,
				),
			},
			g.last,
			g,
		)
		votes := make([]*blockv0.BlockVote, len(g.validators))
		for i, v := range g.validators {
			votes[i] = blockv0.NewBlockVote(
				v, next.Height(), 0, next.ID(), next.Timestamp(),
			)
		}
		g.lastVotes = blockv0.NewBlockVoteList(votes...)
	}
	assert.NoError(g.t, err)
	assert.NotNil(g.t, next)
	g.last = next
	g.blocks = append(g.blocks, g.last)
	if g.validatorsInTx != nil {
		g.validators = g.validatorsInTx
		g.reps = g.repsInTx
		g.repsByHash[string(g.reps.Hash())] = g.reps
		g.validatorsInTx = nil
		g.repsInTx = nil
	}
	g.txs = nil
}

const defaultDelta = int64(1000)

func NewNextV01a(
	w module.Wallet,
	jsn blockv0.BlockV01aJSON,
	prev blockv0.Block,
) (blockv0.Block, error) {
	if len(jsn.Version) == 0 {
		jsn.Version = blockv0.Version01a
	}
	if len(jsn.PrevBlockHash) == 0 {
		jsn.PrevBlockHash = prev.ID()
	}
	trs := make([]module.Transaction, len(jsn.Transactions))
	for i, tx := range jsn.Transactions {
		trs[i] = tx.Transaction
	}
	if len(jsn.MerkleTreeRootHash) == 0 {
		transactionList := transaction.NewTransactionListV1FromSlice(trs)
		jsn.MerkleTreeRootHash = transactionList.Hash()
	}
	if jsn.TimeStamp == 0 {
		jsn.TimeStamp = uint64(prev.Timestamp() + defaultDelta)
	}
	if len(jsn.BlockHash) == 0 {
		jsn.BlockHash = jsn.CalcHash()
	}
	if jsn.Height == 0 {
		jsn.Height = prev.Height() + 1
	}
	jsn.PeerID = w.Address().(*common.Address)
	sigBs, err := w.Sign(jsn.BlockHash)
	if err != nil {
		return nil, err
	}
	if err := jsn.Signature.UnmarshalBinary(sigBs); err != nil {
		return nil, err
	}
	return blockv0.NewBlockV01a(&jsn), nil
}

func NewNextV03(
	w module.Wallet,
	jsn blockv0.BlockV03JSON,
	prev blockv0.Block,
	lc blockv0.Store,
) (blockv0.Block, error) {
	if len(jsn.Version) == 0 {
		jsn.Version = blockv0.Version03
	}
	if len(jsn.PrevHash) == 0 {
		jsn.PrevHash = prev.ID()
	}
	if len(jsn.TransactionsHash) == 0 {
		jsn.TransactionsHash = blockv0.TransactionRootForBlockV03(jsn.Transactions)
	}
	if len(jsn.RepsHash) == 0 {
		if prev.Validators() == nil {
			jsn.RepsHash = nil
		} else {
			jsn.RepsHash = prev.Validators().Hash()
		}
	}
	if len(jsn.NextRepsHash) == 0 {
		if prev.NextValidators() == nil {
			jsn.NextRepsHash = nil
		} else {
			jsn.NextRepsHash = prev.NextValidators().Hash()
		}
	}
	if len(jsn.PrevVotesHash) == 0 {
		jsn.PrevVotesHash = jsn.PrevVotes.Root()
	}
	if len(jsn.TransactionsHash) == 0 {
		jsn.TransactionsHash = blockv0.TransactionRootForBlockV03(jsn.Transactions)
	}
	zeroHexInt64 := common.HexInt64{}
	if jsn.Timestamp == zeroHexInt64 {
		jsn.Timestamp = common.HexInt64{Value: prev.Timestamp() + defaultDelta}
	}
	if jsn.Height == zeroHexInt64 {
		jsn.Height = common.HexInt64{Value: prev.Height() + 1}
	}
	jsn.Leader = *w.Address().(*common.Address)
	if len(jsn.Hash) == 0 {
		jsn.Hash = jsn.CalcHash()
	}
	sigBs, err := w.Sign(jsn.Hash)
	if err != nil {
		return nil, err
	}
	if err := jsn.Signature.UnmarshalBinary(sigBs); err != nil {
		return nil, err
	}
	return blockv0.NewBlockV03(&jsn, lc)
}

func (g *BlockV0Generator) GetBlockByHeight(height int) (blockv0.Block, error) {
	if height < len(g.blocks) {
		return g.blocks[height], nil
	}
	return nil, errors.NotFoundError.Errorf("NotFound")
}

func (g *BlockV0Generator) GetReceipt(id []byte) (module.Receipt, error) {
	r, ok := g.receiptByHash[string(id)]
	if ok {
		return r, nil
	}
	return nil, errors.NotFoundError.Errorf("NotFound")
}

func (g *BlockV0Generator) GetRepsByHash(hash []byte) (*blockv0.RepsList, error) {
	reps, ok := g.repsByHash[string(hash)]
	if ok {
		return reps, nil
	}
	return nil, errors.NotFoundError.Errorf("unknown reps for hash %s", hash)
}

func (g *BlockV0Generator) GetVotesByHeight(h int) (*blockv0.BlockVoteList, error) {
	if h == len(g.blocks)-1 {
		return g.lastVotes, nil
	} else if h < len(g.blocks) {
		return g.blocks[h+1].Votes(), nil
	}
	return nil, errors.NotFoundError.Errorf("NotFound")
}
