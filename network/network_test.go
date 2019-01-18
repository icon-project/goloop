package network

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"

	"github.com/icon-project/goloop/module"
)

const (
	testNumValidator = 4
	testNumSeed      = 4
	testNumCitizen   = 4
)

var (
	ProtoTestNetworkBroadcast module.ProtocolInfo = protocolInfo(0x0100)
	ProtoTestNetworkMulticast module.ProtocolInfo = protocolInfo(0x0200)
	ProtoTestNetworkRequest   module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestNetworkResponse  module.ProtocolInfo = protocolInfo(0x0400)
	ProtoTestNetworkNeighbor  module.ProtocolInfo = protocolInfo(0x0500)
)

var (
	testSubProtocols = []module.ProtocolInfo{
		ProtoTestNetworkBroadcast,
		ProtoTestNetworkMulticast,
		ProtoTestNetworkRequest,
		ProtoTestNetworkResponse,
		ProtoTestNetworkNeighbor,
	}
)

type testReactor struct {
	name         string
	ph           module.ProtocolHandler
	codecHandle  codec.Handle
	log          *logger
	t            *testing.T
	nm           module.NetworkManager
	nt           module.NetworkTransport
	p2p          *PeerToPeer
	ch           chan<- context.Context
	responseFunc func(r *testReactor, rm *testNetworkRequest, id module.PeerID) error
}

func newTestReactor(name string, nm module.NetworkManager, t *testing.T) *testReactor {
	r := &testReactor{name: name, nm: nm, codecHandle: &codec.MsgpackHandle{}, log: newLogger("TestReactor", name), t: t}
	r.ph, _ = nm.RegisterReactor(name, r, testSubProtocols, 1)
	r.p2p = nm.(*manager).p2p
	r.t.Log(time.Now(), r.name, "newTestReactor", r.p2p.self.id)
	return r
}

type testNetworkBroadcast struct {
	Message string
}

type testNetworkMulticast struct {
	Message string
}

type testNetworkRequest struct {
	Message string
}

type testNetworkResponse struct {
	Message string
}

func (r *testReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (re bool, err error) {
	r.log.Println("OnReceive", pi, b, id)
	var msg string
	switch pi {
	case ProtoTestNetworkBroadcast:
		rm := &testNetworkBroadcast{}
		r.decode(b, rm)
		msg = rm.Message
		r.log.Println("handleProtoTestNetworkBroadcast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkNeighbor:
		rm := &testNetworkBroadcast{}
		r.decode(b, rm)
		msg = rm.Message
		r.log.Println("handleProtoTestNetworkNeighbor", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = false
	case ProtoTestNetworkMulticast:
		rm := &testNetworkMulticast{}
		r.decode(b, rm)
		msg = rm.Message
		r.log.Println("handleProtoTestNetworkMulticast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkRequest:
		rm := &testNetworkRequest{}
		r.decode(b, rm)
		msg = rm.Message
		r.log.Println("handleProtoTestNetworkRequest", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm)
		if r.responseFunc != nil {
			err = r.responseFunc(r, rm, id)
		} else {
			r.Response(rm.Message, id)
		}
	case ProtoTestNetworkResponse:
		rm := &testNetworkResponse{}
		r.decode(b, rm)
		msg = rm.Message
		r.log.Println("handleProtoTestNetworkResponse", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm)
	default:
		re = false
	}
	ctx := context.WithValue(context.Background(), "pi", pi)
	ctx = context.WithValue(ctx, "msg", msg)
	ctx = context.WithValue(ctx, "name", r.name)
	r.ch <- ctx
	return
}

func (r *testReactor) OnError(err error, pi module.ProtocolInfo, b []byte, id module.PeerID) {
}
func (r *testReactor) OnJoin(id module.PeerID) {
	r.log.Println("OnJoin", id)
}
func (r *testReactor) OnLeave(id module.PeerID) {
	r.log.Println("OnLeave", id)
}

func (r *testReactor) encode(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.NewEncoderBytes(&b, r.codecHandle)
	enc.MustEncode(v)
	return b
}

func (r *testReactor) decode(b []byte, v interface{}) {
	dec := codec.NewDecoderBytes(b, r.codecHandle)
	dec.MustDecode(v)
}

func (r *testReactor) p2pConn() string {
	p2p := r.p2p
	var p int
	if p2p.parent != nil {
		p = 1
	}
	return fmt.Sprintf("friends:%d, parent:%d, uncle:%d, children:%d, nephew:%d",
		p2p.friends.Len(),
		p,
		p2p.uncles.Len(),
		p2p.children.Len(),
		p2p.nephews.Len())
}

func (r *testReactor) Broadcast(msg string) string {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("Broadcast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Broadcast", m, r.p2pConn())
	r.ph.Broadcast(ProtoTestNetworkBroadcast, r.encode(m), module.BROADCAST_ALL)
	r.log.Println("Broadcast", m)
	return m.Message
}

func (r *testReactor) BroadcastNeighbor(msg string) string {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("BroadcastNeighbor.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "BroadcastNeighbor", m, r.p2pConn())
	r.ph.Broadcast(ProtoTestNetworkNeighbor, r.encode(m), module.BROADCAST_NEIGHBOR)
	r.log.Println("BroadcastNeighbor", m)
	return m.Message
}

func (r *testReactor) Multicast(msg string) string {
	m := &testNetworkMulticast{Message: fmt.Sprintf("Multicast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Multicast", m, r.p2pConn())
	r.ph.Multicast(ProtoTestNetworkMulticast, r.encode(m), module.ROLE_VALIDATOR)
	r.log.Println("Multicast", m)
	return m.Message
}

func (r *testReactor) Request(msg string, id module.PeerID) string {
	m := &testNetworkRequest{Message: fmt.Sprintf("Request.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Request", m, r.p2pConn())
	r.ph.Unicast(ProtoTestNetworkRequest, r.encode(m), id)
	r.log.Println("Request", m, id)
	return m.Message
}

func (r *testReactor) Response(msg string, id module.PeerID) string {
	m := &testNetworkResponse{Message: fmt.Sprintf("Response.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Response", m, r.p2pConn())
	r.ph.Unicast(ProtoTestNetworkResponse, r.encode(m), id)
	r.log.Println("Response", m, id)
	return m.Message
}

func generateNetwork(name string, port int, n int, t *testing.T, roles ...module.Role) ([]*testReactor, int) {
	arr := make([]*testReactor, n)
	for i := 0; i < n; i++ {
		nt := NewTransport(fmt.Sprintf("127.0.0.1:%d", port+i), walletFromGeneratedPrivateKey())
		nm := NewManager(testChannel, nt, roles...)
		r := newTestReactor(fmt.Sprintf("%s_%d", name, i), nm, t)
		r.nt = nt
		r.nt.Listen()
		arr[i] = r
	}
	return arr, port + n
}

func timeout(ch <-chan string, d time.Duration) (string, error) {
	t := time.NewTimer(time.Second)
	select {
	case s := <-ch:
		return s, nil
	case <-t.C:
		return "", fmt.Errorf("timeout d:%v", d)
	}
}

func wait(ch <-chan context.Context, pi module.ProtocolInfo, msg string, n int, d time.Duration, dest ...string) error {
	rn := 0
	t := time.NewTimer(d)
	m := make(map[string]int)
	for _, rname := range dest {
		m[rname] = 0
	}
	for {
		select {
		case ctx := <-ch:
			rpi := ctx.Value("pi").(module.ProtocolInfo)
			rmsg := ctx.Value("msg").(string)
			rname := ctx.Value("name").(string)
			if rpi.Uint16() == pi.Uint16() && msg == rmsg {
				z := len(m)
				if z > 0 {
					if c, ok := m[rname]; ok {
						m[rname] = c + 1
					}
					for _, c := range m {
						if c > 0 {
							z--
						}
					}
					if z < 1 {
						return nil
					}
				} else {
					rn++
					if rn >= n {
						return nil
					}
				}
			}
		case <-t.C:
			return fmt.Errorf("timeout d:%v pi:%x msg:%s n:%d rn:%d dest:%v", d, pi.Uint16(), msg, n, rn, dest)
		}
	}
}

func Test_network(t *testing.T) {
	m := make(map[string][]*testReactor)
	p := 8080
	m["TestCitizen"], p = generateNetwork("TestCitizen", p, testNumCitizen, t)                              //8080~8083
	m["TestSeed"], p = generateNetwork("TestSeed", p, testNumSeed, t, module.ROLE_SEED)                     //8084~8087
	m["TestValidator"], p = generateNetwork("TestValidator", p, testNumValidator, t, module.ROLE_VALIDATOR) //8088~8091

	sr := m["TestSeed"][0]
	sna := sr.nt.Address()
	for _, arr := range m {
		for _, r := range arr {
			if r.nt.Address() != sna {
				r.nt.Dial(sna, testChannel)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	ch := make(chan context.Context, testNumCitizen+testNumSeed+testNumValidator)
	for _, v := range m {
		for _, r := range v {
			r.ch = ch
		}
	}

	time.Sleep(3 * DefaultSeedPeriod)
	//time.Sleep(DefaultDiscoveryPeriod)
	t.Log(time.Now(), "Messaging")

	msg := m["TestValidator"][0].Broadcast("Test1")
	n := testNumValidator - 1 + testNumSeed + testNumCitizen
	err := wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1")

	msg = m["TestValidator"][0].BroadcastNeighbor("Test2")
	n = testNumValidator - 1 + DefaultChildrenLimit + DefaultNephewLimit
	err = wait(ch, ProtoTestNetworkNeighbor, msg, n, time.Second)
	assert.NoError(t, err, "BroadcastNeighbor", "Test2")

	msg = m["TestValidator"][0].Multicast("Test3")
	n = testNumValidator - 1
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second)
	assert.NoError(t, err, "Multicast", "Test3")

	msg = m["TestSeed"][0].Multicast("Test4")
	n = testNumValidator
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second)
	assert.NoError(t, err, "Multicast", "Test4")

	msg = m["TestCitizen"][0].Multicast("Test5")
	n = testNumValidator + 1 + DefaultUncleLimit
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second+DefaultAlternateSendPeriod)
	assert.NoError(t, err, "Multicast", "Test5")

	tr := sr
	for _, r := range m["TestSeed"] {
		p := m["TestCitizen"][0].p2p.getPeer(r.nt.PeerID(), true)
		if p != nil{
			tr = r
			break
		}
	}
	respCh := make(chan string, 1)
	tr.responseFunc = func(r *testReactor, rm *testNetworkRequest, id module.PeerID) error {
		m := r.Response(rm.Message, id)
		respCh <- m
		return nil
	}
	msg = m["TestCitizen"][0].Request("Test6", tr.nt.PeerID())
	err = wait(ch, ProtoTestNetworkRequest, msg, 1, time.Second, tr.name)
	assert.NoError(t, err, "Request", "Test6")

	msg, err = timeout(respCh, time.Second)
	assert.NoError(t, err, "timeout", "responseFunc")

	err = wait(ch, ProtoTestNetworkResponse, msg, 1, time.Second, m["TestCitizen"][0].name)
	assert.NoError(t, err, "Response", "Test6")

	for _, arr := range m {
		for _, r := range arr {
			r.nt.Close()
		}
	}
	t.Log(time.Now(), "Finish")
}

func Test_sort(t *testing.T) {
	s := []int{1, 2, 3}
	fmt.Println(s)
	sort.Slice(s, func(i, j int) bool { return s[i] > s[j] })
	fmt.Println(s)
}
