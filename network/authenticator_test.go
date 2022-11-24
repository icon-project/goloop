package network

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/log"
)

const (
	testSecureSuite     = SecureSuiteEcdhe + 1
	testSecureAeadSuite = SecureAeadSuiteAes256Gcm + 1
)

func Test_Authenticator(t *testing.T) {
	a := newAuthenticator(walletFromGeneratedPrivateKey(), log.GlobalLogger())
	sss := []SecureSuite{
		SecureSuiteNone,
		SecureSuiteTls,
		SecureSuiteEcdhe,
		testSecureSuite,
	}
	err := a.SetSecureSuites(testChannel, sss)
	assert.NoError(t, err)
	assert.Equal(t, sss, a.GetSecureSuites(testChannel))

	for _, ss := range a.GetSecureSuites(testChannel) {
		assert.True(t, a.isSupportedSecureSuite(testChannel, ss))
	}
	assert.Equal(t, sss[0], a.resolveSecureSuite(testChannel, sss))
	assert.Equal(t, sss[1], a.resolveSecureSuite(testChannel, sss[1:]))
	assert.Equal(t, SecureSuite(SecureSuiteUnknown), a.resolveSecureSuite(testChannel, sss[:0]))

	//duplicated
	isss := []SecureSuite{SecureSuiteNone, SecureSuiteNone}
	err = a.SetSecureSuites(testChannel, isss)
	assert.Error(t, err)

	sas := []SecureAeadSuite{
		SecureAeadSuiteChaCha20Poly1305,
		SecureAeadSuiteAes128Gcm,
		SecureAeadSuiteAes256Gcm,
		testSecureAeadSuite,
	}
	err = a.SetSecureAeads(testChannel, sas)
	assert.NoError(t, err)
	assert.Equal(t, sas, a.GetSecureAeads(testChannel))

	for _, sa := range a.GetSecureAeads(testChannel) {
		assert.True(t, a.isSupportedSecureAeadSuite(testChannel, sa))
	}
	assert.Equal(t, sas[0], a.resolveSecureAeadSuite(testChannel, sas))
	assert.Equal(t, sas[1], a.resolveSecureAeadSuite(testChannel, sas[1:]))
	assert.Equal(t, SecureAeadSuite(SecureAeadSuiteNone), a.resolveSecureAeadSuite(testChannel, sas[:0]))

	//duplicated
	isas := []SecureAeadSuite{SecureAeadSuiteNone, SecureAeadSuiteNone}
	err = a.SetSecureAeads(testChannel, isas)
	assert.Error(t, err)
}
