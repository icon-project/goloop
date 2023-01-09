package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

func Test_PeerDispatcher(t *testing.T) {
	logger := testLogger()
	connInfo := func(in bool, i int) string {
		return fmt.Sprintf("in:%v,idx:%d", in, i)
	}
	onPeerFunc := func(ph *testPeerHandler, ch chan string, i int) peerFunc {
		return func(p *Peer) {
			ch <- connInfo(p.In(), i)
			ph.nextOnPeer(p)
		}
	}
	id := generatePeerID()

	var listenPHs []PeerHandler
	var dialPHs []PeerHandler
	numOfPeerHandler := 2
	listenCh := make(chan string, numOfPeerHandler+1)
	dialCh := make(chan string, numOfPeerHandler+1)
	for i := 0; i < numOfPeerHandler; i++ {
		ph := newTestPeerHandler(id, logger)
		ph.onPeerFunc = onPeerFunc(ph, listenCh, i)
		listenPHs = append(listenPHs, ph)

		ph = newTestPeerHandler(id, logger)
		ph.onPeerFunc = onPeerFunc(ph, dialCh, i)
		dialPHs = append(dialPHs, ph)
	}
	listenPD := newPeerDispatcher(id, logger, listenPHs...)
	l := newListener(getAvailableLocalhostAddress(t), listenPD.onAccept, logger)
	if err := l.Listen(); err != nil {
		assert.FailNow(t, err.Error())
	}
	pi := module.ProtocolInfo(0x0000)
	listenPH := newTestPeerHandler(id, logger)
	listenPH.onPeerFunc = func(p *Peer) {}
	listenPH.onPacketFunc = func(pkt *Packet, p *Peer) {
		listenPH.logger.Traceln("listenPH.onPacket", pkt, p)
		var channel string
		listenPH.decodePeerPacket(p, &channel, pkt)
		p.setChannel(channel)
		listenPH.nextOnPeer(p)
	}
	listenPD.registerPeerHandler(listenPH, false)

	dialPD := newPeerDispatcher(id, logger)
	dialPH := newTestPeerHandler(id, logger)
	dialPH.onPeerFunc = func(p *Peer) {
		dialPH.logger.Traceln("dialPH.onPeer", p)
		dialPH.sendMessage(pi, pi, p.Channel(), p)
		dialPH.nextOnPeer(p)
	}
	dialPD.registerPeerHandler(dialPH, false)
	for _, ph := range dialPHs {
		dialPD.registerPeerHandler(ph, true)
	}

	if err := newDialer(testChannel, dialPD.onConnect).Dial(l.address); err != nil {
		assert.FailNow(t, err.Error())
	}
	assertPeerHandler := func(n int) {
		for i := 0; i < n; i++ {
			assert.Equal(t, connInfo(true, i), <-listenCh)
			assert.Equal(t, connInfo(false, i), <-dialCh)
		}
	}
	assertPeerHandler(numOfPeerHandler)

	channelFunc := func(i int) string {
		return fmt.Sprintf("%s_%d", testChannel, i)
	}
	channelConnInfo := func(close, in bool, i int) string {
		return fmt.Sprintf("in:%v,idx:%d,ch:%s", in, i, channelFunc(i))
	}
	channelOnPeerFunc := func(ph *testPeerHandler, ch chan string, i int) peerFunc {
		return func(p *Peer) {
			ch <- channelConnInfo(false, p.In(), i)
		}
	}
	channelOnCloseFunc := func(ph *testPeerHandler, ch chan string, i int) peerFunc {
		return func(p *Peer) {
			ch <- channelConnInfo(true, p.In(), i)
		}
	}

	numOfChannel := 2
	for i := 0; i < numOfChannel; i++ {
		mtr := metric.NewNetworkMetric(metric.GetMetricContextByCID(i))

		ph := newTestPeerHandler(id, logger)
		ph.onPeerFunc = channelOnPeerFunc(ph, listenCh, i)
		ph.onCloseFunc = channelOnCloseFunc(ph, listenCh, i)
		listenPD.registerByChannel(channelFunc(i), ph, mtr)

		ph = newTestPeerHandler(id, logger)
		ph.onPeerFunc = channelOnPeerFunc(ph, dialCh, i)
		ph.onCloseFunc = channelOnCloseFunc(ph, dialCh, i)
		dialPD.registerByChannel(channelFunc(i), ph, mtr)

		if err := newDialer(channelFunc(i), dialPD.onConnect).Dial(l.address); err != nil {
			assert.FailNow(t, err.Error())
		}
		assertPeerHandler(numOfPeerHandler)
		assert.Equal(t, channelConnInfo(false, true, i), <-listenCh)
		assert.Equal(t, channelConnInfo(false, false, i), <-dialCh)
	}
	//registerByChannel failure if duplicated channel
	assert.False(t, listenPD.registerByChannel(channelFunc(0), nil, nil))

	//not registered channel case
	listenPD.unregisterByChannel(channelFunc(0))
	if err := newDialer(channelFunc(0), dialPD.onConnect).Dial(l.address); err != nil {
		assert.FailNow(t, err.Error())
	}
	assertPeerHandler(numOfPeerHandler)
	assert.Equal(t, channelConnInfo(false, false, 0), <-dialCh)
	//PeerDispatcher.onPeer of listener side should close by 'not exists PeerToPeer'
	//after that channel PeerHandler of dialer side called onClose
	assert.Equal(t, channelConnInfo(true, false, 0), <-dialCh)
}
