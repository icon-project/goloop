package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
)

func TestAuthentication(t *testing.T) {

	serverWallet := wallet.New()

	request := make(map[string]interface{})
	request["address"] = serverWallet.Address()
	request["timestamp"] = common.FormatInt(time.Now().Unix())

	signingString, err := json.Marshal(&request)
	if err != nil {
		t.Error(err)
	}
	enc := base64.URLEncoding.EncodeToString(signingString)
	hash := crypto.SHASum256([]byte(enc))
	signature, err := serverWallet.Sign(hash)
	if err != nil {
		t.Error(err)
	}
	request["signature"] = base64.StdEncoding.EncodeToString(signature)
	signedString, err := json.MarshalIndent(&request, "", "\t")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("request: %s\n", string(signedString))

	var param authParam
	err = json.Unmarshal(signedString, &param)
	if err != nil {
		t.Error(err)
	}
	hash, err = param.signingHashValue()
	if err != nil {
		t.Error(err)
	}
	sign, err := param.signature()
	if err != nil {
		t.Error(err)
	}
	pubKey, err := sign.RecoverPublicKey(hash)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, sign.Verify(hash, pubKey))
	address := common.NewAccountAddressFromPublicKey(pubKey)
	assert.Equal(t, serverWallet.Address(), address, "signature verify : %s", address)
}

func TestJWTToken(t *testing.T) {
	serverWallet := wallet.New()

	claims := &tokenClaims{
		"admin",
		"0x01",
		jwt.StandardClaims{
			Issuer:    serverWallet.Address().String(),
			IssuedAt:  time.Now().Unix(),
			Audience:  serverWallet.Address().String(),
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
	}

	token := jwt.NewWithClaims(SigningMethodSECP256K1, claims)
	signedToken, err := token.SignedString(serverWallet)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("token: %s\n", signedToken)

	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return serverWallet, nil
	}

	token, err = jwt.ParseWithClaims(signedToken, claims, keyFunc)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, token.Valid, "token valid: %t", token.Valid)
}
