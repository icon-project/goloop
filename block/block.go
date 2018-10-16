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
	"github.com/icon-project/goloop/service"
)

type Transaction struct {
	service.Transaction
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (t *Transaction) UnmarshalJSON(b []byte) error {
	if tr, err := service.NewTransaction(b); err != nil {
		return err
	} else {
		t.Transaction = tr
		return nil
	}
}

func (t Transaction) String() string {
	return fmt.Sprintf("%+v", t.Transaction)
}

type Block struct {
	Version            string             `json:"version"`
	PrevBlockHash      common.RawHexBytes `json:"prev_block_hash"`
	MerkleTreeRootHash common.RawHexBytes `json:"merkle_tree_root_hash"`
	Transactions       []Transaction      `json:"confirmed_transaction_list"`
	BlockHash          common.RawHexBytes `json:"block_hash"`
	Height             int                `json:"height"`
	PeerID             string             `json:"peer_id"`
	TimeStamp          uint64             `json:"time_stamp"`
	Signature          common.Signature   `json:"signature"`
}

func NewBlock(b []byte) (*Block, error) {
	var blk = new(Block)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
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

func (blk *Block) Verify() error {
	b := make([]byte, 128+8)
	copy(b[0:], []byte(blk.PrevBlockHash.String()))
	copy(b[64:], []byte(blk.MerkleTreeRootHash.String()))
	binary.LittleEndian.PutUint64(b[128:], blk.TimeStamp)
	bhash := crypto.SHA3Sum256(b)

	var txs = map[int]int{}
	for _, t := range blk.Transactions {
		txs[t.GetVersion()] += 1
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
		merkle[i] = t.GetHash()
	}
	mrh := calcMerkleTreeRoot(merkle)
	if bytes.Compare(mrh, blk.MerkleTreeRootHash) != 0 {
		log.Println("MerkleRootHash STORE", hex.EncodeToString(blk.MerkleTreeRootHash))
		log.Println("MerkleRootHash CALC ", hex.EncodeToString(mrh))
		return errors.New("MerkleTreeRootHash is different")
	}
	return nil
}
