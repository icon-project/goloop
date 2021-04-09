package txresult

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/icon-project/goloop/common"
)

func TestLogsBloom_AddLog(t *testing.T) {
	lb1 := NewLogsBloom(nil)

	l1 := lb1.LogBytes()

	lb1.AddLog(common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	l2 := lb1.LogBytes()

	if bytes.Equal(l1, l2) {
		t.Error("Log bloom data should be different from original")
	}
}

func TestLogsBloom_Initial(t *testing.T) {
	lb1 := NewLogsBloom(nil)
	if len(lb1.Bytes()) != 0 {
		t.Error("New log bloom must have empty bytes")
	}
	if len(lb1.CompressedBytes()) != 0 {
		t.Error("New log bloom must have empty bytes")
	}
}

func TestLogsBloom_MergeContains(t *testing.T) {
	lb1 := NewLogsBloom(nil)
	lb1.AddLog(common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	lb2 := NewLogsBloom(nil)
	lb2.AddLog(common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x04},
	})

	lb3 := NewLogsBloom(nil)
	lb3.Merge(lb1)
	lb3.Merge(lb2)

	if !lb3.Contain(lb1) {
		t.Error("Merge log bloom should contain merged one")
	}

	lb4 := NewLogsBloom(nil)
	lb4.AddLog(common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x05},
	})

	if lb3.Contain(lb4) {
		t.Error("Unmerged log bloom should not contain it")
	}

	lb5 := NewLogsBloom(nil)
	if !lb3.Contain(lb5) {
		t.Error("Empty logsbloom should be contained by all of them")
	}

	if lb1.Contain(lb3) {
		t.Error("Shouldn't contain smaller one")
	}
}

func TestLogsBloom_Bytes(t *testing.T) {
	lb1 := NewLogsBloom(nil)
	lb1.AddLog(common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"), [][]byte{
		{0x01, 0x02, 0x03},
	})

	lb2 := NewLogsBloom(lb1.Bytes())
	if !bytes.Equal(lb2.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}

	if !bytes.Equal(lb2.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}
}

func TestLogsBloom_Compressed(t *testing.T) {
	lb1 := NewLogsBloom(nil)
	lb1.AddLog(
		common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"),
		[][]byte{[]byte("TestFunc(int)"), []byte{0x12, 0x23}},
	)
	lb1.AddLog(
		common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"),
		[][]byte{[]byte("TestFunc2(int)"), []byte{0x13, 0x45}},
	)

	lb2 := NewLogsBloom(lb1.Bytes())

	if !bytes.Equal(lb2.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}
	if !bytes.Equal(lb2.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}

	fmt.Println("Original:", len(lb2.Bytes()),
		"Compressed:", len(lb2.CompressedBytes()))

	lb3 := NewLogsBloom(nil)
	lb3.SetCompressedBytes(lb2.CompressedBytes())

	if !bytes.Equal(lb3.LogBytes(), lb1.LogBytes()) {
		t.Error("Deserialized one should be same as origin")
	}
	if !bytes.Equal(lb3.Bytes(), lb1.Bytes()) {
		t.Error("Deserialized one should be same as origin")
	}
}

func BenchmarkLogsBloom_Compressed(b *testing.B) {
	b.StopTimer()
	lb1 := NewLogsBloom(nil)
	lb1.AddLog(
		common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"),
		[][]byte{[]byte("TestFunc(int)"), []byte{0x12, 0x23}},
	)
	lb1.AddLog(
		common.MustNewAddressFromString("cx0000000000000000000000000000000000000000"),
		[][]byte{[]byte("TestFunc2(int)"), []byte{0x13, 0x45}},
	)
	lb2 := NewLogsBloom(nil)
	b.Run("CompressAndDecompress", func(b *testing.B) {
		lb2.SetCompressedBytes(lb1.CompressedBytes())
	})
}
