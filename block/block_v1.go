package block

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

type transactionV3 struct {
	module.Transaction
}

func (t *transactionV3) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (t *transactionV3) UnmarshalJSON(b []byte) error {
	if tr, err := service.NewTransactionV3(b); err != nil {
		return err
	} else {
		t.Transaction = tr
		return nil
	}
}

func (t transactionV3) String() string {
	return fmt.Sprint(t.Transaction)
}

type blockV1 struct {
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

type BlockV1 struct {
	*blockV1
}

func (b *BlockV1) ID() []byte {
	return b.blockV1.BlockHash.ToBytes()
}

func (b *BlockV1) Height() int64 {
	return b.blockV1.Height
}

func (b *BlockV1) PrevRound() int {
	return 0
}

func (b *BlockV1) PrevID() []byte {
	return b.blockV1.PrevBlockHash.ToBytes()
}

func (b *BlockV1) Votes() []module.Vote {
	return nil
}

func (b *BlockV1) NextValidators() []module.Address {
	return nil
}

func (b *BlockV1) Verify() error {
	return b.blockV1.Verify()
}

func (b *BlockV1) String() string {
	return fmt.Sprint(b.blockV1)
}

func NewBlockV1(b []byte) (module.Block, error) {
	var blk = new(blockV1)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	return &BlockV1{blk}, nil
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

func (blk *blockV1) Verify() error {
	b := make([]byte, 128+8)
	copy(b[0:], []byte(blk.PrevBlockHash.String()))
	copy(b[64:], []byte(blk.MerkleTreeRootHash.String()))
	binary.LittleEndian.PutUint64(b[128:], blk.TimeStamp)
	bhash := crypto.SHA3Sum256(b)

	var txs = map[int]int{}
	for _, t := range blk.Transactions {
		txs[t.Version()] += 1
	}
	fmt.Printf("<> BLOCK %8d %s tx=%v\n",
		blk.Height, hex.EncodeToString(blk.BlockHash), txs)

	if bytes.Compare(bhash, blk.BlockHash) != 0 {
		log.Println("RECORDED  ", blk.BlockHash)
		log.Println("CALCULATED", hex.EncodeToString(bhash))
		return errors.New("HASH is incorrect")
	}

	addr, err := blk.Signature.RecoverAddressWithHash(bhash)
	if err != nil {
		log.Println("FAIL to recover address from signature")
		return err
	}

	if addr != blk.PeerID {
		log.Println("PEERID    ", blk.PeerID)
		log.Println("SIGNER    ", addr)
		return errors.New("SIGNER is different from PEERID")
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
