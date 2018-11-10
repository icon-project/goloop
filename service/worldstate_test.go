package service

import (
	"github.com/icon-project/goloop/common/db"
	"math/big"
	"testing"
)

func TestNewWorldState(t *testing.T) {
	balance1 := big.NewInt(0x1000)
	balance2 := big.NewInt(0x2000)

	testid := []byte("test")

	database := db.NewMapDB()
	ws := NewWorldState(database, nil)
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
