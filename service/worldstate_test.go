package service

import (
	"bytes"
	"log"
	"math/big"
	"testing"

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
	balance2 := big.NewInt(2000)
	contractAddr := new(common.Address)
	contractAddr.SetString("cx001")
	contractOwner := new(common.Address)
	contractOwner.SetString("'0x12345")

	type testStruct struct {
		testStatus   contractStatus
		testApiInfo  []byte
		testCodeHash []byte
		testAuditTx  []byte
		testDeployTx []byte
		testParams   []byte
	}

	test := []*testStruct{
		&testStruct{
			testStatus:   csActive,
			testApiInfo:  []byte("APIINFO"),
			testCodeHash: []byte("CODEHASH"),
			testAuditTx:  []byte("AUDITTX"),
			testDeployTx: []byte("DEPLOYTX"),
			testParams:   []byte("PARAMS"),
		},
		&testStruct{
			testStatus:   csRejected,
			testApiInfo:  []byte("APIINFO2"),
			testCodeHash: []byte("CODEHASH2"),
			testAuditTx:  []byte("AUDITTX2"),
			testDeployTx: []byte("DEPLOYTX2"),
			testParams:   []byte("PARAMS2"),
		},
	}

	db := db.NewMapDB()
	ws := NewWorldState(db, nil, nil)
	as := ws.GetAccountState(contractAddr.ID())

	as.SetBalance(balance1)
	as.SetContractOwner(contractOwner)
	if bytes.Compare(as.GetContractOwner().Bytes(), contractOwner.Bytes()) != 0 {
		log.Panicf("Wrong ContractOwner : %x\n", as.GetContractOwner().Bytes())
	}

	cc := as.GetCurContract()
	c := func(c Contract, i int) {
		c.SetStatus(test[i].testStatus)
		c.SetApiInfo(test[i].testApiInfo)
		c.SetCodeHash(test[i].testCodeHash)
		c.SetAuditTx(test[i].testAuditTx)
		c.SetDeployTx(test[i].testDeployTx)
		c.SetParams(test[i].testParams)
	}
	if cc == nil {
		cc = newContractImpl()
	}
	c(cc, 0)

	as.SetCurContract(cc)

	tCc := as.GetCurContract()
	f := func(c ContractSnapshot, i int) {
		if c.GetStatus() != test[i].testStatus {
			log.Panicf("Wrong status. %d\n", c.GetStatus())
		}
		if bytes.Compare(c.GetApiInfo(), test[i].testApiInfo) != 0 {
			log.Panicf("Wrong ApiInop. %x\n", c.GetApiInfo())
		}
		if bytes.Compare(c.GetCodeHash(), test[i].testCodeHash) != 0 {
			log.Panicf("Wrong GetCodeHash. %x\n", c.GetCodeHash())
		}
		if bytes.Compare(c.GetAuditTx(), test[i].testAuditTx) != 0 {
			log.Panicf("Wrong GetAuditTx. %x\n", c.GetAuditTx())
		}
		if bytes.Compare(c.GetDeployTx(), test[i].testDeployTx) != 0 {
			log.Panicf("Wrong GetDeployTx. %x\n", c.GetDeployTx())
		}
		if bytes.Compare(c.GetParams(), test[i].testParams) != 0 {
			log.Panicf("Wrong GetParams. %x\n", c.GetParams())
		}
	}
	f(tCc, 0)

	ws1 := ws.GetSnapshot()

	ss1 := ws1.GetAccountSnapshot(contractAddr.ID())

	if bytes.Compare(ss1.GetContractOwner().Bytes(), contractOwner.Bytes()) != 0 {
		log.Panicf("Wrong ContractOwner : %x\n", as.GetContractOwner().Bytes())
	}
	if ss1.GetBalance().Cmp(balance1) != 0 {
		log.Panicf("Wrong balance. %s\n", ss1.GetBalance().String())
	}
	sCc1 := ss1.GetCurContract()
	f(sCc1, 0)

	c(cc, 1)
	as.SetCurContract(cc)

	tCc2 := as.GetCurContract()
	f(tCc2, 1)
	as.SetBalance(balance2)

	ws.Reset(ws1)
	ss2 := ws1.GetAccountSnapshot(contractAddr.ID())
	sCc2 := ss2.GetCurContract()
	f(sCc2, 0)
	if as.GetBalance().Cmp(balance2) != 0 {
		log.Panicf("Wrong balance. %s\n", as.GetBalance().String())
	}

	sn := ws.GetSnapshot()
	sn.Flush()
	sh := sn.StateHash()
	ws2 := NewWorldState(db, sh, nil)
	as2 := ws2.GetAccountState(contractAddr.ID())
	cc = as2.GetCurContract()
	if bytes.Compare(as2.GetContractOwner().Bytes(), contractOwner.Bytes()) != 0 {
		log.Panicf("Wrong ContractOwner : %x\n", as2.GetContractOwner().Bytes())
	}
	if as2.GetBalance().Cmp(balance1) != 0 {
		log.Panicf("Wrong balance. %s\n", as2.GetBalance().String())
	}
	f(cc, 0)
}
