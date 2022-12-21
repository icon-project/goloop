package network

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/errors"
)

type SecureConn struct {
	//*tls.Conn
	net.Conn
	sa  SecureAeadSuite
	k   *secureKey
	in  *SecureAead
	out *SecureAead
}

type SecureAead struct {
	conn   net.Conn
	secret []byte
	aead   cipher.AEAD
	nonce  []byte
}

const (
	secureConnHeaderSize = 4
	secureConnFrameSize  = 1024
)

func NewSecureConn(conn net.Conn, sa SecureAeadSuite, k *secureKey) (*SecureConn, error) {
	var inSecret, outSecret []byte
	numOfSecret := len(k.secret)
	if numOfSecret >= 2 {
		if k.isLower {
			inSecret = k.secret[0]
			outSecret = k.secret[1]
		} else {
			inSecret = k.secret[1]
			outSecret = k.secret[0]
		}
	} else if numOfSecret == 1 {
		inSecret = k.secret[0]
		outSecret = k.secret[0]
	} else {
		return nil, fmt.Errorf("secureKey has not secret")
	}

	in, err := newSecureAead(conn, sa, inSecret)
	if err != nil {
		return nil, err
	}
	out, err := newSecureAead(conn, sa, outSecret)
	if err != nil {
		return nil, err
	}
	c := &SecureConn{Conn: conn, sa: sa, k: k, in: in, out: out}
	return c, nil
}

func (c *SecureConn) Read(b []byte) (n int, err error) {
	return c.in.Read(b)
}

func (c *SecureConn) Write(b []byte) (n int, err error) {
	return c.out.Write(b)
}

func newSecureAead(conn net.Conn, sa SecureAeadSuite, secret []byte) (*SecureAead, error) {
	var aead cipher.AEAD
	var err error
	switch sa {
	case SecureAeadSuiteAes128Gcm, SecureAeadSuiteAes256Gcm:
		aes, err := aes.NewCipher(secret)
		if err != nil {
			return nil, err
		}
		aead, err = cipher.NewGCM(aes)
		if err != nil {
			return nil, err
		}
	case SecureAeadSuiteChaCha20Poly1305:
		aead, err = chacha20poly1305.New(secret)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("not supported secure aead %v", sa)
	}
	nonce := make([]byte, aead.NonceSize())
	a := &SecureAead{conn: conn, secret: secret, aead: aead, nonce: nonce}
	return a, nil
}

func (sa *SecureAead) increaseNonce() {
	for i := sa.aead.NonceSize() - 1; i >= 0; i-- {
		sa.nonce[i]++
		if sa.nonce[i] != 0 {
			return
		}
	}
}
func (sa *SecureAead) Read(b []byte) (n int, err error) {
	frame := make([]byte, secureConnFrameSize)
	_, err = io.ReadFull(sa.conn, frame[:secureConnHeaderSize])
	if err != nil {
		return
	}
	n = int(binary.BigEndian.Uint16(frame))
	sealed := make([]byte, n+sa.aead.Overhead())
	_, err = io.ReadFull(sa.conn, sealed)
	if err != nil {
		return
	}

	_, err = sa.aead.Open(frame[:0], sa.nonce, sealed[:], nil)
	if err != nil {
		return
	}
	sa.increaseNonce()

	copy(b, frame[:n])
	return
}

func (sa *SecureAead) Write(b []byte) (n int, err error) {
	wn := len(b)
	frame := make([]byte, secureConnFrameSize)
	sealed := make([]byte, secureConnHeaderSize+secureConnFrameSize+sa.aead.Overhead())
	for wn > n {
		cn := copy(frame[:secureConnFrameSize], b[n:])
		binary.BigEndian.PutUint16(sealed, uint16(cn))
		_ = sa.aead.Seal(sealed[secureConnHeaderSize:secureConnHeaderSize], sa.nonce, frame[:cn], nil)
		sa.increaseNonce()
		_, err = sa.conn.Write(sealed[:secureConnHeaderSize+cn+sa.aead.Overhead()])
		if err != nil {
			return
		}
		n += cn
	}
	return
}

type secureKey struct {
	*ecdsa.PrivateKey
	pX, pY       *big.Int
	isLower      bool
	secret       [][]byte
	extra        []byte
	sa           SecureAeadSuite
	keyLogWriter io.Writer
}

func newSecureKey(curve elliptic.Curve, keyLogWriter io.Writer) *secureKey {
	k, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic(err)
	}
	return &secureKey{PrivateKey: k, keyLogWriter: keyLogWriter}
}

func (k *secureKey) marshalPublicKey() []byte {
	return elliptic.Marshal(k.Curve, k.X, k.Y)
}
func (k *secureKey) setup(sa SecureAeadSuite, peerPublicKey []byte, defaultLower bool, numOfSecret int) error {
	k.sa = sa
	if err := k.setPeerPublicKey(peerPublicKey, defaultLower); err != nil {
		return err
	}
	return k.hkdf(numOfSecret)
}

func (k *secureKey) setPeerPublicKey(publicKey []byte, defaultLower bool) error {
	c := k.Curve
	k.pX, k.pY = elliptic.Unmarshal(c, publicKey)
	if k.pX == nil || k.pY == nil {
		return errors.New("InvalidPublicKey")
	}

	xc := k.pX.Cmp(k.X)
	yc := k.pY.Cmp(k.Y)
	if xc > 0 {
		k.isLower = true
	} else if xc == 0 {
		if yc > 0 {
			k.isLower = true
		} else if yc == 0 {
			k.isLower = defaultLower
		}
	}
	return nil
}
func (k *secureKey) hkdf(numOfSecret int) error {
	var secretLen int
	switch k.sa {
	case SecureAeadSuiteNone:
		secretLen = 32
	case SecureAeadSuiteAes128Gcm:
		secretLen = 16
	case SecureAeadSuiteAes256Gcm, SecureAeadSuiteChaCha20Poly1305:
		secretLen = 32
	default:
		return fmt.Errorf("secureKey: unknown SecureAeadSuite")
	}

	if k.pX == nil || k.pY == nil {
		return fmt.Errorf("secureKey: PeerPublicKey is nil")
	}
	c := k.Curve
	mx, _ := c.ScalarMult(k.pX, k.pY, k.D.Bytes())
	preSecret := make([]byte, (c.Params().BitSize+7)>>3)
	xBytes := mx.Bytes()
	copy(preSecret[len(preSecret)-len(xBytes):], xBytes)
	hr := hkdf.New(sha3.New256, preSecret, nil, []byte("GOLOOP_SECUREKEY_SECRET_HKDF"))
	b := make([]byte, secretLen*(numOfSecret+1))
	if _, err := io.ReadFull(hr, b[:]); err != nil {
		return err
	}
	n := 0
	k.secret = make([][]byte, numOfSecret)
	for i := 0; i < numOfSecret; i++ {
		k.secret[i] = make([]byte, secretLen)
		copy(k.secret[i], b[n:n+secretLen])
		n += secretLen
	}
	k.extra = make([]byte, secretLen)
	copy(k.extra, b[n:n+secretLen])
	if k.keyLogWriter != nil {
		s := fmt.Sprintf("SECUREKEY_HKDF %x ", k.D.Bytes())
		for _, secret := range k.secret {
			s += fmt.Sprintf("%x ", secret)
		}
		s += fmt.Sprintf("%x \n", k.extra)
		_, _ = k.keyLogWriter.Write([]byte(s))
	}
	return nil
}

func (k *secureKey) tlsConfig() (*tls.Config, error) {
	var cs uint16 = 0
	switch k.sa {
	case SecureAeadSuiteAes128Gcm:
		cs = tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	case SecureAeadSuiteAes256Gcm:
		cs = tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	case SecureAeadSuiteChaCha20Poly1305:
		cs = tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305
	default:
		return nil, fmt.Errorf("secureKey: unknown SecureAeadSuite")
	}
	cert, err := k.selfCertificate("")
	if err != nil {
		return nil, err
	}
	var curve tls.CurveID
	switch k.Curve {
	case elliptic.P256():
		curve = tls.CurveP256
	case elliptic.P384():
		curve = tls.CurveP384
	case elliptic.P521():
		curve = tls.CurveP521
	default:
		return nil, fmt.Errorf("secureKey: unknown Curve")
	}

	config := &tls.Config{
		InsecureSkipVerify:    true,
		Certificates:          []tls.Certificate{cert},
		ClientAuth:            tls.RequireAnyClientCert,
		CipherSuites:          []uint16{cs},
		CurvePreferences:      []tls.CurveID{curve},
		VerifyPeerCertificate: k.verifyCertificate,
		KeyLogWriter:          k.keyLogWriter,
	}
	return config, nil
}

func (k *secureKey) selfCertificate(commonName string) (tls.Certificate, error) {
	cur := time.Now()
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"IconLoop"},
			CommonName:   commonName,
		},
		NotBefore: cur,
		NotAfter:  cur.Add(365 * 24 * time.Hour),

		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	b, err := x509.CreateCertificate(rand.Reader, &template, &template, k.Public(), k.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert := tls.Certificate{}
	cert.Certificate = append(cert.Certificate, b)
	cert.PrivateKey = k.PrivateKey

	cert.Leaf, err = x509.ParseCertificate(b)
	return cert, err
}

func (k *secureKey) verifyCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	cp := x509.NewCertPool()
	certs := make([]*x509.Certificate, 0)
	opts := x509.VerifyOptions{
		Roots:       cp,
		CurrentTime: time.Now(),
		//DNSName:       "commonName",
		//Intermediates: x509.NewCertPool(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	for _, rc := range rawCerts {
		cert, err := x509.ParseCertificate(rc)
		if err != nil {
			return err
		}
		certs = append(certs, cert)
		cp.AddCert(cert)
	}
	for _, cert := range certs {
		pubK := cert.PublicKey.(*ecdsa.PublicKey)
		if pubK.X.Cmp(k.pX) != 0 || pubK.Y.Cmp(k.pY) != 0 {
			return fmt.Errorf("secureKey: mismatch certificate public key")
		}
		if _, err := cert.Verify(opts); err != nil {
			return err
		}
	}
	return nil
}

type SecureSuite byte

const (
	SecureSuiteUnknown = iota
	SecureSuiteNone
	SecureSuiteTls
	SecureSuiteEcdhe
)

func (s SecureSuite) String() string {
	switch s {
	case SecureSuiteNone:
		return "none"
	case SecureSuiteTls:
		return "tls"
	case SecureSuiteEcdhe:
		return "ecdhe"
	default:
		return "unknown"
	}
}

func SecureSuiteFromString(s string) SecureSuite {
	switch s {
	case "none":
		return SecureSuiteNone
	case "tls":
		return SecureSuiteTls
	case "ecdhe":
		return SecureSuiteEcdhe
	default:
		return SecureSuiteUnknown
	}
}

type SecureAeadSuite byte

const (
	SecureAeadSuiteNone = iota
	SecureAeadSuiteChaCha20Poly1305
	SecureAeadSuiteAes128Gcm
	SecureAeadSuiteAes256Gcm
)

func (s SecureAeadSuite) String() string {
	switch s {
	case SecureAeadSuiteChaCha20Poly1305:
		return "chacha"
	case SecureAeadSuiteAes128Gcm:
		return "aes128"
	case SecureAeadSuiteAes256Gcm:
		return "aes256"
	default:
		return "unknown"
	}
}

func SecureAeadSuiteFromString(s string) SecureAeadSuite {
	switch s {
	case "chacha":
		return SecureAeadSuiteChaCha20Poly1305
	case "aes128":
		return SecureAeadSuiteAes128Gcm
	case "aes256":
		return SecureAeadSuiteAes256Gcm
	default:
		return SecureAeadSuiteNone
	}
}

type SecureError string

const (
	SecureErrorNone    = ""
	SecureErrorInvalid = "invalid"
)
