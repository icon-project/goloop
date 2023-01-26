package icobject

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
)

func TestFactoryOf(t *testing.T) {
	dbase := db.NewMapDB()
	fac := FactoryOf(dbase)
	assert.Nil(t, fac)

	dbase = AttachObjectFactory(dbase, testFactory)
	fac = FactoryOf(dbase)
	assert.NotNil(t, fac)
}
