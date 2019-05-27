package txresult

import (
	"bytes"
	"testing"

	"github.com/icon-project/goloop/common"
)

func TestLogBloom_AddLog(t *testing.T) {
	lb1 := NewLogBloom(nil)

	l1 := lb1.LogBytes()

	lb1.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	l2 := lb1.LogBytes()

	if bytes.Equal(l1, l2) {
		t.Error("Log bloom data should be different from original")
	}
}

func TestLogBloom_Initial(t *testing.T) {
	lb1 := NewLogBloom(nil)
	if len(lb1.Bytes()) != 0 {
		t.Error("New log bloom must have empty bytes")
	}
	if len(lb1.CompressedBytes()) != 0 {
		t.Error("New log bloom must have empty bytes")
	}
}

func TestLogBloom_MergeContains(t *testing.T) {
	lb1 := NewLogBloom(nil)
	lb1.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	lb2 := NewLogBloom(nil)
	lb2.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x04},
	})

	lb3 := NewLogBloom(nil)
	lb3.Merge(lb1)
	lb3.Merge(lb2)

	if !lb3.Contain(lb1) {
		t.Error("Merge log bloom should contain merged one")
	}

	lb4 := NewLogBloom(nil)
	lb4.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x05},
	})

	if lb3.Contain(lb4) {
		t.Error("Unmerged log bloom should not contain it")
	}

	lb5 := NewLogBloom(nil)
	if !lb3.Contain(lb5) {
		t.Error("Empty logbloom should be contained by all of them")
	}

	if lb1.Contain(lb3) {
		t.Error("Shouldn't contain smaller one")
	}
}

func TestLogBloom_Bytes(t *testing.T) {
	lb1 := NewLogBloom(nil)
	lb1.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	lb2 := NewLogBloom(lb1.Bytes())
	if !bytes.Equal(lb2.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}

	if !bytes.Equal(lb2.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}
}

func TestLogBloom_Compressed(t *testing.T) {
	lb1 := NewLogBloom(nil)
	lb1.AddLog(common.NewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	lb2 := NewLogBloom(lb1.Bytes())

	if !bytes.Equal(lb2.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}
	if !bytes.Equal(lb2.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}

	lb3 := NewLogBloom(nil)
	lb3.SetCompressedBytes(lb2.CompressedBytes())

	if !bytes.Equal(lb3.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}
	if !bytes.Equal(lb3.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}
}
