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

package blockv0

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

type blockV01aJSON struct {
	Version            string             `json:"version"`
	PrevBlockHash      common.RawHexBytes `json:"prev_block_hash,omitempty"`
	MerkleTreeRootHash common.RawHexBytes `json:"merkle_tree_root_hash"`
	Transactions       []Transaction      `json:"confirmed_transaction_list,omitempty"`
	BlockHash          common.RawHexBytes `json:"block_hash"`
	Height             int64              `json:"height"`
	PeerID             *common.Address    `json:"peer_id"`
	TimeStamp          uint64             `json:"time_stamp"`
	Signature          common.Signature   `json:"signature,omitempty"`
}

type BlockV01a struct {
	*blockV01aJSON
	txs []module.Transaction
}

func (b *BlockV01a) Version() string {
	return b.blockV01aJSON.Version
}

func (b *BlockV01a) ID() []byte {
	return b.blockV01aJSON.BlockHash.Bytes()
}

func (b *BlockV01a) Height() int64 {
	return b.blockV01aJSON.Height
}

func (b *BlockV01a) PrevID() []byte {
	return b.blockV01aJSON.PrevBlockHash.Bytes()
}

func (b *BlockV01a) Votes() *BlockVoteList {
	return nil
}

func (b *BlockV01a) calcHash() []byte {
	bs := make([]byte, 0, 128+8)
	bs = append(bs, []byte(b.PrevBlockHash.String())...)
	bs = append(bs, []byte(b.MerkleTreeRootHash.String())...)
	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, b.TimeStamp)
	bs = append(bs, ts...)
	return crypto.SHA3Sum256(bs)
}

func (b *BlockV01a) Verify(prev Block) error {
	if hash := b.calcHash(); bytes.Compare(hash, b.BlockHash) != 0 {
		return errors.CriticalFormatError.Errorf(
			"IncorrectID(exp=%#x,calc=%#x", b.BlockHash, hash)
	}

	if b.Height() > 0 {
		if prev != nil {
			if pid := prev.ID(); !bytes.Equal(pid, b.PrevBlockHash.Bytes()) {
				return errors.CriticalFormatError.Errorf(
					"InvalidPrevID(exp=%#x,real=%#x)",
					b.PrevBlockHash.Bytes(), pid,
				)
			}
		}

		if pk, err := b.Signature.RecoverPublicKey(b.BlockHash); err == nil {
			addr := common.NewAccountAddressFromPublicKey(pk)
			if !b.PeerID.Equal(addr) {
				return errors.Errorf("InvalidPeerID(peerID=%s,signer=%s)", b.PeerID, addr)
			}
		} else {
			return errors.CriticalFormatError.Wrap(err, "FailureOnRecover")
		}
	} else {
		if prev != nil {
			return errors.CriticalFormatError.New("NonNilPreviousBlock")
		}
	}

	transactionList := transaction.NewTransactionListV1FromSlice(b.txs)
	mrh := transactionList.Hash()
	if bytes.Compare(b.MerkleTreeRootHash, mrh) != 0 {
		return errors.CriticalFormatError.Errorf(
			"InvalidTransactionMerkleRoot(exp=%#x,calc=%#x)",
			b.MerkleTreeRootHash, mrh)
	}
	return nil
}

func (b *BlockV01a) String() string {
	return fmt.Sprint(b.blockV01aJSON)
}

func (b *BlockV01a) NormalTransactions() []module.Transaction {
	return b.txs
}

func (b *BlockV01a) Timestamp() int64 {
	return int64(b.TimeStamp)
}

func (b *BlockV01a) Proposer() module.Address {
	return b.PeerID
}

func (b *BlockV01a) LogsBloom() module.LogsBloom {
	return nil
}

func (b *BlockV01a) Validators() *RepsList {
	return nil
}

func (b *BlockV01a) NextValidators() *RepsList {
	return nil
}

func (b *BlockV01a) ToJSON(version module.JSONVersion) (interface{}, error) {
	return b.blockV01aJSON, nil
}

func ParseBlockV01a(b []byte) (Block, error) {
	var blk = new(blockV01aJSON)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	trs := make([]module.Transaction, len(blk.Transactions))
	for i, tx := range blk.Transactions {
		trs[i] = tx.Transaction
	}
	return &BlockV01a{blk, trs}, nil
}
