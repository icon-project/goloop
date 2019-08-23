package sync

import "fmt"

type protocolInfo uint16

func (pi protocolInfo) ID() byte {
	return byte(pi >> 8)
}

func (pi protocolInfo) Version() byte {
	return byte(pi)
}

func (pi protocolInfo) String() string {
	return fmt.Sprintf("%04x", pi.Uint16())
}

func (pi protocolInfo) Copy(b []byte) {
	b[0] = pi.ID()
	b[1] = pi.Version()
}

func (pi protocolInfo) Uint16() uint16 {
	return uint16(pi)
}
