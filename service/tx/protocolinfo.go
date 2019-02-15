package tx

import (
	"encoding/binary"
	"fmt"
)

type protocolInfo uint16

func (pi protocolInfo) ID() byte {
	return byte(pi >> 8)
}

func (pi protocolInfo) Version() byte {
	return byte(pi)
}

func (pi protocolInfo) Copy(b []byte) {
	binary.BigEndian.PutUint16(b[:2], uint16(pi))
}

func (pi protocolInfo) String() string {
	return fmt.Sprintf("{ID:SERVICE:%#02x,Ver:%#02x}", pi.ID(), pi.Version())
}

func (pi protocolInfo) Uint16() uint16 {
	return uint16(pi)
}
