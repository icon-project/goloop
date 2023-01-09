package network

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	testNumValidator      = 4
	testNumSeed           = 4
	testNumCitizen        = 4
	testNumAllowedPeer    = 8
	testNumNotAllowedPeer = 2
	testProtoPriority     = 1
	testNumChild          = 4

	testValidator  = "TestValidator"
	testSeed       = "TestSeed"
	testCitizen    = "TestCitizen"
	testAllowed    = "TestAllowed"
	testNotAllowed = "TestNotAllowed"
	testChild      = "TestChild"
)

var (
	ProtoTestNetwork          = module.ProtoReserved
	ProtoTestNetworkBroadcast = module.ProtocolInfo(0x0100)
	ProtoTestNetworkMulticast = module.ProtocolInfo(0x0200)
	ProtoTestNetworkRequest   = module.ProtocolInfo(0x0300)
	ProtoTestNetworkResponse  = module.ProtocolInfo(0x0400)
	ProtoTestNetworkNeighbor  = module.ProtocolInfo(0x0500)
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

var (
	normalConnectionLimit    = []int{0, 1, 0, DefaultUnclesLimit, 0, 0}
	validatorConnectionLimit = []int{0, 0, 0, 0, 0, testNumValidator - 1}
	defaultConnectionLimit   = map[PeerRoleFlag][]int{
		p2pRoleNone: normalConnectionLimit,
		p2pRoleSeed: normalConnectionLimit,
		p2pRoleRoot: validatorConnectionLimit,
	}
)

type testReactor struct {
	name         string
	ph           module.ProtocolHandler
	logger       log.Logger
	t            *testing.T
	nm           module.NetworkManager
	nt           module.NetworkTransport
	p2p          *PeerToPeer
	ch           chan<- context.Context
	responseFunc func(r *testReactor, rm *testNetworkRequest, id module.PeerID) error
	c            module.Chain
}

func newTestReactor(name string, nm module.NetworkManager, pi module.ProtocolInfo, t *testing.T) *testReactor {
	logger := nm.(*manager).logger.WithFields(log.Fields{"TestReactor": name})
	r := &testReactor{name: name, nm: nm, logger: logger, t: t}
	ph, err := nm.RegisterReactor(name, pi, r, testSubProtocols, testProtoPriority, module.NotRegisteredProtocolPolicyClose)
	assert.NoError(t, err, "RegisterReactor")
	r.ph = ph
	r.p2p = nm.(*manager).p2p
	r.p2p.setEventCbFunc(p2pEventNotAllowed, r.ph.(*protocolHandler).protocol.Uint16(), r.onEvent)
	r.t.Log(time.Now(), r.name, "newTestReactor", r.p2p.ID(), r.p2p.NetAddress())
	return r
}

type testNetworkMessage struct {
	Message string
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
	r.logger.Println("OnReceive", pi, b, id)
	var msg string
	switch pi {
	case ProtoTestNetworkBroadcast:
		rm := &testNetworkBroadcast{}
		r.decode(b, rm)
		msg = rm.Message
		r.logger.Println("handleProtoTestNetworkBroadcast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkNeighbor:
		rm := &testNetworkBroadcast{}
		r.decode(b, rm)
		msg = rm.Message
		r.logger.Println("handleProtoTestNetworkNeighbor", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = false
	case ProtoTestNetworkMulticast:
		rm := &testNetworkMulticast{}
		r.decode(b, rm)
		msg = rm.Message
		r.logger.Println("handleProtoTestNetworkMulticast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkRequest:
		rm := &testNetworkRequest{}
		r.decode(b, rm)
		msg = rm.Message
		r.logger.Println("handleProtoTestNetworkRequest", rm, id)
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
		r.logger.Println("handleProtoTestNetworkResponse", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm)
	default:
		re = false
	}
	ctx := context.WithValue(context.Background(), "op", "recv")
	ctx = context.WithValue(ctx, "pi", pi)
	ctx = context.WithValue(ctx, "msg", msg)
	ctx = context.WithValue(ctx, "name", r.name)
	r.ch <- ctx
	return
}

func (r *testReactor) OnJoin(id module.PeerID) {
	r.logger.Println("OnJoin", id)
	ctx := context.WithValue(context.Background(), "op", "join")
	ctx = context.WithValue(ctx, "p2pConnInfo", r.p2pConnInfo())
	ctx = context.WithValue(ctx, "name", r.name)
	r.ch <- ctx
}
func (r *testReactor) OnLeave(id module.PeerID) {
	r.logger.Println("OnLeave", id)
}
func (r *testReactor) onEvent(evt string, p *Peer) {
	r.logger.Println("onEvent", evt, p.ID())
	ctx := context.WithValue(context.Background(), "op", "event")
	ctx = context.WithValue(ctx, "event", evt)
	ctx = context.WithValue(ctx, "name", r.name)
	ctx = context.WithValue(ctx, "peer", p.ID())
	r.ch <- ctx
}

func (r *testReactor) encode(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.MP.NewEncoderBytes(&b)
	if err := enc.Encode(v); err != nil {
		log.Panicf("Fail to encode err=%+v", err)
	}
	return b
}

func (r *testReactor) decode(b []byte, v interface{}) {
	codec.MP.MustUnmarshalFromBytes(b, v)
}

func (r *testReactor) p2pConn() string {
	return r.p2pConnInfo().String()
}

func (r *testReactor) p2pConnInfo() *p2pConnInfo {
	m := Inspect(r.c, true)["p2p"].(map[string]interface{})
	role := m["self"].(map[string]interface{})["role"]
	parent := 0
	if len(m["parent"].(map[string]interface{})) > 0 {
		parent = 1
	}
	return &p2pConnInfo{
		role:     role.(PeerRoleFlag),
		friends:  len(m["friends"].([]map[string]interface{})),
		parent:   parent,
		uncles:   len(m["uncles"].([]map[string]interface{})),
		children: len(m["children"].([]map[string]interface{})),
		nephews:  len(m["nephews"].([]map[string]interface{})),
		others:   len(m["others"].([]map[string]interface{})),
	}
}

type p2pConnInfo struct {
	role     PeerRoleFlag
	friends  int
	parent   int
	uncles   int
	children int
	nephews  int
	others   int
}

func (ci *p2pConnInfo) String() string {
	return fmt.Sprintf("role:%d, friends:%d, parent:%d, uncle:%d, children:%d, nephew:%d, others:%d",
		ci.role,
		ci.friends,
		ci.parent,
		ci.uncles,
		ci.children,
		ci.nephews,
		ci.others)
}

func (r *testReactor) Broadcast(msg string) string {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("Broadcast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Broadcast", m, r.p2pConn())
	err := r.ph.Broadcast(ProtoTestNetworkBroadcast, r.encode(m), module.BroadcastAll)
	assert.NoError(r.t, err, m.Message)
	r.logger.Println("Broadcast", m)
	return m.Message
}

func (r *testReactor) BroadcastNeighbor(msg string) (string, int) {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("BroadcastNeighbor.%s.%s", msg, r.name)}
	ci := r.p2pConnInfo()
	r.t.Log(time.Now(), r.name, "BroadcastNeighbor", m, ci.String())
	err := r.ph.Broadcast(ProtoTestNetworkNeighbor, r.encode(m), module.BroadcastNeighbor)
	assert.NoError(r.t, err, m.Message)
	r.logger.Println("BroadcastNeighbor", m)
	return m.Message, ci.friends + ci.children + ci.nephews
}

func (r *testReactor) Multicast(msg string) string {
	m := &testNetworkMulticast{Message: fmt.Sprintf("Multicast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Multicast", m, r.p2pConn())
	err := r.ph.Multicast(ProtoTestNetworkMulticast, r.encode(m), module.RoleValidator)
	assert.NoError(r.t, err, m.Message)
	r.logger.Println("Multicast", m)
	return m.Message
}

func (r *testReactor) Request(msg string, id module.PeerID) string {
	m := &testNetworkRequest{Message: fmt.Sprintf("Request.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Request", m, r.p2pConn())
	err := r.ph.Unicast(ProtoTestNetworkRequest, r.encode(m), id)
	assert.NoError(r.t, err, m.Message)
	r.logger.Println("Request", m, id)
	return m.Message
}

func (r *testReactor) Response(msg string, id module.PeerID) string {
	m := &testNetworkResponse{Message: fmt.Sprintf("Response.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Response", m, r.p2pConn())
	err := r.ph.Unicast(ProtoTestNetworkResponse, r.encode(m), id)
	assert.NoError(r.t, err, m.Message)
	r.logger.Println("Response", m, id)
	return m.Message
}

type dummyChain struct {
	module.Chain
	nid       int
	metricCtx context.Context
	logger    log.Logger
	nm        module.NetworkManager
}

func (c *dummyChain) NID() int                              { return c.nid }
func (c *dummyChain) CID() int                              { return c.nid }
func (c *dummyChain) NetID() int                            { return c.nid }
func (c *dummyChain) Logger() log.Logger                    { return c.logger }
func (c *dummyChain) MetricContext() context.Context        { return c.metricCtx }
func (c *dummyChain) ChildrenLimit() int                    { return -1 }
func (c *dummyChain) NephewsLimit() int                     { return -1 }
func (c *dummyChain) NetworkManager() module.NetworkManager { return c.nm }

type dummyReactor struct{}

func (d dummyReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	return false, nil
}
func (d dummyReactor) OnJoin(id module.PeerID)  {}
func (d dummyReactor) OnLeave(id module.PeerID) {}

func failIfError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err != nil {
		assert.FailNow(t, err.Error(), msgAndArgs)
	}
}

func generateNetwork(name string, n int, t *testing.T, roles ...module.Role) []*testReactor {
	lv := log.GlobalLogger().GetLevel()
	if testing.Verbose() {
		lv = log.TraceLevel
	}
	arr := make([]*testReactor, n)
	for i := 0; i < n; i++ {
		w := walletFromGeneratedPrivateKey()
		nodeLogger := log.New().WithFields(log.Fields{log.FieldKeyWallet: hex.EncodeToString(w.Address().ID())})
		nodeLogger.SetLevel(lv)
		nodeLogger.SetConsoleLevel(lv)
		nt := NewTransport(getAvailableLocalhostAddress(t), w, nodeLogger)
		chainLogger := nodeLogger.WithFields(log.Fields{log.FieldKeyCID: "1"})
		c := &dummyChain{nid: 1, metricCtx: context.Background(), logger: chainLogger}
		nm := NewManager(c, nt, "", roles...)
		c.nm = nm
		var emptyProtocols []module.ProtocolInfo
		for j, p := range defaultProtocols {
			_, err := nm.RegisterReactor(fmt.Sprintf("default_%d", j), p, dummyReactor{}, emptyProtocols, testProtoPriority, module.NotRegisteredProtocolPolicyClose)
			assert.NoError(t, err)
			failIfError(t, err, "fail to register defaultProtocols", p)
		}
		r := newTestReactor(fmt.Sprintf("%s_%d", name, i), nm, ProtoTestNetwork, t)
		r.nt = nt
		r.c = c
		failIfError(t, r.nt.Listen(), "fail to listen", r.name)
		failIfError(t, nm.Start(), "fail to start", r.name)
		arr[i] = r
	}
	return arr
}

func timeout(ch <-chan string, d time.Duration) (string, error) {
	t := time.NewTimer(d)
	select {
	case s := <-ch:
		return s, nil
	case <-t.C:
		return "", fmt.Errorf("timeout d:%v", d)
	}
}

func timeoutCtx(ch <-chan context.Context, d time.Duration, k interface{}) (context.Context, error) {
	t := time.NewTimer(d)
	for {
		select {
		case s := <-ch:
			if s.Value(k) != nil {
				return s, nil
			}
			str := ""
			for _, key := range []string{"op", "pi", "msg", "name", "p2pConnInfo", "event", "peer", "error"} {
				str += fmt.Sprintf("%s:%#v,", key, s.Value(key))
			}
			log.Println("ignore timeoutCtx", str)
		case <-t.C:
			return nil, fmt.Errorf("timeout d:%v", d)
		}
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
func waitConnection(ch <-chan context.Context, limit map[PeerRoleFlag][]int, n int, d time.Duration) (map[string]time.Duration, time.Duration, error) {
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
			roleBasedLimit := limit[ci.role]
			fmt.Println("rname:", rname, "roleBasedLimit:", roleBasedLimit, "ci:", ci)
			if ci.role.Has(p2pRoleRoot) {
				if ci.friends >= roleBasedLimit[p2pConnTypeFriend] &&
					ci.children >= roleBasedLimit[p2pConnTypeChildren] && ci.nephews >= roleBasedLimit[p2pConnTypeNephew] {
					if _, ok := m[rname]; !ok {
						m[rname] = time.Since(s)
					}
				}
			} else if ci.role.Has(p2pRoleSeed) {
				if ci.parent >= roleBasedLimit[p2pConnTypeParent] && ci.uncles >= roleBasedLimit[p2pConnTypeUncle] &&
					ci.children >= roleBasedLimit[p2pConnTypeChildren] && ci.nephews >= roleBasedLimit[p2pConnTypeNephew] {
					if _, ok := m[rname]; !ok {
						m[rname] = time.Since(s)
					}
				}
			} else {
				if ci.parent >= roleBasedLimit[p2pConnTypeParent] && ci.uncles >= roleBasedLimit[p2pConnTypeUncle] {
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
			return m, maxD, fmt.Errorf("timeout d:%v, limit:%v, expect:%d, complete:%d", d, limit, n, len(m))
		}
	}
}

func waitEvent(ch <-chan context.Context, n int, d time.Duration, evt string, peers ...module.PeerID) (map[string]map[string]int, error) {
	t := time.NewTimer(d)
	m := make(map[string]map[string]int)
	for _, p := range peers {
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

func dailByMap(t *testing.T, m map[string][]*testReactor, na NetAddress, delay time.Duration) {
	for _, arr := range m {
		dailByList(t, arr, na, delay)
	}
}
func dailByList(t *testing.T, arr []*testReactor, na NetAddress, delay time.Duration) {
	for _, r := range arr {
		if r.p2p.NetAddress() != na {
			failIfError(t, r.p2p.dial(na), "dial", r.name, "->", na)
			if delay > 0 {
				time.Sleep(delay)
			}
		}
	}
}

func listenerClose(t *testing.T, m map[string][]*testReactor) {
	var wg sync.WaitGroup
	for _, arr := range m {
		for _, r := range arr {
			wg.Add(1)
			go func(r *testReactor) {
				log.Println("Try stopping", r.name)
				failIfError(t, r.nt.Close(), "fail to close", r.name)
				r.nm.Term()
				wg.Done()
			}(r)
		}
	}
	wg.Wait()
}

func baseNetwork(t *testing.T) (m map[string][]*testReactor, ch chan context.Context) {
	m = make(map[string][]*testReactor)
	m[testValidator] = generateNetwork(testValidator, testNumValidator, t, module.RoleValidator)
	m[testSeed] = generateNetwork(testSeed, testNumSeed, t, module.RoleSeed)
	m[testCitizen] = generateNetwork(testCitizen, testNumCitizen, t)

	n := testNumValidator + testNumSeed + testNumCitizen
	na := m[testSeed][0].p2p.NetAddress()
	ch = make(chan context.Context, 2*n)
	for _, v := range m {
		for _, r := range v {
			r.ch = ch
			if r.p2p.NetAddress() != na {
				r.nm.SetTrustSeeds(string(na))
			}
		}
	}

	connMap, maxD, err := waitConnection(ch, defaultConnectionLimit, n, 10*DefaultSeedPeriod)
	t.Log(time.Now(), "max:", maxD, connMap)
	failIfError(t, err, "waitConnection", connMap)
	return m, ch
}

func assertEqualProtocolHandler(t *testing.T, expected, actual *protocolHandler) {
	assert.Equal(t, expected.protocol, actual.protocol)
	assert.Equal(t, sortProtocols(expected.getSubProtocols()),
		sortProtocols(actual.getSubProtocols()))
	assert.Equal(t, expected.getReactor(), actual.getReactor())
	assert.Equal(t, expected.getName(), actual.getName())
	assert.Equal(t, expected.getPriority(), expected.getPriority())
	assert.Equal(t, expected.getPolicy(), expected.getPolicy())
}

func Test_manager(t *testing.T) {
	w := walletFromGeneratedPrivateKey()
	logger := testLogger()
	nt := NewTransport(getAvailableLocalhostAddress(t), w, logger)
	chainLogger := logger.WithFields(log.Fields{log.FieldKeyCID: "1"})
	c := &dummyChain{nid: 1, metricCtx: context.Background(), logger: chainLogger}
	nm := NewManager(c, nt, "", module.RoleValidator).(*manager)
	type registerReactorParam struct {
		name     string
		pi       module.ProtocolInfo
		reactor  module.Reactor
		piList   []module.ProtocolInfo
		priority uint8
		policy   module.NotRegisteredProtocolPolicy
	}
	arg := registerReactorParam{
		name:     "dummyReactor",
		pi:       ProtoTestNetwork,
		reactor:  &dummyReactor{},
		piList:   testSubProtocols,
		priority: testProtoPriority,
		policy:   module.NotRegisteredProtocolPolicyClose,
	}
	expected := newProtocolHandler(
		nm,
		arg.pi,
		arg.piList,
		arg.reactor,
		arg.name,
		arg.priority,
		arg.policy,
		nm.logger)

	expectPanicFunc := func(priority uint8) assert.PanicTestFunc {
		return func() {
			_, _ = nm.RegisterReactor(
				arg.name,
				arg.pi,
				arg.reactor,
				arg.piList,
				priority,
				arg.policy)
		}
	}
	assert.Panics(t, expectPanicFunc(0))
	assert.Panics(t, expectPanicFunc(DefaultSendQueueMaxPriority+1))

	actual, err := nm.RegisterReactor(
		arg.name,
		arg.pi,
		arg.reactor,
		arg.piList,
		arg.priority,
		arg.policy)
	assert.NoError(t, err)
	assertEqualProtocolHandler(t, expected, actual.(*protocolHandler))

	assertRegistered := func(expected module.ProtocolHandler) {
		ph, ok := nm.getProtocolHandler(ProtoTestNetwork)
		assert.True(t, ok)
		assert.Equal(t, expected, ph)
	}
	assertRegistered(actual)

	r2 := &dummyReactor{}
	actual, err = nm.RegisterReactor(
		arg.name,
		arg.pi,
		r2,
		arg.piList,
		arg.priority,
		arg.policy)
	assert.NoError(t, err)
	assertRegistered(actual)

	argFuncs := []func(arg *registerReactorParam){
		func(arg *registerReactorParam) {
			arg.name = arg.name + "test"
		},
		func(arg *registerReactorParam) {
			arg.priority = arg.priority + 1
		},
		func(arg *registerReactorParam) {
			arg.policy = arg.policy + 1
		},
		func(arg *registerReactorParam) {
			arg.piList = arg.piList[1:]
		},
		func(arg *registerReactorParam) {
			arg.piList = append(arg.piList, arg.piList[1])[1:]
		},
	}
	for _, argFunc := range argFuncs {
		copiedArg := arg
		argFunc(&copiedArg)
		_, err = nm.RegisterReactor(
			copiedArg.name,
			copiedArg.pi,
			copiedArg.reactor,
			copiedArg.piList,
			copiedArg.priority,
			copiedArg.policy)
		assert.Error(t, err)
		assertRegistered(actual)
	}

	err = nm.UnregisterReactor(arg.reactor)
	assert.NoError(t, err)
	_, ok := nm.getProtocolHandler(ProtoTestNetwork)
	assert.False(t, ok)

	err = nm.UnregisterReactor(arg.reactor)
	assert.Error(t, err)
}

func Test_network_basic(t *testing.T) {
	m, ch := baseNetwork(t)

	t.Log(time.Now(), "Messaging")

	msg := m[testValidator][0].Broadcast("Test1")
	n := testNumValidator - 1 + testNumSeed + testNumCitizen
	err := wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1")

	msg, n = m[testValidator][0].BroadcastNeighbor("Test2")
	err = wait(ch, ProtoTestNetworkNeighbor, msg, n, time.Second)
	assert.NoError(t, err, "BroadcastNeighbor", "Test2")

	msg = m[testValidator][0].Multicast("Test3")
	n = testNumValidator - 1
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second)
	assert.NoError(t, err, "Multicast", "Test3")

	msg = m[testSeed][0].Multicast("Test4")
	n = testNumValidator
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second)
	assert.NoError(t, err, "Multicast", "Test4")

	msg = m[testCitizen][0].Multicast("Test5")
	n = testNumValidator + 1 + DefaultUnclesLimit
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second+DefaultAlternateSendPeriod)
	assert.NoError(t, err, "Multicast", "Test5")

	tr := m[testSeed][0]
	for _, r := range m[testSeed] {
		p := m[testCitizen][0].p2p.getPeer(r.nt.PeerID(), true)
		if p != nil {
			tr = r
			break
		}
	}
	respCh := make(chan string, 1)
	tr.responseFunc = func(r *testReactor, rm *testNetworkRequest, id module.PeerID) error {
		resp := r.Response(rm.Message, id)
		respCh <- resp
		return nil
	}
	msg = m[testCitizen][0].Request("Test6", tr.nt.PeerID())

	msg, err = timeout(respCh, time.Second)
	assert.NoError(t, err, "timeout", "responseFunc")

	err = wait(ch, ProtoTestNetworkResponse, msg, 1, time.Second, m[testCitizen][0].name)
	assert.NoError(t, err, "Response", "Test6")

	time.Sleep(2 * DefaultAlternateSendPeriod)

	listenerClose(t, m)
	t.Log(time.Now(), "Finish")
}

func Test_network_allowedPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	m := make(map[string][]*testReactor)
	m[testAllowed] = generateNetwork(testAllowed, testNumAllowedPeer, t, module.RoleValidator)
	m[testNotAllowed] = generateNetwork(testNotAllowed, testNumNotAllowedPeer, t, module.RoleValidator)
	allowed := make([]module.PeerID, 0)
	notAllowed := make([]module.PeerID, 0)

	for _, r := range m[testAllowed] {
		allowed = append(allowed, r.nt.PeerID())
	}
	for _, r := range m[testNotAllowed] {
		notAllowed = append(notAllowed, r.nt.PeerID())
	}
	for _, r := range m[testAllowed] {
		r.nm.SetRole(1, module.RoleNormal, allowed...)
	}

	ch := make(chan context.Context, testNumAllowedPeer+testNumNotAllowedPeer)
	for _, v := range m {
		for _, r := range v {
			r.ch = ch
		}
	}

	sr := m[testAllowed][0]
	dailByMap(t, m, sr.p2p.NetAddress(), 100*time.Millisecond)

	n := testNumAllowedPeer
	connMap, maxD, err := waitConnection(ch, map[PeerRoleFlag][]int{
		p2pRoleRoot: []int{0, 0, 0, 0, 0, testNumAllowedPeer - 1},
	}, n, 10*DefaultSeedPeriod)
	t.Log(time.Now(), "max:", maxD, connMap)
	failIfError(t, err, "waitConnection", connMap)

	go func() {
		for _, r := range m[testAllowed] {
			dailByList(t, m[testNotAllowed], r.p2p.NetAddress(), 0)
		}
	}()
	evtMap, err := waitEvent(ch, n, 2*time.Second, p2pEventNotAllowed, notAllowed...)
	t.Log(time.Now(), "Before", evtMap)
	assert.NoError(t, err, "waitEvent", evtMap)

	t.Log(time.Now(), "Messaging")
	msg := m[testAllowed][0].Broadcast("Test1")
	n = testNumAllowedPeer - 1
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1")

	remove := allowed[testNumAllowedPeer-1]
	go func() {
		for _, r := range m[testAllowed] {
			r.nm.RemoveRole(module.RoleNormal, remove)
		}
	}()
	evtMap, err = waitEvent(ch, n-1, 2*time.Second, p2pEventNotAllowed, remove)
	t.Log(time.Now(), "After", evtMap)
	assert.NoError(t, err, "waitEvent2", evtMap)

	msg = m[testAllowed][0].Broadcast("Test2")
	n = testNumAllowedPeer - 2
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test2")

	listenerClose(t, m)
	t.Log(time.Now(), "Finish")
}

func Test_network_trustSeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	m, ch := baseNetwork(t)

	trustSeeds := make([]string, 0)
	for _, r := range m[testCitizen] {
		trustSeeds = append(trustSeeds, string(r.p2p.NetAddress()))
	}
	strTrustSeeds := strings.Join(trustSeeds, ",")
	fmt.Println("strTrustSeeds: ", strTrustSeeds)

	m[testChild] = generateNetwork(testChild, testNumChild, t)
	ch1 := make(chan context.Context, testNumChild)
	for _, r := range m[testChild] {
		r.ch = ch1
		r.nm.SetTrustSeeds(strTrustSeeds)
	}

	connMap, maxD, err := waitConnection(ch1, defaultConnectionLimit, testNumChild, 10*DefaultSeedPeriod)
	t.Log(time.Now(), "max:", maxD, connMap)
	failIfError(t, err, "waitConnection", connMap)

	t.Log(time.Now(), "Messaging")

	msg := m[testValidator][0].Broadcast("Test1")
	n := testNumValidator - 1 + testNumSeed + testNumCitizen
	err = wait(ch, ProtoTestNetworkBroadcast, msg, n, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1")
	err = wait(ch1, ProtoTestNetworkBroadcast, msg, testNumChild, time.Second)
	assert.NoError(t, err, "Broadcast", "Test1 child")

	msg = m[testValidator][0].Multicast("Test2")
	err = wait(ch, ProtoTestNetworkMulticast, msg,
		testNumValidator-1, time.Second)
	assert.NoError(t, err, "Multicast", "Test2")

	msg = m[testCitizen][0].Multicast("Test3")
	n = testNumValidator + 1 + DefaultUnclesLimit
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second)
	assert.NoError(t, err, "Multicast", "Test3")

	msg = m[testChild][0].Multicast("Test4")
	n = testNumValidator + 2 + DefaultUnclesLimit
	err = wait(ch, ProtoTestNetworkMulticast, msg, n, time.Second+DefaultAlternateSendPeriod)
	assert.NoError(t, err, "Multicast", "Test4")

	time.Sleep(5 * time.Second)

	listenerClose(t, m)
	t.Log(time.Now(), "Finish")

}
