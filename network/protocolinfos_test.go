package network

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func TestProtocolInfos(t *testing.T) {
	basePI := module.ProtocolInfo(0x0000)
	duplicatedPI := module.NewProtocolInfo(basePI.ID(), basePI.Version())
	newerPI := module.NewProtocolInfo(basePI.ID(), basePI.Version()+1)
	differentPI := module.NewProtocolInfo(basePI.ID()+1, basePI.Version())
	exceptPI := module.NewProtocolInfo(differentPI.ID()+1, differentPI.Version())
	exceptPI2 := module.NewProtocolInfo(exceptPI.ID()+1, exceptPI.Version())
	piList := []module.ProtocolInfo{basePI, duplicatedPI, newerPI, differentPI}
	expectArr := []module.ProtocolInfo{basePI, newerPI, differentPI}
	expectLenOfIDSet := 2

	pis := newProtocolInfos()
	for _, pi := range piList {
		pis.Add(pi)
	}
	assert.Equal(t, len(expectArr), pis.Len())
	assert.Equal(t, expectArr, pis.Array())
	assert.Equal(t, expectLenOfIDSet, pis.LenOfIDSet())

	for _, pi := range piList {
		assert.True(t, pis.Exists(pi))
		assert.True(t, pis.ExistsByID(pi))
	}
	assert.False(t, pis.Exists(exceptPI))
	assert.False(t, pis.ExistsByID(exceptPI))

	piList1 := append(piList, exceptPI)
	expectArr1 := append(expectArr, exceptPI)
	pis1 := newProtocolInfos()
	pis1.Set(piList1)
	assert.Equal(t, expectArr1, pis1.Array())

	piList2 := append(piList, exceptPI2)
	expectArr2 := append(expectArr, exceptPI2)
	pis2 := newProtocolInfos()
	pis2.Set(piList2)
	assert.Equal(t, expectArr2, pis2.Array())

	//intersect by id, and top 1 sort by version
	resolvedArr := []module.ProtocolInfo{newerPI, differentPI}
	pis1.Resolve(pis2)
	assert.Equal(t, len(resolvedArr), pis1.Len())
	for _, pi := range resolvedArr {
		assert.True(t, pis1.Exists(pi))
	}

	//remove
	for _, pi := range piList {
		pis.Remove(pi)
		assert.False(t, pis.Exists(pi))
	}
	assert.Empty(t, pis.Array())

	for i := len(resolvedArr) - 1; i >= 0; i-- {
		pi := resolvedArr[i]
		pis1.Remove(pi)
		assert.False(t, pis1.Exists(pi))
	}
	assert.Empty(t, pis1.Array())

	shiftArr := expectArr2[1:]
	shiftArr = append(shiftArr, expectArr2[0])
	for _, pi := range shiftArr {
		pis2.Remove(pi)
		assert.False(t, pis2.Exists(pi))
	}
	assert.Empty(t, pis2.Array())
}
