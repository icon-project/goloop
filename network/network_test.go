package network

import (
	"testing"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	testSeedAddress             = "127.0.0.1:8081"
	testCitizenAddress          = "127.0.0.1:8082"
	PayloadTestNetworkBroadcast = "TestBroasdcast"
	PayloadTestNetworkMulticast = "TestMulticast"
	PayloadTestNetworkRequest   = "Hello"
	PayloadTestNetworkResponse  = "World"
)

var (
	ProtoTestNetworkBroadcast module.ProtocolInfo = protocolInfo(0x0100)
	ProtoTestNetworkMulticast module.ProtocolInfo = protocolInfo(0x0200)
	ProtoTestNetworkRequest   module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestNetworkResponse  module.ProtocolInfo = protocolInfo(0x0400)
)

var (
	testSubProtocols = []module.ProtocolInfo{
		ProtoTestNetworkBroadcast,
		ProtoTestNetworkMulticast,
		ProtoTestNetworkRequest,
		ProtoTestNetworkResponse,
	}
)

type TestReactor struct {
	name string
	ms   module.Membership
	t    *testing.T
}

func (r *TestReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (re bool, err error) {
	s := string(b)
	r.t.Logf("%s.OnReceive pi:%v, payload:%v, id:%v", r.name, pi, b, id)
	switch pi {
	case ProtoTestNetworkBroadcast:
		r.t.Logf("%s.OnReceive ProtoTestNetworkBroadcast %s", r.name, s)
		re = true
	case ProtoTestNetworkMulticast:
		r.t.Logf("%s.OnReceive ProtoTestNetworkMulticast %s", r.name, s)
		re = true
	case ProtoTestNetworkRequest:
		r.t.Logf("%s.OnReceive ProtoTestNetworkRequest %s", r.name, s)
		r.Response(id)
	case ProtoTestNetworkResponse:
		r.t.Logf("%s.OnReceive ProtoTestNetworkResponse %s", r.name, s)
	default:
		re = false
	}
	return
}

func (r *TestReactor) OnError() {

}

func (r *TestReactor) Broadcast() {
	r.ms.Broadcast(ProtoTestNetworkBroadcast, []byte(r.name+PayloadTestNetworkBroadcast), module.BROADCAST_ALL)
	r.t.Logf("%s.Broadcast ProtoTestNetworkBroadcast %s", r.name, r.name+PayloadTestNetworkBroadcast)
}

func (r *TestReactor) Multicast() {
	r.ms.Multicast(ProtoTestNetworkMulticast, []byte(r.name+PayloadTestNetworkMulticast), module.ROLE_VALIDATOR)
	r.t.Logf("%s.Multicast ProtoTestNetworkMulticast %s", r.name, r.name+PayloadTestNetworkMulticast)
}

func (r *TestReactor) Request(id module.PeerID) {
	r.ms.Unicast(ProtoTestNetworkRequest, []byte(r.name+PayloadTestNetworkRequest), id)
	r.t.Logf("%s.Request ProtoTestNetworkRequest %s", r.name, r.name+PayloadTestNetworkRequest)
}

func (r *TestReactor) Response(id module.PeerID) {
	r.ms.Unicast(ProtoTestNetworkResponse, []byte(r.name+PayloadTestNetworkResponse), id)
	r.t.Logf("%s.Response ProtoTestNetworkResponse %s", r.name, r.name+PayloadTestNetworkResponse)
}

func newTestNetwork(channel string, pd *PeerDispatcher, addr string, d *Dialer) (*manager, module.Membership) {
	nm := newManager(channel, pd.self, NetAddress(addr), d)
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
	spd, sl, sd := newTestTransport(testChannel, testSeedAddress)
	cpd, cl, cd := newTestTransport(testChannel, testCitizenAddress)

	//t.Logf("pd:%v,spd:%v,cpd:%v", pd.self, spd.self, cpd.self)

	_, ms := getNetwork(testChannel)
	_, sms := newTestNetwork(testChannel, spd, sl.address, sd)
	_, cms := newTestNetwork(testChannel, cpd, cl.address, cd)

	vr := &TestReactor{"TestValidator", ms, t}
	sr := &TestReactor{"TestSeed", sms, t}
	cr := &TestReactor{"TestCitizen", cms, t}
	ms.RegistReactor(vr.name, vr, testSubProtocols)
	sms.RegistReactor(sr.name, sr, testSubProtocols)
	cms.RegistReactor(cr.name, cr, testSubProtocols)

	vrp := []module.PeerID{pd.self}
	srp := []module.PeerID{pd.self, spd.self}
	ms.AddRole(module.ROLE_VALIDATOR, vrp...)
	ms.AddRole(module.ROLE_SEED, srp...)
	sms.AddRole(module.ROLE_VALIDATOR, vrp...)
	sms.AddRole(module.ROLE_SEED, srp...)
	cms.AddRole(module.ROLE_VALIDATOR, vrp...)
	cms.AddRole(module.ROLE_SEED, srp...)

	l.Listen()
	sl.Listen()
	cl.Listen()

	// snal := []NetAddress{NetAddress(sl.address)}
	// pd.peerToPeers[testChannel].seeds.Merge(snal...)
	// spd.peerToPeers[testChannel].seeds.Merge(snal...)
	// cpd.peerToPeers[testChannel].seeds.Merge(snal...)
	d.Dial(sl.address)
	cd.Dial(sl.address)

	time.Sleep(5 * time.Second)
	vr.Broadcast()
	sr.Multicast()
	// sp := cnm.peerToPeer.getPeer(spd.self)
	// err := sp.conn.Close()
	// if err != nil {
	// 	t.Logf("sp.conn.Close error:%v", err)
	// }
	cr.Multicast()
	cr.Request(spd.self)
	time.Sleep(5 * time.Second)
	l.Close()
	sl.Close()
	cl.Close()
}
