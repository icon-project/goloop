package icstate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
)

func TestBlockVotersData_init(t *testing.T) {
	bvd := newBlockVotersData(nil)
	assert.Zero(t, bvd.Len())
	assert.Nil(t, bvd.Get(0))
	assert.True(t, bvd.IndexOf(common.MustNewAddressFromString("hx1")) < 0)
}