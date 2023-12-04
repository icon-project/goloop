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

package lcimporter_test

import (
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/ictest"
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type BTX = lcimporter.BlockTransaction

type testChain struct {
	module.Chain
	dbase     db.Database
	wallet    module.Wallet
	log       log.Logger
	regulator module.Regulator
}

func (c *testChain) Database() db.Database {
	return c.dbase
}

func (c *testChain) Wallet() module.Wallet {
	return c.wallet
}

func (c *testChain) NID() int {
	return 1
}

func (c *testChain) CID() int {
	return 1
}

func (c *testChain) NetID() int {
	return 1
}

func (c *testChain) Channel() string {
	return "icon"
}

func (c *testChain) ConcurrencyLevel() int {
	return 1
}

func (c *testChain) NormalTxPoolSize() int {
	return 5000
}

func (c *testChain) PatchTxPoolSize() int {
	return 2
}

func (c *testChain) MaxBlockTxBytes() int {
	return 2 * 1024 * 1024
}

func (c *testChain) TransactionTimeout() time.Duration {
	return time.Second * 5
}

func (c *testChain) Regulator() module.Regulator {
	return c.regulator
}

func (c *testChain) Logger() log.Logger {
	return c.log
}

func (c *testChain) NetworkManager() module.NetworkManager {
	return nil
}

func newTestChain(database db.Database, logger log.Logger) (*testChain, error) {
	w := wallet.New()
	return &testChain{
		dbase:     database,
		wallet:    w,
		log:       logger,
		regulator: lcimporter.NewRegulator(),
	}, nil
}

type testService struct {
	lcimporter.BasicService
	chn chan<- *TransitionRequest
}

func newTestService(c module.Chain, plt base.Platform, baseDir string) *testService {
	return &testService{
		BasicService: lcimporter.BasicService{
			Chain:   c,
			Plt:     plt,
			BaseDir: baseDir,
		},
	}
}

func (s *testService) NewSyncTransition(tr module.Transition, result []byte, vl []byte) module.Transition {
	panic("implement me")
}

type TransitionRequest struct {
	parent           module.Transition
	patchtxs         module.TransactionList
	normaltxs        module.TransactionList
	bi               module.BlockInfo
	csi              module.ConsensusInfo
	alreadyValidated bool
}

func (s *testService) ForwardTransitionRequest(chn chan<-*TransitionRequest) {
	s.chn = chn
}

func (s *testService) NewTransition(
	parent module.Transition, patchtxs module.TransactionList,
	normaltxs module.TransactionList, bi module.BlockInfo,
	csi module.ConsensusInfo, alreadyValidated bool,
) module.Transition {
	if s.chn != nil {
		s.chn <- &TransitionRequest{
			parent, patchtxs, normaltxs, bi,
			csi, alreadyValidated,
		}
	}
	return s.BasicService.NewTransition(
		parent, patchtxs, normaltxs, bi, csi, alreadyValidated,
	)
}

func newTestStore(dbase db.Database) (*testStore, error) {
	s := &testStore{
		blocks:   make(map[int]blockv0.Block),
		reps:     make(map[string]*blockv0.RepsList),
		receipts: make(map[string]module.Receipt),
	}
	for h, b := range blocks {
		blk, err := blockv0.ParseBlock([]byte(b), s)
		if err != nil {
			return nil, err
		}
		s.blocks[h] = blk
	}
	for idStr, r := range receipts {
		receipt, err := txresult.NewReceiptFromJSON(dbase, module.LatestRevision, []byte(r))
		if err != nil {
			return nil, err
		}
		id, err := hex.DecodeString(idStr)
		if err != nil {
			return nil, err
		}
		s.receipts[string(id)] = receipt
	}
	return s, nil
}

type testStore struct {
	blocks   map[int]blockv0.Block
	reps     map[string]*blockv0.RepsList
	receipts map[string]module.Receipt
	dbase    db.Database
}

var blocks = map[int]string{
	0: `
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
`,
	1: `
{
  "version": "0.1a",
  "prev_block_hash": "cf43b3fd45981431a0e64f79d07bfcf703e064b73b802c5f32834eec72142190",
  "merkle_tree_root_hash": "375540830d475a73b704cf8dee9fa9eba2798f9d2af1fa55a85482e48daefd3b",
  "time_stamp": 1516819217223222,
  "confirmed_transaction_list": [
    {
      "from": "hx54f7853dc6481b670caf69c5a27c7c8fe5be8269",
      "to": "hx49a23bd156932485471f582897bf1bec5f875751",
      "value": "0x56bc75e2d63100000",
      "fee": "0x2386f26fc10000",
      "nonce": "0x1",
      "tx_hash": "375540830d475a73b704cf8dee9fa9eba2798f9d2af1fa55a85482e48daefd3b",
      "signature": "bjarKeF3izGy469dpSciP3TT9caBQVYgHdaNgjY+8wJTOVSFm4o/ODXycFOdXUJcIwqvcE9If8x6Zmgt//XmkQE=",
      "method": "icx_sendTransaction"
    }
  ],
  "block_hash": "3add53134014e940f6f6010173781c4d8bd677d9931a697f962483e04a685e5c",
  "height": 1,
  "peer_id": "hx7e1a1ece096ef3fa44ac9692394c2e11d0017e4a",
  "signature": "liAIa7aPYvBRdZAdBz6zt2Gc9vVo/4+gkDz5uscS8Mw+B5gkp6zQeHhD5sNpyWcIsq5c9OxwOCUaBp0vu8eAgwE=",
  "next_leader": ""
}
`,
}

var receipts = map[string]string{
	"5aa2453a84ba2fb1e3394b9e3471f5dcebc6225fc311a97ca505728153b9d246": `
{
  "cumulativeStepUsed": "0x0",
  "stepUsed": "0x0",
  "stepPrice": "0x0",
  "status": "0x1"
}`,
	"375540830d475a73b704cf8dee9fa9eba2798f9d2af1fa55a85482e48daefd3b": `
{
    "blockHash": "0x3add53134014e940f6f6010173781c4d8bd677d9931a697f962483e04a685e5c",
    "blockHeight": "0x1",
    "cumulativeStepUsed": "0xf4240",
    "eventLogs": [],
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "status": "0x1",
    "stepPrice": "0x2540be400",
    "stepUsed": "0xf4240",
    "to": "hx49a23bd156932485471f582897bf1bec5f875751",
    "txHash": "0x375540830d475a73b704cf8dee9fa9eba2798f9d2af1fa55a85482e48daefd3b",
    "txIndex": "0x0"
}
`,
}

func (s *testStore) GetRepsByHash(id []byte) (*blockv0.RepsList, error) {
	reps, ok := s.reps[string(id)]
	if !ok {
		return nil, errors.NotFoundError.New("reps not found")
	}
	return reps, nil
}

func (s *testStore) GetBlockByHeight(height int) (blockv0.Block, error) {
	blk, ok := s.blocks[height]
	if !ok {
		return nil, errors.NotFoundError.New("block not found")
	}
	return blk, nil
}

func (s *testStore) GetReceipt(id []byte) (module.Receipt, error) {
	receipt, ok := s.receipts[string(id)]
	if !ok {
		return nil, errors.NotFoundError.New("receipts not found")
	}
	return receipt, nil
}

func (s *testStore) GetVotesByHeight(h int) (*blockv0.BlockVoteList, error) {
	return nil, errors.NotFoundError.New("votes not found")
}

type blockConverterTest struct {
	*testing.T
	*lcimporter.BlockConverter
	chain       *testChain
	store       lcimporter.Store
	svc         *testService
	emptyResult []byte
}

type transitionCallback chan error

func (cb transitionCallback) OnValidate(transition module.Transition, err error) {
	cb <- err
}

func (cb transitionCallback) OnExecute(transition module.Transition, err error) {
	cb <- err
}

func newBlockConverterTest(t *testing.T) *blockConverterTest {
	return newBlockConverterTestWithDB(t, db.NewMapDB())
}
func newBlockConverterTestWithDB(t *testing.T, dbase db.Database) *blockConverterTest {
	s, err := newTestStore(dbase)
	assert.NoError(t, err)
	plt := ictest.NewPlatform()
	return newBlockConverterTest2(t, dbase, s, plt)
}

func newBlockConverterTest2(t *testing.T, dbase db.Database, s lcimporter.Store, plt base.Platform) *blockConverterTest {
	base, err := os.MkdirTemp("", "goloop-blockconverter-test")
	c, err := newTestChain(dbase, log.New())
	assert.NoError(t, err)
	assert.NoError(t, err)
	svc := newTestService(c, plt, base)

	itr, err := svc.NewInitTransition(nil, nil, c.Logger())
	assert.NoError(t, err)
	tr := svc.NewTransition(
		itr,
		transaction.NewTransactionListFromSlice(c.Database(), nil),
		transaction.NewTransactionListFromSlice(c.Database(), nil),
		common.NewBlockInfo(0, 0),
		common.NewConsensusInfo(nil, nil, nil),
		true,
	)
	cb := make(transitionCallback, 1)
	_, err = tr.Execute(cb)
	assert.NoError(t, err)
	err = <-cb
	assert.Nil(t, err)
	err = <-cb
	assert.Nil(t, err)
	emptyResult := tr.Result()

	bc, err := lcimporter.NewBlockConverterWithService(c, plt, s, base, svc)
	assert.NoError(t, err)
	return &blockConverterTest{
		T:              t,
		BlockConverter: bc,
		chain:          c,
		store:          s,
		svc:            svc,
		emptyResult:    emptyResult,
	}
}

func assertBlockTransaction(t assert.TestingT, res interface{}, height int, txCount int, f func(r *BTX)) {
	switch r := res.(type) {
	case *lcimporter.BlockTransaction:
		assert.EqualValues(t, height, r.Height)
		assert.EqualValues(t, txCount, r.TXCount)
		if f!= nil {
			f(r)
		}
	case error:
		assert.NoError(t, r)
	default:
		assert.Fail(t, "Unknown result type", "%+v", res)
	}
}

func TestBlockConverter_Genesis(t_ *testing.T) {
	t := newBlockConverterTest(t_)
	ch, err := t.Start(0, 0)
	assert.NoError(t, err)
	for res := range ch {
		assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
			assert.Equal(t, t.emptyResult, r.Result)
			assert.Nil(t, r.ValidatorHash)
		})
	}
}

func TestBlockConverter_Continue(t_ *testing.T) {
	t := newBlockConverterTest(t_)
	ch, err := t.Start(0, 1)
	assert.NoError(t, err)
	res := <-ch
	assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
		assert.Equal(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	ch, err = t.Rebase(0, 1, nil)
	assert.NoError(t, err)
	res = <-ch
	assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
		assert.Equal(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
}

func TestBlockConverter_Continue2(t_ *testing.T) {
	t := newBlockConverterTest(t_)
	ch, err := t.Start(0, 1)
	assert.Nil(t, err)
	res := <-ch
	res = <-ch
	var eResult []byte
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
		eResult = r.Result
	})

	t = newBlockConverterTest(t_)
	ch, err = t.Start(0, 0)
	assert.NoError(t, err)
	res = <-ch
	t = newBlockConverterTestWithDB(t_, t.chain.Database())
	ch, err = t.Start(1, 1)
	assert.NoError(t, err)
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.Equal(t, eResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
}

func TestBlockConverter_Continue3(t_ *testing.T) {
	t := newBlockConverterTest(t_)
	ch, err := t.Start(0, 1)
	assert.NoError(t, err)
	res := <-ch
	res = <-ch
	var blockHash []byte
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		blockHash = r.BlockHash
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	t = newBlockConverterTestWithDB(t_, t.chain.Database())
	ch, err = t.Start(0, 1)
	assert.NoError(t, err)
	res = <-ch
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.Equal(t, blockHash, r.BlockHash)
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
}

func TestBlockConverter_Term(t_ *testing.T) {
	t := newBlockConverterTest(t_)
	ch, err := t.Start(0, 1)
	assert.NoError(t, err)
	res := <-ch
	assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
		assert.Equal(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	t.Term()
	ch, err = t.Start(0, 1)
	assert.NoError(t, err)
	res = <-ch
	assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
		assert.Equal(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
}

func TestBlockConverter_CloseOnVersionChange(_t *testing.T) {
	bg := ictest.NewBlockV0Generator(_t, "")
	defer bg.Close()

	bg.AddSetRandomValidatorsTx(4)
	w := bg.ValidatorsInTx()[0]
	bg.GenerateNext(w)
	bg.AddSetNextBlockVersionTx(module.BlockVersion2)
	bg.GenerateNext(w)
	bg.GenerateNext(w)

	t := newBlockConverterTest2(_t, db.NewMapDB(), bg, ictest.NewPlatform())
	ch, err := t.Start(0, -1)
	assert.NoError(t, err)
	res := <-ch
	assertBlockTransaction(t, res, 0, 1, func(r *BTX) {
		assert.NotNil(t, r.Result)
		assert.Equal(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, func(r *BTX) {
		assert.NotNil(t, r.Result)
		assert.NotEqual(t, t.emptyResult, r.Result)
		assert.Nil(t, r.ValidatorHash)
	})
	res = <-ch
	assertBlockTransaction(t, res, 2, 1, func(r *BTX) {
		assert.NotNil(t, r.ValidatorHash)
	})
	res = <-ch
	assertBlockTransaction(t, res, 3, 0, func(r *BTX) {
		assert.NotNil(t, r.ValidatorHash)
	})
	res, ok := <-ch
	assert.False(_t, ok)
}

func TestBlockConverter_BlockInfoAndConsensusInfo(_t *testing.T) {
	bg := ictest.NewBlockV0Generator(_t, "")
	defer bg.Close()

	bg.AddSetRandomValidatorsTx(4)
	w := bg.ValidatorsInTx()[0]
	bg.GenerateNext(w)
	bg.GenerateNext(w)
	bg.AddSetNextBlockVersionTx(module.BlockVersion2)
	bg.GenerateNext(w)
	bg.GenerateNext(w)

	getBlock := func (h int) blockv0.Block {
		blk, err := bg.GetBlockByHeight(h)
		assert.NoError(_t, err)
		return blk
	}

	mdb := db.NewMapDB()
	getTXLOfBlock := func (h int) module.TransactionList {
		blk, err := bg.GetBlockByHeight(h)
		assert.NoError(_t, err)
		txs := blk.NormalTransactions()
		txl := transaction.NewTransactionListFromSlice(mdb, txs)
		return txl
	}

	t := newBlockConverterTest2(_t, db.NewMapDB(), bg, ictest.NewPlatform())
	trChn := make(chan *TransitionRequest)
	t.svc.ForwardTransitionRequest(trChn)
	ch, err := t.Start(0, -1)
	assert.NoError(t, err)

	tr := <-trChn
	assert.Nil(t, tr.normaltxs.Hash())
	assert.EqualValues(t, 0, tr.bi.Height())
	assert.Nil(t, tr.csi.Proposer())
	res := <-ch
	assertBlockTransaction(t, res, 0, 1, nil)

	tr = <-trChn
	assert.EqualValues(t, 0, tr.bi.Height())
	assert.Equal(t, getTXLOfBlock(0).Hash(), tr.normaltxs.Hash())
	assert.Nil(t, tr.csi.Proposer())
	res = <-ch
	assertBlockTransaction(t, res, 1, 1, nil)

	tr = <-trChn
	assert.EqualValues(t, 1, tr.bi.Height())
	assert.Equal(t, getTXLOfBlock(1).Hash(), tr.normaltxs.Hash())
	assert.Equal(t, getBlock(0).Proposer().Bytes(), tr.csi.Proposer().Bytes())
	res = <-ch
	assertBlockTransaction(t, res, 2, 0, nil)

	tr = <-trChn
	assert.EqualValues(t, 2, tr.bi.Height())
	assert.Equal(t, getTXLOfBlock(2).Hash(), tr.normaltxs.Hash())
	assert.Equal(t, getBlock(1).Proposer().Bytes(), tr.csi.Proposer().Bytes())
	assert.EqualValues(t, 0, len(tr.csi.Voted()))
	res = <-ch
	assertBlockTransaction(t, res, 3, 1, nil)

	tr = <-trChn
	assert.EqualValues(t, 3, tr.bi.Height())
	assert.Equal(t, getTXLOfBlock(3).Hash(), tr.normaltxs.Hash())
	assert.Equal(t, getBlock(2).Proposer().Bytes(), tr.csi.Proposer().Bytes())
	assert.Equal(t, []bool{true, true, true, true}, tr.csi.Voted())
	res = <-ch
	assertBlockTransaction(t, res, 4, 0, nil)

	res, ok := <-ch
	assert.False(_t, ok)
}
