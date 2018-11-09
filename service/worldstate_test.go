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
	ws := newWorldState(database, nil)
	as := ws.getAccountState(testid)

	as.setBalance(balance1)
	s1 := ws.getSnapshot()
	ac1 := s1.getAccountSnapshot(testid)
	rb1 := ac1.getBalance()
	if rb1.Cmp(balance1) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for snapshot1", rb1.String(), balance1.String())
		return
	}

	as.setBalance(balance2)
	s2 := ws.getSnapshot()
	ac2 := s2.getAccountSnapshot(testid)
	rb2 := ac2.getBalance()
	if rb2.Cmp(balance2) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for snapshot2", rb2.String(), balance2.String())
		return
	}

	ws.reset(s1)
	as2 := ws.getAccountState(testid)
	rb3 := as2.getBalance()
	if rb3.Cmp(balance1) != 0 {
		t.Errorf("Fail to check balance returned=%s expected=%s for state with snapshot1 ", rb3.String(), balance1.String())
		return
	}
}
