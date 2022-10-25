package crypto

import (
	"encoding/hex"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
)

var (
	testHash             = SHA3Sum256([]byte("icx_sendTransaction.fee.0x2386f26fc10000.from.hx57b8365292c115d3b72d948272cc4d788fa91f64.timestamp.1538976759263551.to.hx57b8365292c115d3b72d948272cc4d788fa91f64.value.0xde0b6b3a7640000"))
	testPrivateKey, _    = hex.DecodeString("ca158b1d3c81c492e7785a3bba6aa755e07c28d2711811e7014bcf911ea2643b")
	testPublicKey, _     = hex.DecodeString("0448250ebe88d77e0a12bcf530fe6a2cf1ac176945638d309b840d631940c93b78c2bd6d16f227a8877e3f1604cd75b9c5a8ab0cac95174a8a0a0f8ea9e4c10bca")
	testPublicKeyComp, _ = hex.DecodeString("0248250ebe88d77e0a12bcf530fe6a2cf1ac176945638d309b840d631940c93b78")
	testSignature, _     = hex.DecodeString("4011de30c04302a2352400df3d1459d6d8799580dceb259f45db1d99243a8d0c64f548b7776cb93e37579b830fc3efce41e12e0958cda9f8c5fcad682c61079500")
)

// TODO add performance test
func TestSignAndVerify(t *testing.T) {
	priv, pub := GenerateKeyPair()
	sig, err := NewSignature(testHash, priv)
	if err != nil {
		t.Errorf("error signing:%s", err)
		return
	}

	if !sig.Verify(testHash, pub) {
		t.Errorf("Verify failed")
	}

	hash := make([]byte, len(testHash))
	copy(hash, testHash)
	hash[0] ^= 0xff
	if sig.Verify(hash, pub) {
		t.Errorf("Verify always works!")
	}

	// test signature without V
	rs, err := sig.SerializeRS()
	assert.NoError(t, err)

	sig2, err := ParseSignature(rs)
	assert.NoError(t, err)

	assert.True(t, sig2.Verify(testHash, pub))

	// forged signature
	rs[0] ^= 0xff
	sig3, err := ParseSignature(rs)
	assert.NoError(t, err)

	assert.False(t, sig3.Verify(testHash, pub))
}

func TestVerifySignature(t *testing.T) {
	sig, err := ParseSignature(testSignature)
	assert.NoError(t, err)
	pub, err := ParsePublicKey(testPublicKey)
	assert.NoError(t, err)
	result := sig.Verify(testHash, pub)
	assert.True(t, result, "Verify failed")
}

func TestRecoverPublicKey(t *testing.T) {
	sig, err := ParseSignature(testSignature)
	assert.NoError(t, err)
	pub, err := ParsePublicKey(testPublicKey)
	assert.NoError(t, err)
	pk2, err := sig.RecoverPublicKey(testHash)
	assert.NoError(t, err)
	assert.True(t, pub.Equal(pk2))

	pk3, err := sig.RecoverPublicKey(nil)
	assert.Error(t, err)
	assert.Nil(t, pk3)

	rs, err := sig.SerializeRS()
	assert.NoError(t, err)
	sig2, err := ParseSignature(rs)
	pk4, err := sig2.RecoverPublicKey(testHash)
	assert.Error(t, err)
	assert.Nil(t, pk4)
}

func TestRecoverPublicKeyAfterSign(t *testing.T) {
	priv, pub := GenerateKeyPair()
	sig, err := NewSignature(testHash, priv)
	assert.NoError(t, err, "fail to make signature")

	pub1, err := sig.RecoverPublicKey(testHash)
	assert.NoError(t, err, "fail to recover public key")
	assert.True(t, pub1.Equal(pub), "recovered public key is not same")

	// making invalid signature
	rsv, err := sig.SerializeRSV()
	assert.NoError(t, err, "fail on SerializeRS()")
	rsv[1] = rsv[1] ^ 0x0f

	// ensure that it fails with invalid signature
	sig2, err := ParseSignature(rsv)
	assert.NoError(t, err)
	pub2, err := sig2.RecoverPublicKey(testHash)
	if err != nil {
		assert.Nil(t, pub2)
	} else {
		assert.False(t, pub2.Equal(pub))
	}

	sig3, err := ParseSignature(rsv[:SignatureLenRaw])
	assert.NoError(t, err)
	result := sig3.Verify(testHash, pub)
	assert.False(t, result)
}

func TestPrintSignature(t *testing.T) {
	sig, err := ParseSignature(testSignature)
	assert.NoError(t, err)

	str := "0x" + hex.EncodeToString(testSignature)
	assert.Equal(t, str, sig.String())

	sig, _ = ParseSignature(testSignature[:64])
	str = "0x" + hex.EncodeToString(testSignature[:64]) + "[no V]"
	assert.Equal(t, str, sig.String())

	sig, _ = ParseSignature([]byte("invalid"))
	str = "[empty]"
	assert.Equal(t, str, sig.String())
}

func TestRace(t *testing.T) {
	const SubRoutineCount = 4
	const RepeatCount = 5

	var lock sync.Mutex
	cond := sync.NewCond(&lock)

	var readyWG sync.WaitGroup
	readyWG.Add(SubRoutineCount)
	wait := func() {
		lock.Lock()
		defer lock.Unlock()
		readyWG.Done()
		cond.Wait()
	}

	startAll := func() {
		lock.Lock()
		defer lock.Unlock()
		cond.Broadcast()
	}

	var finishWG sync.WaitGroup
	subRoutine := func(idx int) {
		wait()
		for i := 0; i < RepeatCount; i++ {
			priv, pub := GenerateKeyPair()
			sig, err := NewSignature(testHash, priv)

			pub1, err := sig.RecoverPublicKey(testHash)
			if err != nil {
				t.Errorf("error recover public key:%s", err)
				return
			}

			if !pub.Equal(pub1) {
				t.Errorf("recovered public key is not same")
			}
			r := sig.Verify(testHash, pub)
			assert.True(t, r)
			delay := time.Millisecond * time.Duration(rand.Intn(10))
			time.Sleep(delay)
		}
		finishWG.Done()
	}

	// start subroutines
	finishWG.Add(SubRoutineCount)
	for i := 0; i < SubRoutineCount; i++ {
		go subRoutine(i)
	}

	// wait until the subroutines reach wait()
	readyWG.Wait()
	time.Sleep(10 * time.Millisecond)

	// start subroutines
	startAll()

	// wait for DONE
	finishWG.Wait()
}

func TestSignature_RLPEncodeSelf(t *testing.T) {
	priv, pub := GenerateKeyPair()
	sig, err := NewSignature(testHash, priv)
	assert.NoError(t, err)

	bs := codec.MustMarshalToBytes(&sig)
	var sig2 Signature
	codec.MustUnmarshalFromBytes(bs, &sig2)
	rpub, err := sig.RecoverPublicKey(testHash)
	assert.NoError(t, err)
	rpub2, err := sig2.RecoverPublicKey(testHash)
	assert.NoError(t, err)

	assert.EqualValues(t, pub.SerializeCompressed(), rpub.SerializeCompressed())
	assert.EqualValues(t, rpub.SerializeCompressed(), rpub2.SerializeCompressed())
}

func TestSignature_RLPEncodeSelf_nil(t *testing.T) {
	var psig *Signature
	bs := codec.MustMarshalToBytes(psig)
	var psig2 *Signature
	codec.MustUnmarshalFromBytes(bs, &psig2)
	assert.Nil(t, psig2)
}
