package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

type blockV1Impl struct {
	Version            string             `json:"version"`
	PrevBlockHash      common.RawHexBytes `json:"prev_block_hash"`
	MerkleTreeRootHash common.RawHexBytes `json:"merkle_tree_root_hash"`
	Transactions       []transaction      `json:"confirmed_transaction_list"`
	BlockHash          common.RawHexBytes `json:"block_hash"`
	Height             int64              `json:"height"`
	PeerID             string             `json:"peer_id"`
	TimeStamp          uint64             `json:"time_stamp"`
	Signature          common.Signature   `json:"signature"`
}

type blockV1 struct {
	*blockV1Impl
}

func (b *blockV1) Version() int {
	return 1
}

func (b *blockV1) ID() []byte {
	return b.blockV1Impl.BlockHash.ToBytes()
}

func (b *blockV1) Height() int64 {
	return b.blockV1Impl.Height
}

func (b *blockV1) PrevRound() int {
	return 0
}

func (b *blockV1) PrevID() []byte {
	return b.blockV1Impl.PrevBlockHash.ToBytes()
}

func (b *blockV1) Votes() module.VoteList {
	return nil
}

func (b *blockV1) NextValidators() module.ValidatorList {
	return nil
}

func (b *blockV1) Verify() error {
	return b.blockV1Impl.Verify()
}

func (b *blockV1) String() string {
	return fmt.Sprint(b.blockV1Impl)
}

func (b *blockV1) NormalTransactions() module.TransactionList {
	trs := make([]module.Transaction, len(b.blockV1Impl.Transactions))
	for i, tx := range b.blockV1Impl.Transactions {
		trs[i] = tx
	}
	return service.NewTransactionListFromSlice(trs)
}

func (b *blockV1) PatchTransactions() module.TransactionList {
	return nil
}

func (b *blockV1) Timestamp() time.Time {
	return time.Time{}
}

func (b *blockV1) Proposer() module.Address {
	return nil
}

func (b *blockV1) LogBloom() []byte {
	return nil
}

func (b *blockV1) Result() []byte {
	return nil
}

func (b *blockV1) NormalReceipts() module.ReceiptList {
	return nil
}

func (b *blockV1) PatchReceipts() module.ReceiptList {
	return nil
}

func (b *blockV1) MarshalHeader(w io.Writer) error {
	return nil
}

func (b *blockV1) MarshalBody(w io.Writer) error {
	return nil
}

func NewBlockV1(b []byte) (module.Block, error) {
	var blk = new(blockV1Impl)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	return &blockV1{blk}, nil
}

func calcMergedHash(h1, h2 []byte) []byte {
	var b [128]byte
	copy(b[0:], []byte(hex.EncodeToString(h1)))
	copy(b[64:], []byte(hex.EncodeToString(h2)))
	ts := crypto.SHASum256(b[:])
	return ts[:]
}

func calcMerkleTreeRoot(m [][]byte) []byte {
	if len(m) == 0 {
		var empty [32]byte
		return empty[:]
	}
	ml := make([][]byte, len(m))
	copy(ml, m)
	for mlen := len(ml); mlen > 1; mlen = (mlen + 1) / 2 {
		for i := 0; i < mlen; i += 2 {
			if i+1 < mlen {
				ml[i/2] = calcMergedHash(ml[i], ml[i+1])
			} else {
				ml[i/2] = calcMergedHash(ml[i], ml[i])
			}
		}
	}
	return ml[0]
}

func (blk *blockV1Impl) Verify() error {
	b := make([]byte, 128+8)
	copy(b[0:], []byte(blk.PrevBlockHash.String()))
	copy(b[64:], []byte(blk.MerkleTreeRootHash.String()))
	binary.LittleEndian.PutUint64(b[128:], blk.TimeStamp)
	bhash := crypto.SHA3Sum256(b)

	if bytes.Compare(bhash, blk.BlockHash) != 0 {
		log.Println("RECORDED  ", blk.BlockHash)
		log.Println("CALCULATED", hex.EncodeToString(bhash))
		return errors.New("HASH is incorrect")
	}

	if pk, err := blk.Signature.RecoverPublicKey(bhash); err == nil {
		addr := common.NewAccountAddressFromPublicKey(pk).String()
		if addr != blk.PeerID {
			log.Println("PEERID    ", blk.PeerID)
			log.Println("SIGNER    ", addr)
			return errors.New("SIGNER is different from PEERID")
		}
	} else {
		log.Println("FAIL to recover address from signature")
		return err
	}

	merkle := make([][]byte, len(blk.Transactions))
	for i, t := range blk.Transactions {
		if err := t.Verify(); err != nil {
			log.Printf("Transaction[%d] Verification fails", i)
			return err
		}
		merkle[i] = t.ID()
	}
	mrh := calcMerkleTreeRoot(merkle)
	if bytes.Compare(mrh, blk.MerkleTreeRootHash) != 0 {
		log.Println("MerkleRootHash STORE", hex.EncodeToString(blk.MerkleTreeRootHash))
		log.Println("MerkleRootHash CALC ", hex.EncodeToString(mrh))
		return errors.New("MerkleTreeRootHash is different")
	}
	return nil
}
