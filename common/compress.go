package common

import (
	"bytes"
	"io"

	"github.com/icon-project/goloop/common/lzw"
)

func Compress(bs []byte) []byte {
	if len(bs) == 0 {
		return []byte{}
	}
	wb := bytes.NewBuffer(nil)
	fd := lzw.NewWriter(wb, lzw.MSB, 8)
	_, _ = fd.Write(bs)
	_ = fd.Close()
	return wb.Bytes()
}

func Decompress(bs []byte) []byte {
	if len(bs) == 0 {
		return []byte{}
	}
	wb := bytes.NewBuffer(bs)
	fd := lzw.NewReader(wb, lzw.MSB, 8)
	c, _ := io.ReadAll(fd)
	_ = fd.Close()
	return c
}
