package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	coinTypeICON    = "icx"
	cipherAES128CTR = "aes-128-ctr"
	kdfScrypt       = "scrypt"
)

type AES128CTRParams struct {
	IV common.RawHexBytes `json:"iv"`
}

type ScryptParams struct {
	DKLen int                `json:"dklen"`
	N     int                `json:"n"`
	R     int                `json:"r"`
	P     int                `json:"p"`
	Salt  common.RawHexBytes `json:"salt"`
}

func (p *ScryptParams) Init() error {
	salt := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	p.DKLen = 32
	p.P = 1
	p.R = 8
	p.N = 1 << 16
	p.Salt = salt
	return nil
}

func (p *ScryptParams) Key(pw []byte) ([]byte, error) {
	return scrypt.Key(pw, p.Salt.Bytes(), p.N, p.R, p.P, p.DKLen)
}

type CryptoData struct {
	Cipher       string             `json:"cipher"`
	CipherParams json.RawMessage    `json:"cipherparams"`
	CipherText   common.RawHexBytes `json:"ciphertext"`
	KDF          string             `json:"kdf"`
	KDFParams    json.RawMessage    `json:"kdfparams"`
	MAC          common.RawHexBytes `json:"mac"`
}

type KeyStoreData struct {
	Address  common.Address `json:"address"`
	ID       string         `json:"id"`
	Version  int            `json:"version"`
	CoinType string         `json:"coinType"`
	Crypto   CryptoData     `json:"crypto"`
}

func SHA3SumKeccak256(data ...[]byte) []byte {
	s := sha3.NewLegacyKeccak256()
	for _, d := range data {
		s.Write(d)
	}
	return s.Sum([]byte{})
}

func EncryptKeyAsKeyStore(s *crypto.PrivateKey, pw []byte) ([]byte, error) {
	var ks KeyStoreData
	var c AES128CTRParams
	var k ScryptParams

	if err := k.Init(); err != nil {
		return nil, err
	}
	key, err := k.Key(pw)
	if err != nil {
		return nil, err
	}
	ks.Crypto.KDF = kdfScrypt
	ks.Crypto.KDFParams, err = json.Marshal(&k)
	if err != nil {
		return nil, err
	}

	b, err := aes.NewCipher(key[0:16])
	if err != nil {
		return nil, err
	}
	c.IV = make([]byte, b.BlockSize())
	_, err = io.ReadFull(rand.Reader, c.IV)
	if err != nil {
		return nil, err
	}
	secret := s.Bytes()
	cipherText := make([]byte, len(secret))
	enc := cipher.NewCTR(b, c.IV)
	enc.XORKeyStream(cipherText, secret)

	ks.Crypto.Cipher = cipherAES128CTR
	ks.Crypto.CipherParams, err = json.Marshal(&c)
	if err != nil {
		return nil, err
	}
	ks.Crypto.CipherText = cipherText
	ks.Crypto.MAC = SHA3SumKeccak256(key[16:32], cipherText)
	ks.Version = 3
	ks.CoinType = coinTypeICON
	ks.ID = uuid.Must(uuid.NewV4()).String()
	if addr := common.NewAccountAddressFromPublicKey(s.PublicKey()); addr == nil {
		return nil, errors.New("FailToMakeAddressForTheKey")
	} else {
		ks.Address.Set(addr)
	}

	return json.Marshal(&ks)
}

func DecryptKeyStore(data, pw []byte) (*crypto.PrivateKey, error) {
	var ksData KeyStoreData
	if err := json.Unmarshal(data, &ksData); err != nil {
		return nil, err
	}
	if ksData.CoinType != coinTypeICON {
		return nil, errors.Errorf("InvalidCoinType(coin=%s)", ksData.CoinType)
	}

	if ksData.Crypto.Cipher != cipherAES128CTR {
		return nil, errors.Errorf("UnsupportedCipher(cipher=%s)",
			ksData.Crypto.Cipher)
	}
	var cipherParams AES128CTRParams
	if err := json.Unmarshal(ksData.Crypto.CipherParams, &cipherParams); err != nil {
		return nil, err
	}

	if ksData.Crypto.KDF != kdfScrypt {
		return nil, errors.Errorf("UnsupportedKDF(kdf=%s)", ksData.Crypto.KDF)
	}
	var kdfParams ScryptParams
	if err := json.Unmarshal(ksData.Crypto.KDFParams, &kdfParams); err != nil {
		return nil, err
	}

	key, err := kdfParams.Key(pw)
	if err != nil {
		return nil, err
	}

	cipheredBytes := ksData.Crypto.CipherText.Bytes()

	s := sha3.NewLegacyKeccak256()
	s.Write(key[16:32])
	s.Write(cipheredBytes)
	mac := s.Sum([]byte{})
	if !bytes.Equal(mac, ksData.Crypto.MAC.Bytes()) {
		return nil, errors.Errorf("InvalidPassword")
	}

	block, err := aes.NewCipher(key[0:16])
	if err != nil {
		return nil, err
	}

	secretBytes := make([]byte, len(cipheredBytes))

	stream := cipher.NewCTR(block, cipherParams.IV.Bytes())
	stream.XORKeyStream(secretBytes, cipheredBytes)

	secret, err := crypto.ParsePrivateKey(secretBytes)
	if err != nil {
		return nil, err
	}
	public := secret.PublicKey()
	address := common.NewAccountAddressFromPublicKey(public)
	if !address.Equal(&ksData.Address) {
		log.Warnf("Recovered address %s != keyStore address %s",
			address.String(), ksData.Address.String())
	}
	return secret, nil
}

func ReadAddressFromKeyStore(data []byte) (module.Address, error) {
	var ksData KeyStoreData
	if err := json.Unmarshal(data, &ksData); err != nil {
		return nil, err
	}
	if ksData.CoinType != coinTypeICON {
		return nil, errors.Errorf("InvalidCoinType(coin=%s)", ksData.CoinType)
	}
	return &ksData.Address, nil
}

func NewFromKeyStore(data, pw []byte) (module.Wallet, error) {
	secret, err := DecryptKeyStore(data, pw)
	if err != nil {
		return nil, err
	}
	return NewFromPrivateKey(secret)
}

func KeyStoreFromWallet(w module.Wallet, pw []byte) ([]byte, error) {
	s, ok := w.(*softwareWallet)
	if ok {
		return EncryptKeyAsKeyStore(s.skey, pw)
	} else {
		return nil, nil
	}
}
