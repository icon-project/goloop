package state

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
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

	err = as.ActivateNextContract()
	assert.NoError(t, err)

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

	err = as.ActivateNextContract()
	assert.NoError(t, err)

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

func TestAccountData_State(t *testing.T) {
	var expected struct {
		IsDisabled       bool
		IsBlocked        bool
		UseSystemDeposit bool
		IsContract       bool
		ContractOwner    module.Address
	}
	assertState := func(t *testing.T, ad AccountData) {
		assert.Equal(t, expected.IsDisabled, ad.IsDisabled())
		assert.Equal(t, expected.IsBlocked, ad.IsBlocked())
		assert.Equal(t, expected.UseSystemDeposit, ad.UseSystemDeposit())
		assert.Equal(t, expected.IsContract, ad.IsContract())
		assert.True(t, common.AddressEqual(expected.ContractOwner, ad.ContractOwner()))
	}

	dbase := db.NewMapDB()
	ass := newAccountSnapshot(dbase)
	assert.True(t, ass.IsEmpty())
	assertState(t, ass)

	// state has same value as snapshot
	as := newAccountState(dbase, ass, nil, false)
	assert.True(t, as.IsEmpty())
	assertState(t, as)

	// disabling EoA does nothing
	as.SetDisable(true)
	assertState(t, as)

	// blocking EoA should work
	as.SetBlock(true)
	expected.IsBlocked = true
	assertState(t, as)
	ass2 := as.GetSnapshot()
	assertState(t, ass2)

	// UseSystemDeposit can't be set on EoA
	err := as.SetUseSystemDeposit(true)
	assert.Error(t, err)
	assertState(t, as)

	// recover it to initial state
	as.SetBlock(false)
	expected.IsBlocked = false
	assertState(t, as)

	owner1 := common.MustNewAddressFromString("hx123456")

	err = as.SetContractOwner(owner1)
	assert.Error(t, err)
	assertState(t, as)

	// change to a contract
	ok := as.InitContractAccount(owner1)
	assert.True(t, ok)
	expected.IsContract = true
	expected.ContractOwner = owner1
	assertState(t, as)

	// no need to do DEPLOY and ACCEPT for testing state

	// disable contract
	as.SetDisable(true)
	expected.IsDisabled = true
	assertState(t, as)
	ass3 := as.GetSnapshot()
	assertState(t, ass3)

	// use system deposit
	err = as.SetUseSystemDeposit(true)
	assert.NoError(t, err)
	expected.UseSystemDeposit = true
	assertState(t, as)
	ass4 := as.GetSnapshot()
	assertState(t, ass4)

	// blocking contract should work
	as.SetBlock(true)
	expected.IsBlocked = true
	assertState(t, as)
	ass5 := as.GetSnapshot()
	assertState(t, ass5)

	// change owner
	owner2 := common.MustNewAddressFromString("hx123457")
	err = as.SetContractOwner(owner2)
	assert.NoError(t, err)
	expected.ContractOwner = owner2
	assertState(t, as)
	ass6 := as.GetSnapshot()
	assertState(t, ass6)
}

func handleTestDeploy(t *testing.T, as AccountState, txID []byte, code []byte) []byte {
	oldDeploy, err := as.DeployContract(code, JavaEE, CTAppJava, []byte(`{ "name": "test"}`), txID)
	assert.NoError(t, err)

	// check the result
	ct1 := as.NextContract()
	assert.NotNil(t, ct1)
	code1, err := ct1.Code()
	assert.NoError(t, err)
	assert.Equal(t, code, code1)
	return oldDeploy
}

func handleTestFlush(t *testing.T, as AccountState) AccountSnapshot {
	ass := as.GetSnapshot()
	err := ass.Flush()
	assert.NoError(t, err)
	return ass
}

func handleTestAccept(t *testing.T, as AccountState, txID []byte, deployTx []byte, rev module.Revision, apiInfo *scoreapi.Info, handleInit func() ) {
	ct1 := as.NextContract()
	assert.Equal(t, CSPending, ct1.Status())
	err := as.ActivateNextContract()
	assert.NoError(t, err)
	assert.Equal(t, CSActive, ct1.Status())
	code, err := ct1.Code()
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// migrate version if possible
	err = as.MigrateForRevision(rev)
	assert.NoError(t, err)

	// set API info from the contract
	as.SetAPIInfo(apiInfo)

	if handleInit != nil {
		handleInit()
	}

	// accept contract after init
	err = as.AcceptContract(deployTx, txID)
	assert.NoError(t, err)

	// check the result
	assert.Nil(t, as.NextContract())
	ct2 := as.ActiveContract()
	assert.NotNil(t, ct2)
	assert.Equal(t, CSActive, ct2.Status())
	code2, err := ct2.Code()
	assert.Equal(t, code, code2)
	assert.Equal(t, ct2.DeployTxHash(), deployTx)
	assert.Equal(t, ct2.AuditTxHash(), txID)
}

func recoverAccountSnapshotFromBytes(t *testing.T, dbase db.Database, bs []byte) AccountSnapshot {
	assValue := reflect.New(AccountType.Elem())
	ass := assValue.Interface().(AccountSnapshot)
	err := ass.Reset(dbase, bs)
	assert.NoError(t, err)
	return ass
}

func TestAccount_Deploy(t *testing.T) {
	dbase := db.NewMapDB()
	accountKey := []byte("\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1\xa1")
	owner := common.MustNewAddressFromString("hxa1")
	apiInfo := scoreapi.NewInfo([]*scoreapi.Method{
		{
			Name:    "transfer",
			Type:    scoreapi.Function,
			Flags:   scoreapi.FlagExternal,
			Indexed: 3,
			Inputs:  []scoreapi.Parameter{},
		},
	})
	deployTx := crypto.SHA3Sum256([]byte("dummy_test_deploy_tx"))
	acceptTx := crypto.SHA3Sum256([]byte("dummy_test_accept_tx"))
	rejectTx := crypto.SHA3Sum256([]byte("dummy_test_reject_tx"))

	deployTx2 := crypto.SHA3Sum256([]byte("dummy_test_deploy_tx2"))
	// acceptTx2 := crypto.SHA3Sum256([]byte("dummy_test_accept_tx2"))


	t.Run("DeployAndAcceptAtOnce", func(t *testing.T) {
		var code = []byte("dummy code base")
		var err error
		ass := newAccountSnapshot(dbase)
		as := newAccountState(dbase, ass, accountKey, false)

		// expected failures
		err = as.AcceptContract(deployTx, acceptTx)
		assert.Error(t, err)

		// deploy new one
		ok := as.InitContractAccount(owner)
		assert.True(t, ok)
		assert.True(t, as.IsContract())
		old := handleTestDeploy(t, as, deployTx, code)
		assert.Empty(t, old)

		// accept
		handleTestAccept(t, as, deployTx, deployTx, module.LatestRevision, apiInfo, func() {
			var err error
			// possible expected failures during calling <init>
			err = as.ActivateNextContract() // another accept by <init>
			assert.Error(t, err)
			err = as.RejectContract(deployTx, deployTx) // reject by <init>
			assert.Error(t, err)

			// another deploy by <init>
			bs, err := as.DeployContract(code, JavaEE, CTAppJava, []byte(`{ "name": "test"}`), deployTx2)
			assert.Error(t, err)
			assert.Empty(t, bs)
		})

		// flush
		ass1 := as.GetSnapshot()
		assert.False(t, ass.Equal(ass1))
		err = ass1.Flush()
		assert.NoError(t, err)
		accountBytes := ass1.Bytes()

		// recover state
		ass2 := recoverAccountSnapshotFromBytes(t, dbase, accountBytes)

		// check data
		assert.True(t, ass2.IsContract())
		apiInfo2, err := ass2.APIInfo()
		assert.NoError(t, err)
		assert.EqualValues(t, apiInfo, apiInfo2)
		ct3 := ass2.ActiveContract()
		code3, err := ct3.Code()
		assert.NoError(t, err)
		assert.Equal(t, code, code3)
	})

	t.Run( "DeployThenAccept", func(t *testing.T) {
		var code = []byte("dummy code2")
		var err error

		ass := newAccountSnapshot(dbase)
		as := newAccountState(dbase, ass, accountKey, false)

		// deploy
		ok := as.InitContractAccount(owner)
		assert.True(t, ok)
		assert.True(t, as.IsContract())
		old := handleTestDeploy(t, as, deployTx, code)
		assert.Empty(t, old)

		// check api info isn't available
		apiInfo1, err := as.APIInfo()
		assert.NoError(t, err)
		assert.Nil(t, apiInfo1)

		// flush
		ass1 := as.GetSnapshot()
		assert.False(t, ass.Equal(ass1))
		err = ass1.Flush()
		assert.NoError(t, err)
		accountBytes := ass1.Bytes()

		// recover state
		ass2 := recoverAccountSnapshotFromBytes(t, dbase, accountBytes)
		as = newAccountState(dbase, ass2, accountKey, false)

		// accept
		handleTestAccept(t, as, acceptTx, deployTx, module.LatestRevision, apiInfo, nil)

		// check data
		apiInfo2, err := as.APIInfo()
		assert.NoError(t, err)
		assert.EqualValues(t, apiInfo, apiInfo2)
		ct1 := as.ActiveContract()
		code1, err := ct1.Code()
		assert.NoError(t, err)
		assert.Equal(t, code, code1)

		err = as.RejectContract(rejectTx, deployTx)
		assert.Error(t, err)
	})
	t.Run( "DeployDeployThenAccept", func(t *testing.T) {
		var code = []byte("dummy code2")
		var codeNew = []byte("dummy code new")
		var err error

		ass := newAccountSnapshot(dbase)
		as := newAccountState(dbase, ass, accountKey, false)

		// deploy
		ok := as.InitContractAccount(owner)
		assert.True(t, ok)
		assert.True(t, as.IsContract())
		old := handleTestDeploy(t, as, deployTx, code)
		assert.Empty(t, old)

		// deploy again
		old = handleTestDeploy(t, as, deployTx2, codeNew)
		assert.Equal(t, deployTx, old)

		// fail to accept on old one
		err = as.AcceptContract(deployTx, acceptTx)
		assert.Error(t, err)

		// accept
		handleTestAccept(t, as, acceptTx, deployTx2, module.LatestRevision, apiInfo, nil)

		// check data
		apiInfo2, err := as.APIInfo()
		assert.NoError(t, err)
		assert.EqualValues(t, apiInfo, apiInfo2)
		ct1 := as.ActiveContract()
		code1, err := ct1.Code()
		assert.NoError(t, err)
		assert.Equal(t, codeNew, code1)

		// fail to reject
		err = as.RejectContract(rejectTx, deployTx)
		assert.Error(t, err)
	})

	t.Run( "DeployThenReject", func(t *testing.T) {
		var code = []byte("dummy code3")
		var err error

		ass := newAccountSnapshot(dbase)
		as := newAccountState(dbase, ass, accountKey, false)

		// deploy new
		ok := as.InitContractAccount(owner)
		assert.True(t, ok)
		assert.True(t, as.IsContract())
		handleTestDeploy(t, as, deployTx, code)

		// flush
		ass1 := handleTestFlush(t, as)
		assert.False(t, ass.Equal(ass1))
		accountBytes := ass1.Bytes()

		// recover state
		ass2 := recoverAccountSnapshotFromBytes(t, dbase, accountBytes)
		as = newAccountState(dbase, ass2, accountKey, false)

		// expected failures
		err = as.RejectContract(deployTx2, rejectTx)
		assert.Error(t, err)

		// reject it
		err = as.RejectContract(deployTx, rejectTx)
		assert.NoError(t, err)

		// flush
		ass3 := handleTestFlush(t, as)
		assert.False(t, ass.Equal(ass3))
		accountBytes = ass3.Bytes()

		// recover state
		ass4 := recoverAccountSnapshotFromBytes(t, dbase, accountBytes)
		as = newAccountState(dbase, ass4, accountKey, false)

		// check data
		apiInfo2, err := as.APIInfo()
		assert.NoError(t, err)
		assert.Nil(t, apiInfo2)
		ct1 := as.ActiveContract()
		assert.Nil(t, ct1)

		// check failures after reject
		err = as.ActivateNextContract()
		assert.Error(t, err)
		err = as.AcceptContract(deployTx, acceptTx)
		assert.Error(t, err)
	})
}

type testContext struct {
	PayContext
	DepositContext

	fsEnabled bool
	height    int64
	stepPrice *big.Int
	stepLimit *big.Int
	txID      []byte
}

func (ctx *testContext) FeeSharingEnabled() bool {
	return ctx.fsEnabled
}

func (ctx *testContext) StepPrice() *big.Int {
	return ctx.stepPrice
}

func (ctx *testContext) FeeLimit() *big.Int {
	return new(big.Int).Mul(ctx.stepPrice, ctx.stepLimit)
}

func (ctx *testContext) DepositTerm() int64 {
	return 0
}

func (ctx *testContext) TransactionID() []byte {
	return ctx.txID
}

func (ctx *testContext) BlockHeight() int64 {
	return ctx.height
}

func TestAccountData_CanAcceptTx(t *testing.T) {
	dbase := db.NewMapDB()
	var ass AccountSnapshot = newAccountSnapshot(dbase)
	as := newAccountState(dbase, ass, nil, false)

	ctx := &testContext{}

	// EoA accept
	ass = as.GetSnapshot()
	assert.True(t, ass.CanAcceptTx(ctx))
	assNormal := ass

	// blocked EoA accept
	as.SetBlock(true)
	ass =  as.GetSnapshot()
	assert.True(t, ass.CanAcceptTx(ctx))

	// normal contract accept
	assert.NoError(t, as.Reset(assNormal))
	as.InitContractAccount(common.MustNewAddressFromString("cx1234"))
	assContract := as.GetSnapshot()
	assert.True(t, assContract.CanAcceptTx(ctx))

	// disabled contract DO NOT accept
	as.SetDisable(true)
	ass = as.GetSnapshot()
	assert.False(t, ass.CanAcceptTx(ctx))

	// blocked contract DO NOT accept
	assert.NoError(t, as.Reset(assContract))
	as.SetBlock(true)
	ass = as.GetSnapshot()
	assert.False(t, ass.CanAcceptTx(ctx))

	// enable fee sharing feature
	ctx.fsEnabled = true
	ctx.stepLimit = big.NewInt(100)
	ctx.stepPrice = big.NewInt(1000)

	// contract without deposit accept
	assert.NoError(t, as.Reset(assContract))
	ass = as.GetSnapshot()
	assert.True(t, ass.CanAcceptTx(ctx))

	// contract with deposit >= FeeLimit accept
	assert.NoError(t, as.Reset(assContract))
	assert.NoError(t, as.AddDeposit(ctx, big.NewInt(100000)) )
	ass = as.GetSnapshot()
	assert.True(t, ass.CanAcceptTx(ctx))

	// contract with deposit < FeeLimit DO NOT accept
	assert.NoError(t, as.Reset(assContract))
	assert.NoError(t, as.AddDeposit(ctx, big.NewInt(10000)) )
	ass = as.GetSnapshot()
	assert.False(t, ass.CanAcceptTx(ctx))
}
