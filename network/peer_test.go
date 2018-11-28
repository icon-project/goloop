package network

import (
	"log"
	"testing"
	"time"
)

func Test_peer_PeerRTT(t *testing.T) {
	r := NewPeerRTT()
	r.Start()
	time.Sleep(100 * time.Millisecond)
	r.Stop()
	log.Println(r.Last(time.Millisecond))
	log.Println(r.Avg(time.Millisecond))
}
