package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
)

func TestAccountSnapshot_Equal(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil, nil, false)

	s1 := as.GetSnapshot()
	if !s1.Equal(s1) {
		t.Errorf("Fail to check equality with same snapshot")
		return
	}

	s2 := as.GetSnapshot()
	if !s1.Equal(s2) {
		t.Errorf("Fail to check equality with another snapshot without change")
		return
	}

	v1 := s1.GetBalance()
	as.SetBalance(new(big.Int).Add(v1, big.NewInt(30)))

	s3 := as.GetSnapshot()
	if s1.Equal(s3) {
		t.Errorf("Fail to compare snapshot after SetBalance()")
	}

	kv := []byte("Test")
	as.SetValue(kv, kv)

	s4 := as.GetSnapshot()
	if s3.Equal(s4) {
		t.Errorf("Fail to compare snapshot after SetValue()")
	}

	as.DeleteValue(kv)

	s5 := as.GetSnapshot()
	if !s3.Equal(s5) {
		t.Errorf("Fail to compare snapshot after DeleteValue()")
	}
}

func TestAccountSnapshot_Bytes(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil, nil, false)
	v1 := big.NewInt(3000)
	as.SetBalance(v1)
	tv := []byte("Puha")
	as.SetValue(tv, tv)
	s1 := as.GetSnapshot()

	serialized := s1.Bytes()
	s1.Flush()

	t.Logf("Serialized:% X", serialized)

	s2 := new(accountSnapshotImpl)
	s2.Reset(database, serialized)

	assert.Equal(t, serialized, s2.Bytes())

	v2 := s2.GetBalance()
	if v1.Cmp(v2) != 0 {
		t.Errorf("Fail to get same balance")
	}

	tv2, _ := s2.GetValue(tv)
	assert.Equal(t, tv, tv2)
}

func TestAccountState_DepositTest(t *testing.T) {
	database := db.NewMapDB()

	tid1 := []byte{0x00}
	// tid2 := []byte{0x01}
	dc := &depositContext{
		rate:   defaultDepositIssueRate,
		price:  big.NewInt(100),
		height: 10,
		period: 100,
		tid:    tid1,
	}
	sender := common.MustNewAddressFromString("hx0000000000000000000000000000000000000001")
	amount := big.NewInt(50000)

	as := newAccountState(database, nil, nil, false)
	as.InitContractAccount(sender)

	err := as.AddDeposit(dc, amount)
	assert.NoError(t, err)

	ass := as.GetSnapshot()
	ass.Flush()
	serialized := ass.Bytes()

	ass2 := new(accountSnapshotImpl)
	ass2.Reset(database, serialized)

	as2 := newAccountState(database, ass2, nil, false)

	dc.height += 1
	am, fee, err := as2.WithdrawDeposit(dc, tid1, nil)
	assert.NoError(t, err)
	assert.True(t, fee.Sign() == 0)
	assert.True(t, am.Cmp(amount) == 0)
}

func assertAccountSnapshot(t *testing.T, dbase db.Database, ass AccountSnapshot, code []byte, next int, graph []byte) {
	c := ass.Contract()
	code1, err := c.Code()
	assert.NoError(t, err)
	assert.Equal(t, code, code1)

	next1, _, graph1, err := ass.GetObjGraph(c.CodeID(), true)
	assert.NoError(t, err)
	assert.Equal(t, next, next1)
	assert.Equal(t, graph, graph1)

	ass1 := ass.(*accountSnapshotImpl)
	as := newAccountState(dbase, ass1, nil, false)

	c = as.Contract()
	code1, err = c.Code()
	assert.NoError(t, err)
	assert.Equal(t, code, code1)

	next1, _, graph1, err = as.GetObjGraph(c.CodeID(), true)
	assert.NoError(t, err)
	assert.Equal(t, next, next1)
	assert.Equal(t, graph, graph1)
}

func TestAccountStateImpl_SetObjGraph(t *testing.T) {
	var (
		code1    = []byte("application-code1")
		tx1      = []byte{0x00, 0x01}
		next1v1  = 2
		graph1v1 = []byte{0x01, 0x01, 0x01}
		next1v2  = 3
		graph1v2 = []byte{0x01, 0x01, 0x02}

		code2    = []byte("application-code2")
		tx2      = []byte{0x00, 0x02}
		next2v1  = 10
		graph2v1 = []byte{0x02, 0x02, 0x02}
	)

	dbase := db.NewMapDB()
	as := newAccountState(dbase, nil, nil, false)
	sender := common.MustNewAddressFromString("hx0000000000000000000000000000000000000001")

	r := as.InitContractAccount(sender)
	assert.True(t, r)

	_, err := as.DeployContract(code1, JavaEE, "application/jar", nil, tx1)
	assert.NoError(t, err)

	c1 := as.NextContract()
	next, graphHash, graph, err := as.GetObjGraph(c1.CodeID(), true)
	assert.Error(t, err)
	assert.Zero(t, next)
	assert.Nil(t, graphHash)
	assert.Nil(t, graph)

	err = as.SetObjGraph(c1.CodeID(), true, next1v1, graph1v1)
	assert.NoError(t, err)

	err = as.AcceptContract(tx1, tx1)
	assert.NoError(t, err)

	ass1 := as.GetSnapshot()

	c2 := as.Contract()
	next, _, graph, err = as.GetObjGraph(c2.CodeID(), true)
	assert.NoError(t, err)
	assert.Equal(t, next1v1, next)
	assert.Equal(t, graph1v1, graph)

	_, err = as.DeployContract(code2, JavaEE, "application/jar", nil, tx2)
	assert.NoError(t, err)

	c3 := as.NextContract()
	next, graphHash, graph, err = as.GetObjGraph(c3.CodeID(), true)
	assert.Error(t, err)
	assert.Zero(t, next)
	assert.Nil(t, graphHash)
	assert.Nil(t, graph)

	err = as.SetObjGraph(c3.CodeID(), true, next2v1, graph2v1)
	assert.NoError(t, err)

	err = as.AcceptContract(tx2, tx2)
	assert.NoError(t, err)

	ass2 := as.GetSnapshot()

	err = as.SetObjGraph(c2.CodeID(), true, next1v2, graph1v2)
	assert.NoError(t, err)

	ass3 := as.GetSnapshot()

	next, _, graph, err = as.GetObjGraph(c2.CodeID(), true)
	assert.NoError(t, err)
	assert.Equal(t, next1v2, next)
	assert.Equal(t, graph1v2, graph)

	assertAccountSnapshot(t, dbase, ass1, code1, next1v1, graph1v1)
	assertAccountSnapshot(t, dbase, ass2, code2, next2v1, graph2v1)
	assertAccountSnapshot(t, dbase, ass3, code2, next2v1, graph2v1)

	// check last code and environment
	err = ass3.Flush()
	assert.NoError(t, err)
	bs := ass3.Bytes()

	ass := new(accountSnapshotImpl)
	err = ass.Reset(dbase, bs)
	assert.NoError(t, err)

	assertAccountSnapshot(t, dbase, ass, code2, next2v1, graph2v1)
}
