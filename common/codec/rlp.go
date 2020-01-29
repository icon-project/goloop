package codec

import (
	"io"

	"github.com/icon-project/goloop/common/rlp"
)

var rlpCodecObject rlpCodec
var RLP = bytesWrapper{&rlpCodecObject}

type rlpCodec struct{}

func (*rlpCodec) NewEncoder(w io.Writer) Encoder {
	return rlp.NewEncoder(w)
}

func (*rlpCodec) NewDecoder(r io.Reader) Decoder {
	return rlp.NewDecoder(r)
}
