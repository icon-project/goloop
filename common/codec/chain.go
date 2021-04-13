package codec

import (
	"io"
)

func Marshal(w io.Writer, v interface{}) error {
	return BC.Marshal(w, v)
}

func Unmarshal(r io.Reader, v interface{}) error {
	return BC.Unmarshal(r, v)
}

func NewSimpleEncoder(w io.Writer) EncodeAndCloser {
	return BC.NewEncoder(w)
}

func NewSimpleDecoder(r io.Reader) DecodeAndCloser {
	return BC.NewDecoder(r)
}

func NewEncoderBytes(b *[]byte) EncodeAndCloser {
	return BC.NewEncoderBytes(b)
}

func MarshalToBytes(v interface{}) ([]byte, error) {
	return BC.MarshalToBytes(v)
}

func MustMarshalToBytes(v interface{}) []byte {
	return BC.MustMarshalToBytes(v)
}

func MustUnmarshalFromBytes(b []byte, v interface{}) []byte {
	return BC.MustUnmarshalFromBytes(b, v)
}
