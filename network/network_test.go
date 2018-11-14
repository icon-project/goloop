package network

import (
	"testing"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	PROTO_TEST_1   = 0x0101
	PROTO_TEST_2   = 0x0201
	PAYLOAD_TEST_1 = "Hello"
	PAYLOAD_TEST_2 = "World"
)

var (
	testSubProtocols = []module.ProtocolInfo{PROTO_TEST_1, PROTO_TEST_2}
)

type TestReactor struct {
	ms module.Membership
	t  *testing.T
}

func (r *TestReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	switch pi {
	case PROTO_TEST_1:
		r.t.Logf("TestReactor.OnReceive pi:%v, payload:%v, id:%v",
			pi, string(b), id)
	case PROTO_TEST_2:
		r.t.Logf("TestReactor.OnReceive pi:%v, payload:%v, id:%v",
			pi, string(b), id)
		return true, nil
	default:
	}
	return false, nil
}

func (r *TestReactor) OnError() {

}

func (r *TestReactor) SendTest1() {
	r.ms.Broadcast(PROTO_TEST_1, []byte(PAYLOAD_TEST_1), module.BROADCAST_ALL)
}

func newTestNetwork(channel string, pd *PeerDispatcher, addr string) (module.NetworkManager, module.Membership) {
	nm := newManager(channel, pd.self, NetAddress(addr))
	pd.registPeerToPeer(nm.peerToPeer)
	ms := nm.GetMembership(DefaultMembershipName)
	return nm, ms
}

func getNetwork(channel string) (module.NetworkManager, module.Membership) {
	nm := GetNetworkManager(channel)
	ms := nm.GetMembership(DefaultMembershipName)
	return nm, ms
}

func Test_network(t *testing.T) {
	pd, l, d := getTransport(testChannel)
	tpd, tl, _ := newTestTransport(testChannel, testListenAddress)

	_, ms := getNetwork(testChannel)
	_, tms := newTestNetwork(testChannel, tpd, testListenAddress)

	tr := &TestReactor{ms, t}
	ms.RegistReactor("TestReactor", tr, testSubProtocols)
	tms.RegistReactor("TestReactor", tr, testSubProtocols)

	ms.AddRole(module.ROLE_VALIDATOR, pd.self, tpd.self)
	ms.AddRole(module.ROLE_SEED, pd.self, tpd.self)
	tms.AddRole(module.ROLE_VALIDATOR, pd.self, tpd.self)
	tms.AddRole(module.ROLE_SEED, pd.self, tpd.self)

	l.Listen()
	tl.Listen()

	//TODO connect each other, config p2p.self.role
	d.Dial(testListenAddress)
	time.Sleep(5 * time.Second)

	l.Close()
	tl.Close()
}
