package service

import (
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type mockTransaction struct {
	NID       int
	id        []byte
	from      module.Address
	timeStamp int64
}

func (*mockTransaction) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (t *mockTransaction) ID() []byte {
	return t.id
}

func (t *mockTransaction) From() module.Address {
	return t.from
}

func (t *mockTransaction) Bytes() []byte {
	return t.id
}

func (*mockTransaction) Hash() []byte {
	panic("implement me")
}

func (*mockTransaction) Verify() error {
	panic("implement me")
}

func (*mockTransaction) Version() int {
	panic("implement me")
}

func (*mockTransaction) ToJSON(version module.JSONVersion) (interface{}, error) {
	panic("implement me")
}

func (*mockTransaction) PreValidate(wc state.WorldContext, update bool) error {
	panic("implement me")
}

func (*mockTransaction) GetHandler(cm contract.ContractManager) (transaction.Handler, error) {
	panic("implement me")
}

func (t *mockTransaction) Timestamp() int64 {
	return t.timeStamp
}

func (*mockTransaction) Nonce() *big.Int {
	panic("implement me")
}

func (t *mockTransaction) To() module.Address {
	panic("implement me")
}

func (t *mockTransaction) ValidateNetwork(nid int) bool {
	return t.NID == nid
}

func (t *mockTransaction) IsSkippable() bool {
	return true
}

func newMockTransaction(id []byte, from module.Address, ts int64) *mockTransaction {
	return &mockTransaction{
		id:        id,
		from:      from,
		timeStamp: ts,
	}
}

func TestTransactionList_AddRemove(t *testing.T) {
	l := newTransactionList()
	tx := newMockTransaction([]byte{0x01, 0x02, 0x03, 0x04}, common.MustNewAddressFromString("hx1111111111111111111111111111111111111111"), 0)

	if err := l.Add(tx, false); err != nil {
		t.Error("Fail to add transaction")
		return
	}

	if err := l.Add(tx, false); err != ErrDuplicateTransaction {
		t.Errorf("It should return ErrDuplicateTransaction err=%+v", err)
	}

	e := l.Front()
	if e.value != tx {
		t.Error("First one must be at front")
	}

	tx2 := newMockTransaction([]byte{0x01, 0x02, 0x03, 0x05}, common.MustNewAddressFromString("hx1111111111111111111111111111111111111112"), 0)
	if err := l.Add(tx2, false); err != nil {
		t.Error("Fail to add second transaction")
		return
	}

	e = l.Front()
	if e.value != tx {
		t.Error("First one must be the first after adding another")
	}

	if !l.Remove(e) {
		t.Error("It fails to remove first one")
	}

	e = l.Front()
	if e.value != tx2 {
		t.Error("After removing first, it should return second")
	}
	if !l.Remove(e) {
		t.Error("It fails to remove second one")
	}

	if len := l.Len(); len != 0 {
		t.Errorf("After removing all, it still has non-zero length len=%d", len)
	}

	if err := l.Add(tx, false); err != nil {
		t.Errorf("It fails to add first transaction after removal")
	}
	if err := l.Add(tx2, false); err != nil {
		t.Errorf("It fails to add second transaction after removal")
	}

	var next *txElement
	for e := l.Front(); e != nil; e = next {
		next = e.Next()
		if !l.Remove(e) {
			t.Error("Fail to remove contained item")
		}
	}
	if l.Front() != nil || l.Len() != 0 {
		t.Error("After removing all, it still has something")
	}
}

func TestTransactionList_TestSort(t *testing.T) {
	from1 := common.MustNewAddressFromString("hx0000000000000000000000000000000000000001")
	from2 := common.MustNewAddressFromString("hx0000000000000000000000000000000000000002")
	tx1 := newMockTransaction([]byte{0x00, 0x00, 0x00, 0x01}, from1, 1)
	tx2 := newMockTransaction([]byte{0x00, 0x00, 0x00, 0x02}, from1, 2)
	tx3 := newMockTransaction([]byte{0x00, 0x00, 0x00, 0x03}, from2, 1)
	tx4 := newMockTransaction([]byte{0x00, 0x00, 0x00, 0x04}, from2, 2)
	tx5 := newMockTransaction([]byte{0x00, 0x00, 0x00, 0x05}, from2, 3)

	l := newTransactionList()
	l.Add(tx2, false)
	l.Add(tx1, false)
	if l.Front().Value() != tx1 {
		t.Error("tx1 should be first(by re-ordering)")
	}

	l = newTransactionList()
	l.Add(tx2, false)
	l.Add(tx3, false)
	l.Add(tx5, false)
	l.Add(tx1, false)
	if l.Front().Value() != tx1 {
		t.Error("tx1 should be first(by re-ordering)")
	}

	l.RemoveTx(tx3)
	l.Add(tx4, false)
	e := l.Front()
	if tx := e.Value(); tx != tx1 {
		t.Errorf("First item should be tx1 but tx=%x", tx.ID())
	}
	e = e.Next()
	if tx := e.Value(); tx != tx2 {
		t.Errorf("Second item should be tx2 but tx=%x", tx.ID())
	}

	l.RemoveTx(tx2)
	l.RemoveTx(tx1)
	e = l.Front()
	if tx := e.Value(); tx != tx4 {
		t.Errorf("First item should be tx4 but tx=%x", tx.ID())
	}
	e = e.Next()
	if tx := e.Value(); tx != tx5 {
		t.Errorf("Second items should be tx5 but tx=%x", tx.ID())
	}

	l = newTransactionList()
	l.Add(tx5, false)
	l.Add(tx4, false)
	l.RemoveTx(tx5)
	l.Add(tx3, false)

	e = l.Front()
	if tx := e.Value(); tx != tx3 {
		t.Errorf("First item should be tx3 but tx=%x", tx.ID())
	}
	e = e.Next()
	if tx := e.Value(); tx != tx4 {
		t.Errorf("First item should be tx4 but tx=%x", tx.ID())
	}
}
