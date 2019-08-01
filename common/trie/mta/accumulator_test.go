package mta

import (
	"testing"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/stretchr/testify/assert"
)

func TestMTAccumulator_Basic(t *testing.T) {
	mdb := db.NewMapDB()
	bk, _ := mdb.GetBucket("")

	a := &Accumulator{
		KeyForState: []byte("a"),
		Bucket:      bk,
	}
	if a.Len() != 0 {
		t.Errorf("Length is not zero")
	}
	data := []string{
		"dog", "cat", "elephant", "bird", "monkey", "lion", "tiger",
	}
	for _, d := range data {
		bs := []byte(d)
		w := a.AddData(bs)
		if err := a.Verify(w, crypto.SHA3Sum256(bs)); err != nil {
			t.Logf("Accumulator:%s", a)
			t.Errorf("Fail to verify w=%s err=%s", w, err)
		}
	}
	assert.Equal(t, int64(len(data)), a.Len())

	for i, d := range data {
		w, err := a.WitnessFor(int64(i))
		assert.Equal(t, nil, err)
		hs := WitnessesToHashes(w)
		w = HashesToWitness(hs, int64(i))
		if err := a.Verify(w, crypto.SHA3Sum256([]byte(d))); err != nil {
			t.Error("Fail to verify")
		}
	}

	a.Flush()

	a = &Accumulator{
		KeyForState: []byte("a"),
		Bucket:      bk,
	}
	a.Recover()

	for i, d := range data {
		w, err := a.WitnessFor(int64(i))
		assert.Equal(t, nil, err)
		hs := WitnessesToHashes(w)
		w = HashesToWitness(hs, int64(i))
		if err := a.Verify(w, crypto.SHA3Sum256([]byte(d))); err != nil {
			t.Error("Fail to verify")
		}
	}

	t.Logf("%s", a)
}
