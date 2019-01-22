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
	testNumValidator   = 4
	testNumSeed        = 4
	testNumCitizen     = 4
	testNumAllowedPeer = 8
	testNumNotAllowedPeer = 2
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
	r.p2p.setEventCbFunc(p2pEventNotAllowed, r.ph.(*protocolHandler).protocol.Uint16(), r.onEvent)
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
	ctx := context.WithValue(context.Background(), "p2pConnInfo", newP2PConnInfo(r.p2p))
	ctx = context.WithValue(ctx, "name", r.name)
	r.ch <- ctx
}
func (r *testReactor) OnLeave(id module.PeerID) {
	r.log.Println("OnLeave", id)
}
func (r *testReactor) onEvent(evt string, p *Peer) {
	r.log.Println("onEvent", evt, p.id)

	ctx := context.WithValue(context.Background(), "event", evt)
	ctx = context.WithValue(ctx, "name", r.name)
	ctx = context.WithValue(ctx, "peer", p.id)
	r.ch <- ctx
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
	return newP2PConnInfo(r.p2p).String()
}

type p2pConnInfo struct {
	role     PeerRoleFlag
	friends  int
	parent   int
	uncles   int
	children int
	nephews  int
}

func newP2PConnInfo(p2p *PeerToPeer) *p2pConnInfo {//p2p.connections()
	parent := 0
	if p2p.parent != nil {
		parent = 1
	}
	return &p2pConnInfo{p2p.getRole(), p2p.friends.Len(), parent, p2p.uncles.Len(), p2p.children.Len(), p2p.nephews.Len()}
}
func (ci *p2pConnInfo) String() string {
	return fmt.Sprintf("role:%d, friends:%d, parent:%d, uncle:%d, children:%d, nephew:%d",
		ci.role,
		ci.friends,
		ci.parent,
		ci.uncles,
		ci.children,
		ci.nephews)
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
		nm := NewManager(testChannel, nt, "", roles...)
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
			trpi := ctx.Value("pi")
			if trpi == nil {
				continue
			}
			rpi := trpi.(module.ProtocolInfo)
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
func waitConnection(ch <-chan context.Context, limit []int, n int, d time.Duration) (map[string]time.Duration, time.Duration, error) {
	t := time.NewTimer(d)
	m := make(map[string]time.Duration)
	s := time.Now()
	var maxD time.Duration
	for {
		select {
		case ctx := <-ch:
			tci := ctx.Value("p2pConnInfo")
			if tci == nil {
				continue
			}
			ci := tci.(*p2pConnInfo)
			rname := ctx.Value("name").(string)
			switch ci.role {
			case p2pRoleRoot, p2pRoleRootSeed:
				if ci.friends == limit[p2pConnTypeFriend] &&
					ci.children == limit[p2pConnTypeChildren] && ci.nephews == limit[p2pConnTypeNephew] {
					if _, ok := m[rname]; !ok {
						m[rname] = time.Since(s)
					}
				}
			case p2pRoleSeed:
				if ci.parent == limit[p2pConnTypeParent] && ci.uncles == limit[p2pConnTypeUncle] &&
					ci.children == limit[p2pConnTypeChildren] && ci.nephews == limit[p2pConnTypeNephew] {
					if _, ok := m[rname]; !ok {
						m[rname] = time.Since(s)
					}
				}
			case p2pRoleNone:
				if ci.parent == limit[p2pConnTypeParent] && ci.uncles == limit[p2pConnTypeUncle] {
					if _, ok := m[rname]; !ok {
						m[rname] = time.Since(s)
					}
				}
			}
			if len(m) >= n {
				for _, md := range m {
					if maxD < md {
						maxD = md
					}
				}
				return m, maxD, nil
			}
		case <-t.C:
			for _, md := range m {
				if maxD < md {
					maxD = md
				}
			}
			return m, maxD, fmt.Errorf("timeout d:%v, limit:%v, n:%d, rn:%d", d, limit, n, len(m))
		}
	}
}

func waitEvent(ch <-chan context.Context, n int, d time.Duration, evt string, peers ...module.PeerID) (map[string]map[string]int, error) {
	t := time.NewTimer(d)
	m := make(map[string]map[string]int)
	for _,p := range peers {
		m[p.String()] = make(map[string]int)
	}
	for {
		select {
		case ctx := <-ch:
			tevt := ctx.Value("event")
			if tevt == nil {
				continue
			}
			revt := tevt.(string)
			rpeer := ctx.Value("peer").(module.PeerID)
			rname := ctx.Value("name").(string)
			if revt == evt {
				rm := m[rpeer.String()]
				if _, ok := rm[rname]; !ok {
					rm[rname] = 0
				}
				rm[rname]++

				done := true
				for _, tm := range m {
					 if len(tm) < n {
					 	done = false
						break
					 }
				}
				if done {
					return m, nil
				}

			}
		case <-t.C:
			return m, fmt.Errorf("timeout d:%v, evt:%v, peers:%v, n:%d, rn:%d", d, evt, peers, n, len(m))
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

	limit := []int{0, 1, DefaultChildrenLimit, DefaultUncleLimit, DefaultNephewLimit, testNumValidator - 1}
	n := testNumValidator + testNumSeed + testNumCitizen
	connMap, maxD, err := waitConnection(ch, limit, n, 10*DefaultSeedPeriod)
	t.Log(time.Now(), "max:", maxD, connMap)
	assert.NoError(t, err, "waitConnection", connMap)

	t.Log(time.Now(), "Messaging")

	msg := m["TestValidator"][0].Broadcast("Test1")
	n = testNumValidator - 1 + testNumSeed + testNumCitizen
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
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
		if p != nil {
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

	time.Sleep(2 * DefaultAlternateSendPeriod)

	for _, arr := range m {
		for _, r := range arr {
			r.nt.Close()
		}
	}
	t.Log(time.Now(), "Finish")
}

func Test_network_allowedPeer(t *testing.T) {
	m := make(map[string][]*testReactor)
	p := 8080
	m["TestAllowed"], p = generateNetwork("TestAllowed", p, testNumAllowedPeer, t, module.ROLE_VALIDATOR, module.ROLE_VALIDATOR)
	m["TestNotAllowed"], p = generateNetwork("TestNotAllowed", p, testNumNotAllowedPeer, t, module.ROLE_VALIDATOR, module.ROLE_VALIDATOR)
	allowed := make([]module.PeerID, 0)
	notAllowed := make([]module.PeerID, 0)

	for _, r := range m["TestAllowed"] {
		allowed = append(allowed, r.nt.PeerID())
	}
	for _, r := range m["TestNotAllowed"] {
		notAllowed = append(notAllowed, r.nt.PeerID())
	}
	for _, r := range m["TestAllowed"] {
		r.nm.SetRole(module.ROLE_NORMAL, allowed...)
	}
	sr := m["TestAllowed"][0]
	sna := sr.nt.Address()
	for _, r := range m["TestAllowed"] {
		if r.nt.Address() != sna {
			r.nt.Dial(sna, testChannel)
			time.Sleep(100 * time.Millisecond)
		}
	}

	ch := make(chan context.Context, testNumAllowedPeer+testNumNotAllowedPeer)
	for _, v := range m {
		for _, r := range v {
			r.ch = ch
		}
	}

	limit := []int{0, 0, 0, 0, 0, testNumAllowedPeer - 1}
	n := testNumAllowedPeer
	connMap, maxD, err := waitConnection(ch, limit, n, 10*DefaultSeedPeriod)
	t.Log(time.Now(), "max:", maxD, connMap)
	assert.NoError(t, err, "waitConnection", connMap)

	go func() {
		for _, r := range m["TestAllowed"] {
			for _, nr := range m["TestNotAllowed"] {
				nr.nt.Dial(r.nt.Address(), testChannel)
			}
		}
	}()
	evtMap, err := waitEvent(ch, n, 2*time.Second, p2pEventNotAllowed, notAllowed...)
	t.Log(time.Now(), "Before", evtMap)
	assert.NoError(t, err, "waitEvent", evtMap)

	t.Log(time.Now(), "Messaging")
	msg := m["TestAllowed"][0].Broadcast("Test1")
	n = testNumAllowedPeer - 1
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1")

	remove := allowed[testNumAllowedPeer-1]
	go func() {
		for _, r := range m["TestAllowed"] {
			r.nm.RemoveRole(module.ROLE_NORMAL, remove)
		}
	}()
	evtMap, err = waitEvent(ch, n-1, 2*time.Second, p2pEventNotAllowed, remove)
	t.Log(time.Now(), "After", evtMap)
	assert.NoError(t, err, "waitEvent2", evtMap)

	msg = m["TestAllowed"][0].Broadcast("Test2")
	n = testNumAllowedPeer - 2
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test2")

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
