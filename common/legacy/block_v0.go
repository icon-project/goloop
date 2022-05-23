package legacy

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type transactionV3 struct {
	module.Transaction
}

func (t *transactionV3) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (t *transactionV3) UnmarshalJSON(b []byte) error {
	if tr, err := transaction.NewTransactionFromJSON(b); err != nil {
		return err
	} else {
		t.Transaction = tr
		return nil
	}
}

func (t transactionV3) String() string {
	return fmt.Sprint(t.Transaction)
}

type blockV0Impl struct {
	module.Block
	Version            string             `json:"version"`
	PrevBlockHash      common.RawHexBytes `json:"prev_block_hash"`
	MerkleTreeRootHash common.RawHexBytes `json:"merkle_tree_root_hash"`
	Transactions       []transactionV3    `json:"confirmed_transaction_list"`
	BlockHash          common.RawHexBytes `json:"block_hash"`
	Height             int64              `json:"height"`
	PeerID             string             `json:"peer_id"`
	TimeStamp          uint64             `json:"time_stamp"`
	Signature          common.Signature   `json:"signature"`
}

type blockV0 struct {
	*blockV0Impl
	transactionList module.TransactionList
}

func (b *blockV0) Version() int {
	return module.BlockVersion0
}

func (b *blockV0) ID() []byte {
	return b.blockV0Impl.BlockHash.Bytes()
}

func (b *blockV0) Height() int64 {
	return b.blockV0Impl.Height
}

func (b *blockV0) PrevRound() int {
	return 0
}

func (b *blockV0) PrevID() []byte {
	return b.blockV0Impl.PrevBlockHash.Bytes()
}

func (b *blockV0) Votes() module.CommitVoteSet {
	return nil
}

func (b *blockV0) NextValidatorsHash() []byte {
	return nil
}

func (b *blockV0) NextValidators() module.ValidatorList {
	return nil
}

func (b *blockV0) Verify() error {
	bs := make([]byte, 0, 128+8)
	bs = append(bs, []byte(b.PrevBlockHash.String())...)
	bs = append(bs, []byte(b.MerkleTreeRootHash.String())...)
	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, b.TimeStamp)
	bs = append(bs, ts...)
	bhash := crypto.SHA3Sum256(bs)

	if bytes.Compare(bhash, b.BlockHash) != 0 {
		log.Warnln("RECORDED  ", b.BlockHash)
		log.Warnln("CALCULATED", hex.EncodeToString(bhash))
		return errors.New("HASH is incorrect")
	}

	if b.Height() > 0 {
		if pk, err := b.Signature.RecoverPublicKey(bhash); err == nil {
			addr := common.NewAccountAddressFromPublicKey(pk).String()
			if addr != b.PeerID {
				log.Warnln("PEERID    ", b.PeerID)
				log.Warnln("SIGNER    ", addr)
				return errors.New("SIGNER is different from PEERID")
			}
		} else {
			log.Println("FAIL to recover address from signature")
			return err
		}
	}

	mrh := b.NormalTransactions().Hash()
	if bytes.Compare(mrh, b.MerkleTreeRootHash) != 0 {
		log.Warnln("MerkleRootHash STORE", hex.EncodeToString(b.MerkleTreeRootHash))
		log.Warnln("MerkleRootHash CALC ", hex.EncodeToString(mrh))
		return errors.New("MerkleTreeRootHash is different")
	}
	return nil
}

func (b *blockV0) String() string {
	return fmt.Sprint(b.blockV0Impl)
}

func (b *blockV0) NormalTransactions() module.TransactionList {
	return b.transactionList
}

func (b *blockV0) PatchTransactions() module.TransactionList {
	return nil
}

func (b *blockV0) Timestamp() int64 {
	return int64(b.TimeStamp)
}

func (b *blockV0) Proposer() module.Address {
	return nil
}

func (b *blockV0) LogsBloom() module.LogsBloom {
	return nil
}

func (b *blockV0) Result() []byte {
	return nil
}

func (b *blockV0) NormalReceipts() module.ReceiptList {
	return nil
}

func (b *blockV0) PatchReceipts() module.ReceiptList {
	return nil
}

func (b *blockV0) MarshalHeader(w io.Writer) error {
	return nil
}

func (b *blockV0) MarshalBody(w io.Writer) error {
	return nil
}

func (b *blockV0) Marshal(w io.Writer) error {
	return nil
}

func (b *blockV0) ToJSON(version module.JSONVersion) (interface{}, error) {
	return nil, nil
}

func (b *blockV0) NewBlock(tr module.Transition) module.Block {
	return nil
}

func (b *blockV0) Hash() []byte {
	return nil
}

type Block interface {
	module.Block
	Verify() error
}

func ParseBlockV0(b []byte) (Block, error) {
	var blk = new(blockV0Impl)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	trs := make([]module.Transaction, len(blk.Transactions))
	for i, tx := range blk.Transactions {
		trs[i] = tx.Transaction
	}
	transactionList := transaction.NewTransactionListV1FromSlice(trs)
	return &blockV0{blk, transactionList}, nil
}
