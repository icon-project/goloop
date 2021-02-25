package icstate

import (
	"fmt"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
)

func TestNodeOwnerCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr1node := common.MustNewAddressFromString("hx11")
	addr1owner := common.MustNewAddressFromString("hx12")
	addr2node := common.MustNewAddressFromString("hx21")
	addr2owner := common.MustNewAddressFromString("hx22")

	// add
	s.nodeOwnerCache.Add(addr1node, addr1owner)
	s.nodeOwnerCache.Add(addr2node, addr2owner)

	// get from map
	addrRes := s.nodeOwnerCache.Get(addr1node)
	fmt.Println(addrRes)
	assert.Equal(t, "hx0000000000000000000000000000000000000012", addrRes.String())
	// write in dictDB
	s.nodeOwnerCache.Flush()

	// remove all items in map
	s.nodeOwnerCache.Clear()

	// get from dictDB
	addrRes = s.nodeOwnerCache.Get(addr1node)
	assert.Equal(t, "hx0000000000000000000000000000000000000012", addrRes.String())
}
