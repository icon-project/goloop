package legacy

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

type LoopChainDB struct {
	blockbk, scorebk *leveldb.DB
}

func (lc *LoopChainDB) GetBlockJSONByHeight(height int) ([]byte, error) {
	prefix := "block_height_key"
	key := make([]byte, len(prefix)+12)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix)+4:], uint64(height))
	bid, err := lc.blockbk.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	blockjson, err := lc.blockbk.Get(bid, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return blockjson, nil
}

func (lc *LoopChainDB) GetLastBlockJSON() ([]byte, error) {
	bid, err := lc.blockbk.Get([]byte("last_block_key"), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	blockjson, err := lc.blockbk.Get(bid, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return blockjson, nil
}

func (lc *LoopChainDB) GetBlockByHeight(height int) (module.Block, error) {
	if bs, err := lc.GetBlockJSONByHeight(height); err != nil {
		return nil, err
	} else {
		b, err := ParseBlockV0(bs)
		if err != nil {
			log.Warnf("Fail to parse block err=%+v blocks=%s", err, string(bs))
		}
		return b, err
	}
}

func (lc *LoopChainDB) GetLastBlock() (module.Block, error) {
	if bs, err := lc.GetLastBlockJSON(); err != nil {
		return nil, err
	} else {
		b, err := ParseBlockV0(bs)
		if err != nil {
			log.Warnf("Fail to parse block err=%+v blocks=%s", err, string(bs))
		}
		return b, err
	}
}

type TransactionInfo struct {
	BlockID     common.HexBytes `json:"block_hash"`
	BlockHeight int             `json:"block_height"`
	TxIndex     common.HexInt32 `json:"tx_index"`
	Transaction transactionV3   `json:"transaction"`
	Receipt     json.RawMessage `json:"result"`
}

func (lc *LoopChainDB) GetTransactionInfoJSONByTransaction(id []byte) ([]byte, error) {
	key := []byte(hex.EncodeToString(id))
	tinfo, err := lc.blockbk.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return tinfo, nil
}

func (lc *LoopChainDB) GetTransactionInfoByTransaction(id []byte) (*TransactionInfo, error) {
	bs, err := lc.GetTransactionInfoJSONByTransaction(id)
	if err != nil {
		return nil, err
	}
	tinfo := new(TransactionInfo)
	if err := json.Unmarshal(bs, tinfo); err != nil {
		return nil, err
	}
	return tinfo, nil
}

func (lc *LoopChainDB) GetReceiptByTransaction(id []byte) (module.Receipt, error) {
	if tinfo, err := lc.GetTransactionInfoByTransaction(id); err != nil {
		return nil, err
	} else {
		if r, err := txresult.NewReceiptFromJSON(nil, module.NoRevision, tinfo.Receipt); err != nil {
			return nil, err
		} else {
			return r, nil
		}
	}
}

const (
	accountGeneral  = 0
	accountGenesis  = 1
	accountTreasury = 2
	accountContract = 3
)

type accountV1 struct {
	lc                 *LoopChainDB
	accountType, flags byte
	balance            common.HexInt
}

func (ac *accountV1) Bytes() []byte {
	panic("implement me")
}

func (ac *accountV1) Reset(s db.Database, k []byte) error {
	if len(k) < 1 {
		return errors.New("LengthIsShort")
	}
	if len(k) < 36 {
		return errors.Errorf("LengthIsIncorrect len=%d data=% x", len(k), k)
	}
	version := k[0]
	switch version {
	case 0:
		ac.accountType = k[1]
		ac.flags = k[2]
		ac.balance.SetBytes(k[4 : 4+32])
		return nil
	default:
		return errors.New("UnknownVersion")
	}
}

func (ac *accountV1) Flush() error {
	return nil
}

func (ac *accountV1) Equal(to trie.Object) bool {
	return false
}

func (ac *accountV1) GetBalance() *big.Int {
	b := new(big.Int)
	b.Set(&ac.balance.Int)
	return b
}

func (ac *accountV1) IsContract() bool {
	return ac.accountType == accountContract
}

func (ac *accountV1) Empty() bool {
	return false
}

func (ac *accountV1) GetValue(k []byte) ([]byte, error) {
	panic("implement me")
}

func NewAccountV1(lc *LoopChainDB, bs []byte) (*accountV1, error) {
	ac := new(accountV1)
	if err := ac.Reset(nil, bs); err != nil {
		return nil, err
	}
	ac.lc = lc
	return ac, nil
}

func (lc *LoopChainDB) GetAccount(addr module.Address) (*accountV1, error) {
	var key []byte
	if addr.IsContract() {
		key = addr.Bytes()
	} else {
		key = addr.ID()
	}
	value, err := lc.scorebk.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return NewAccountV1(lc, value)
}

func (lc *LoopChainDB) Close() error {
	if err := lc.blockbk.Close(); err != nil {
		return err
	}
	if lc.scorebk != nil {
		if err := lc.scorebk.Close(); err != nil {
			return err
		}
	}
	return nil
}

type accountV1Iterator struct {
	lc *LoopChainDB
	iterator.Iterator
}

func (aitr *accountV1Iterator) Value() (module.Address, *accountV1, error) {
	if aitr.Iterator.Valid() {
		key := aitr.Iterator.Key()
		var addr module.Address
		if len(key) == common.AddressBytes {
			if ptr, err := common.NewAddress(key); err != nil {
				return nil, nil, err
			} else {
				addr = ptr
			}
		} else if len(key) == common.AddressIDBytes {
			addr = common.NewAccountAddress(key)
		} else {
			return nil, nil, nil
		}
		ac, err := NewAccountV1(aitr.lc, aitr.Iterator.Value())
		if err != nil {
			return nil, nil, err
		}
		return addr, ac, nil
	}
	return nil, nil, common.ErrNotFound
}

func (aitr *accountV1Iterator) Next() bool {
	for true {
		valid := aitr.Iterator.Next()
		if valid {
			kb := aitr.Iterator.Key()
			if len(kb) == common.AddressIDBytes {
				return true
			}
			if len(kb) == common.AddressBytes && kb[0] == 1 {
				return true
			}
		} else {
			break
		}
	}
	return false
}

func (lc *LoopChainDB) AccountIterator() *accountV1Iterator {
	rg := &util.Range{
		Start: make([]byte, 20),
		Limit: bytes.Repeat([]byte{0xff}, 21),
	}
	return &accountV1Iterator{lc, lc.scorebk.NewIterator(rg, nil)}
}

func OpenDatabase(blockdir, scoredir string) (*LoopChainDB, error) {
	lcdb := new(LoopChainDB)
	opt := &opt.Options{
		ReadOnly: true,
	}
	if blockbk, err := leveldb.RecoverFile(blockdir, opt); err != nil {
		return nil, err
	} else {
		lcdb.blockbk = blockbk
	}
	if scoredir != "" {
		if scorebk, err := leveldb.RecoverFile(scoredir, opt); err != nil {
			log.Warnf("Fail to open SCORE DB err=%+v (ignore)", err)
		} else {
			lcdb.scorebk = scorebk
		}
	}
	return lcdb, nil
}
