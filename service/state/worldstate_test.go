package state

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestNewWorldState(t *testing.T) {
	balance1 := big.NewInt(0x1000)
	balance2 := big.NewInt(0x2000)

	testid := []byte("test")

	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil, nil, nil)
	as := ws.GetAccountState(testid)

	as.SetBalance(balance1)
	s1 := ws.GetSnapshot()
	ac1 := s1.GetAccountSnapshot(testid)
	rb1 := ac1.GetBalance()
	if rb1.Cmp(balance1) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for snapshot1", rb1.String(), balance1.String())
		return
	}

	as.SetBalance(balance2)
	s2 := ws.GetSnapshot()
	ac2 := s2.GetAccountSnapshot(testid)
	rb2 := ac2.GetBalance()
	if rb2.Cmp(balance2) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for snapshot2", rb2.String(), balance2.String())
		return
	}

	ws.Reset(s1)
	as2 := ws.GetAccountState(testid)
	rb3 := as2.GetBalance()
	if rb3.Cmp(balance1) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for state with snapshot1 ", rb3.String(), balance1.String())
		return
	}

	s1.Flush()
}

func TestNewWorldStateWithContract(t *testing.T) {
	balance1 := big.NewInt(1000)
	//balance2 := big.NewInt(2000)
	contractAddr := new(common.Address)
	contractAddr.SetString("cx001")
	contractOwner := new(common.Address)
	contractOwner.SetString("'0x12345")

	type testStruct struct {
		testStatus      ContractStatus
		testContentType string
		testEeType      EEType
		testApiInfo     []byte
		testCode        []byte
		testAuditTx     []byte
		testDeployTx    []byte
		testParams      []byte
	}

	test := []*testStruct{
		{
			testStatus:      CSActive,
			testContentType: "Application/Zip",
			testEeType:      "Python",
			testApiInfo:     []byte("APIINFO"),
			testCode:        []byte("CODEHASH"),
			testAuditTx:     []byte("AUDITTX"),
			testDeployTx:    []byte("DEPLOYTX"),
			testParams:      []byte("PARAMS"),
		},
		{
			testStatus:      CSRejected,
			testContentType: "Application/Zip2",
			testEeType:      "Python2",
			testApiInfo:     []byte("APIINFO2"),
			testCode:        []byte("CODEHASH2"),
			testAuditTx:     []byte("AUDITTX2"),
			testDeployTx:    []byte("DEPLOYTX2"),
			testParams:      []byte("PARAMS2"),
		},
	}

	db := db.NewMapDB()
	ws := NewWorldState(db, nil, nil, nil, nil)
	as := ws.GetAccountState(contractAddr.ID())

	as.SetBalance(balance1)

	d := func(a AccountState, owner module.Address, i int) {
		a.InitContractAccount(owner)
		a.DeployContract(test[i].testCode, test[i].testEeType,
			test[i].testContentType, test[i].testParams, test[i].testDeployTx)
	}
	d(as, contractOwner, 0)

	check := func(c ContractSnapshot, i int) {
		code, _ := c.Code()
		if bytes.Equal(code, test[i].testCode) == false {
			log.Panicf("Invalid code")
		}
		if bytes.Equal(c.Params(), test[i].testParams) == false {
			log.Panicf("Invalid params")
		}
		codeHash := sha3.Sum256(code)
		if bytes.Equal(c.CodeHash(), codeHash[:]) == false {
			log.Panicf("Invalide codeHash")
		}
		if strings.Compare(c.ContentType(), test[i].testContentType) != 0 {
			log.Panicf("Invalid contentType %s\n", c.ContentType())
		}
		if c.EEType() != test[i].testEeType {
			log.Panicf("Invalid EEType")
		}
	}
	if as.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}

	snapshot := as.GetSnapshot()
	if snapshot.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}
	if contract := snapshot.Contract(); contract != nil {
		log.Panicf("Wrong contract.\n")
	}
	if contract := snapshot.ActiveContract(); contract != nil {
		log.Panicf("Wrong contract.\n")
	}

	if contract := snapshot.NextContract(); contract != nil {
		check(contract, 0)
	}

	wsSnapshot := ws.GetSnapshot()
	wsAs := wsSnapshot.GetAccountSnapshot(contractAddr.ID())
	if wsAs.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}
	check(wsAs.NextContract(), 0)

	if contract := wsAs.NextContract(); contract == nil {
		log.Panicf("Invalid nextContract\n")
	} else {
		check(contract, 0)
	}

	wsSnapshot.Flush()
	hash := wsSnapshot.StateHash()

	ws2 := NewWorldState(db, hash, nil, nil, nil)
	as2 := ws2.GetAccountState(contractAddr.ID())
	if as2.IsContractOwner(contractOwner) == false {
		log.Panicf("Invalid contractOwner. %s\n", contractOwner)
	}
	if contract := as2.NextContract(); contract == nil {
		log.Panicf("Invalid contract.\n")
	} else {
		check(contract, 0)
		if contract.Status() != CSPending {
			log.Panicf("Invalid state %d\n", contract.Status())
		}
	}

	if as2.Contract() != nil {
		log.Panicf("Invalid Contract\n")
	}

	if as2.ActiveContract() != nil {
		log.Panicf("Invalid state\n")
	}

	if err := as2.AcceptContract(test[0].testDeployTx, test[0].testAuditTx); err != nil {
		t.Errorf("Fail to AcceptContract err=%+v", err)
		return
	}
	if as2.NextContract() != nil {
		log.Panicf("Invalid contract. \n")
	}

	if contract := as2.Contract(); contract == nil {
		log.Panicf("Invalid contract. \n")
	} else {
		check(contract, 0)
	}

	if contract := as2.ActiveContract(); contract == nil {
		log.Panicf("Invalid contract\n")
	} else {
		check(contract, 0)
		if contract.Status() != CSActive {
			log.Panicf("Invalid state %d\n", contract.Status())
		}
	}

	d(as2, contractOwner, 1)
	vContract1 := func(as AccountState) {
		if contract := as.Contract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 0)
		}
		if contract := as.ActiveContract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 0)
		}
		if contract := as.NextContract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 1)
		}
	}
	vContract1(as2)
	vContract2 := func(as AccountSnapshot) {
		if contract := as.Contract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 0)
		}
		if contract := as.ActiveContract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 0)
		}
		if contract := as.NextContract(); contract == nil {
			log.Panicf("Invalid Contract")
		} else {
			check(contract, 1)
		}
	}
	ss := as2.GetSnapshot()
	vContract2(ss)
	if v := ss.Version(); v != AccountVersion {
		log.Panicf("Invalid version. %d\n", v)
	}

	wsSnapshot = ws2.GetSnapshot()
	wsSnapshot.Flush()
	hash = wsSnapshot.StateHash()

	ws3 := NewWorldState(db, hash, nil, nil, nil)
	as3 := ws3.GetAccountState(contractAddr.ID())
	if as3.IsContractOwner(contractOwner) == false {
		log.Panicf("Invalid contractOwner. %s\n", contractOwner)
	}
	vContract1(as3)
	ass := as3.GetSnapshot()
	vContract2(ass)

	as3.SetDisable(true)
	if as3.ActiveContract() != nil {
		log.Panicf("Invalid activeContract")
	}
	as3.SetBlock(true)
	if as3.IsBlocked() == false {
		log.Panic("Not blacklisted", as3.IsBlocked())
	}
	if !as3.IsDisabled() {
		log.Panic("Not disabled")
	}
	as3.SetDisable(false)
	if as3.ActiveContract() != nil {
		log.Panicf("Invalid activeContract")
	}
	if as3.IsBlocked() == false {
		log.Panic("Not blacklisted", as3.IsBlocked())
	}
	wsSnapshot = ws3.GetSnapshot()
	wsSnapshot.Flush()
	hash = wsSnapshot.StateHash()
	ws4 := NewWorldState(db, hash, nil, nil, nil)
	as4 := ws4.GetAccountState(contractAddr.ID())
	if as4.ActiveContract() != nil {
		log.Panicf("Invalid activeContract")
	}
	if as4.IsBlocked() == false {
		log.Panic("Not blacklisted", as4.IsBlocked())
	}
	if v := as4.Version(); v != AccountVersion {
		log.Panicf("Not valid version. %d\n", v)
	}
}

func TestWorldStateImpl_GetSnapshot(t *testing.T) {
	assert := assert.New(t)

	dbase := db.NewMapDB()
	addr1 := common.MustNewAddressFromString("hx01")
	addr2 := common.MustNewAddressFromString("hx02")
	v1, err := ValidatorFromAddress(addr1)
	assert.NoError(err)

	vss, err := ValidatorSnapshotFromSlice(dbase, []module.Validator{v1})
	assert.NoError(err)

	ws := NewWorldState(dbase, nil, vss, nil, nil)

	var wss WorldSnapshot
	wss1 := ws.GetSnapshot()

	// GetSnapshot returns same snapshot with no operation
	wss = ws.GetSnapshot()
	assert.Equal(wss1, wss)

	// GetSnapshot returns same snapshot after GetAccountState()
	_ = ws.GetAccountState(addr1.ID())
	wss = ws.GetSnapshot()
	assert.Equal(wss1, wss)

	// GetSnapshot returns different snapshot after balance change
	as2 := ws.GetAccountState(addr2.ID())
	as2.SetBalance(big.NewInt(3))
	wss2 := ws.GetSnapshot()
	assert.NotEqual(wss1, wss2)

	// GetSnapshot returns different snapshot after storage change
	_, err = as2.SetValue([]byte("TestKey"), []byte("TestValue"))
	assert.NoError(err)
	wss3 := ws.GetSnapshot()
	assert.NotEqual(wss2, wss3)

	// GetSnapshot returns same snapshot after reset
	err = ws.Reset(wss1)
	assert.NoError(err)
	wss = ws.GetSnapshot()
	assert.Equal(wss1, wss)
}

func BenchmarkWorldStateImpl_GetSnapshotN(b *testing.B) {
	runs := []struct{ accounts int }{
		{1000},
		{2000},
	}

	for _, run := range runs {
		b.Run(fmt.Sprint(run.accounts), func(b *testing.B) {
			dbase := db.NewMapDB()
			addrs := make([]module.Address, run.accounts)
			for i := 0; i < len(addrs); i++ {
				addrs[i] = common.MustNewAddressFromString(fmt.Sprintf("hx%040d", i))
			}
			ws := NewWorldState(dbase, nil, nil, nil, nil)
			for idx, addr := range addrs {
				as := ws.GetAccountState(addr.ID())
				as.SetBalance(big.NewInt(int64(idx * 10)))
			}
			ws.GetSnapshot()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ws.GetSnapshot()
			}
		})
	}
}

func BenchmarkWorldStateImpl_ResetN(b *testing.B) {
	runs := []struct{ accounts int }{
		{1000},
		{2000},
	}

	for _, run := range runs {
		b.Run(fmt.Sprint(run.accounts), func(b *testing.B) {
			dbase := db.NewMapDB()
			addrs := make([]module.Address, run.accounts)
			for i := 0; i < len(addrs); i++ {
				addrs[i] = common.MustNewAddressFromString(fmt.Sprintf("hx%040d", i))
			}
			ws := NewWorldState(dbase, nil, nil, nil, nil)
			for idx, addr := range addrs[0 : len(addrs)/2] {
				as := ws.GetAccountState(addr.ID())
				as.SetBalance(big.NewInt(int64(idx * 10)))
			}
			for _, addr := range addrs[len(addrs)/2:] {
				ws.GetAccountState(addr.ID())
			}

			wss := ws.GetSnapshot()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for idx, addr := range addrs[len(addrs)/2:] {
					as := ws.GetAccountState(addr.ID())
					as.SetBalance(big.NewInt(int64(idx * 10)))
				}
				ws.Reset(wss)
			}
		})
	}
}
