package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type SignedMethodSECP256K1 struct {
	Name string
}

var SigningMethodSECP256K1 *SignedMethodSECP256K1

func init() {
	// Wallet
	SigningMethodSECP256K1 = &SignedMethodSECP256K1{"SECP256K1"}
	jwt.RegisterSigningMethod(SigningMethodSECP256K1.Alg(), func() jwt.SigningMethod {
		return SigningMethodSECP256K1
	})
}

func (m *SignedMethodSECP256K1) Alg() string {
	return m.Name
}

func (m *SignedMethodSECP256K1) Sign(signingString string, key interface{}) (string, error) {
	if wallet, ok := key.(module.Wallet); ok {
		hashValue := crypto.SHASum256([]byte(signingString))
		sign, err := wallet.Sign(hashValue)
		if err != nil {
			return "", err
		}
		return jwt.EncodeSegment(sign), nil
	}
	return "", jwt.ErrInvalidKeyType
}

func (m *SignedMethodSECP256K1) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = jwt.DecodeSegment(signature); err != nil {
		return err
	}
	sign, err := crypto.ParseSignature(sig)
	if err != nil {
		return err
	}

	hashValue := crypto.SHASum256([]byte(signingString))

	// Get the key
	var wallet module.Wallet
	switch k := key.(type) {
	case module.Wallet:
		wallet = k
	default:
		return jwt.ErrInvalidKeyType
	}

	pubKey, err := crypto.ParsePublicKey(wallet.PublicKey())
	if err != nil {
		return err
	}

	if !sign.Verify(hashValue, pubKey) {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

type tokenClaims struct {
	Role   string `json:"role"`
	Chains string `json:"chains"`
	jwt.StandardClaims
}

func JWTConfig(wallet module.Wallet) middleware.JWTConfig {
	config := middleware.JWTConfig{
		Claims:        &tokenClaims{},
		SigningKey:    wallet,
		SigningMethod: "SECP256K1",
		ContextKey:    "token",
	}
	return config
}

type authParam struct {
	Address   jsonrpc.Address `json:"address" validate:"required,t_addr_eoa"`
	Timestamp jsonrpc.HexInt  `json:"timestamp" validate:"required,t_int"`
	Signature string          `json:"signature" validate:"required,t_sig"`
}

func (p *authParam) signature() (*crypto.Signature, error) {
	var err error
	var sig []byte
	if sig, err = base64.StdEncoding.DecodeString(p.Signature); err != nil {
		return nil, err
	}
	sign, err := crypto.ParseSignature(sig)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

func (p *authParam) signingHashValue() ([]byte, error) {
	signingParam := make(map[string]interface{})
	signingParam["address"] = p.Address
	signingParam["timestamp"] = p.Timestamp
	signingString, err := json.Marshal(signingParam)
	if err != nil {
		return nil, err
	}
	encodeString := base64.URLEncoding.EncodeToString(signingString)
	return crypto.SHASum256([]byte(encodeString)), nil
}

func authentication(wallet module.Wallet) echo.HandlerFunc {
	return func(c echo.Context) error {
		params := new(authParam)
		if err := c.Bind(params); err != nil {
			return echo.ErrBadRequest
		}
		if err := c.Validate(params); err != nil {
			return echo.ErrBadRequest
		}
		hashValue, err := params.signingHashValue()
		if err != nil {
			return err
		}
		signature, err := params.signature()
		if err != nil {
			return err
		}
		pubKey, err := signature.RecoverPublicKey(hashValue)
		if err != nil {
			return err
		}

		// if signature.Verify(hashValue, pubKey)
		addr := common.NewAccountAddressFromPublicKey(pubKey)
		if addr.Equal(params.Address.Address()) {
			signedToken, err := newToken(wallet, params.Address.Address().String())
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, map[string]string{
				"token": signedToken,
			})
		} else {
			return echo.ErrUnauthorized
		}
	}
}

func newToken(wallet module.Wallet, audience string) (string, error) {
	// TODO : role, chains
	claims := &tokenClaims{
		"admin",
		"0x01",
		jwt.StandardClaims{
			Issuer:    wallet.Address().String(),
			IssuedAt:  time.Now().Unix(),
			Audience:  audience,
			ExpiresAt: time.Now().Add(time.Minute * 10).Unix(),
		},
	}
	token := jwt.NewWithClaims(SigningMethodSECP256K1, claims)
	return token.SignedString(wallet)
}
