package service

import (
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/icon-project/goloop/module"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/db"
)

func TestNewWorldState(t *testing.T) {
	balance1 := big.NewInt(0x1000)
	balance2 := big.NewInt(0x2000)

	testid := []byte("test")

	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil)
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
		testStatus      contractStatus
		testContentType string
		testEeType      string
		testApiInfo     []byte
		testCode        []byte
		testAuditTx     []byte
		testDeployTx    []byte
		testParams      []byte
	}

	test := []*testStruct{
		{
			testStatus:      csActive,
			testContentType: "Application/Zip",
			testEeType:      "Python",
			testApiInfo:     []byte("APIINFO"),
			testCode:        []byte("CODEHASH"),
			testAuditTx:     []byte("AUDITTX"),
			testDeployTx:    []byte("DEPLOYTX"),
			testParams:      []byte("PARAMS"),
		},
		{
			testStatus:      csRejected,
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
	ws := NewWorldState(db, nil, nil)
	as := ws.GetAccountState(contractAddr.ID())

	as.SetBalance(balance1)

	c := func(a AccountState, owner module.Address, i int) {
		a.InitContractAccount(owner)
		a.DeployContract(test[i].testCode, test[i].testEeType,
			test[i].testContentType, test[i].testParams, test[i].testDeployTx)
	}
	c(as, contractOwner, 0)
	if as.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}

	snapshot := as.GetSnapshot()
	if snapshot.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}
	if contract := snapshot.Contract(); contract != nil {
		curCode, _ := contract.Code()
		if len(curCode) != 0 {
			log.Panicf("Wrong contrac. %x\n", curCode)
		}
	}

	if contract := snapshot.NextContract(); contract != nil {
		nextCode, _ := contract.Code()
		if bytes.Equal(nextCode, test[0].testCode) == false {
			log.Panicf("Wrong nextCode %x\n", nextCode)
		}
	}

	wsSnapshot := ws.GetSnapshot()
	wsAs := wsSnapshot.GetAccountSnapshot(contractAddr.ID())
	if wsAs.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}
	if contract := wsAs.Contract(); contract != nil {
		curCode, _ := contract.Code()
		if len(curCode) != 0 {
			log.Panicf("Wrong contrac. %x\n", curCode)
		}
	}

	if contract := wsAs.NextContract(); contract != nil {
		nextCode, _ := contract.Code()
		if bytes.Equal(nextCode, test[0].testCode) == false {
			log.Panicf("Wrong nextCode %x\n", nextCode)
		}
	}

	wsSnapshot.Flush()
	hash := wsSnapshot.StateHash()

	ws2 := NewWorldState(db, hash, nil)
	as2 := ws2.GetAccountState(contractAddr.ID())
	if as2.IsContractOwner(contractOwner) == false {
		log.Panicf("Wrong contractOwner. %s\n", contractOwner)
	}
	if contract := as2.NextContract(); contract != nil {
		nextCode, _ := contract.Code()
		if bytes.Equal(nextCode, test[0].testCode) == false {
			log.Panicf("Wrong contrac. %x\n", nextCode)
		}
	}

	if contract := as2.Contract(); contract != nil {
		curCode, _ := contract.Code()
		if len(curCode) != 0 {
			log.Panicf("Wrong curCode %x\n", curCode)
		}
	}

	as2.AcceptContract(test[0].testDeployTx, test[0].testAuditTx)
	if contract := as2.NextContract(); contract != nil {
		nextCode, _ := contract.Code()
		if len(nextCode) != 0 {
			log.Panicf("Wrong contract. %x\n", nextCode)
		}
	}

	if contract := as2.Contract(); contract != nil {
		curCode, _ := contract.Code()
		if bytes.Equal(curCode, test[0].testCode) == false {
			log.Panicf("Wrong curCode %x\n", curCode)
		}
	}
}
