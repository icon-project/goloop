package network

import (
	"context"
	"log"

	"github.com/icon-project/goloop/module"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

func MethodRepository(nt module.NetworkTransport) *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository()
	//RegisterMethod(method string, h Handler, params, result interface{}) error
	mr.RegisterMethod("dial", jsonrpcWithContext(nt, jsonrpcHandleDial), nil, nil)
	mr.RegisterMethod("query", jsonrpcWithContext(nt, jsonrpcHandleSendQuery), nil, nil)
	mr.RegisterMethod("p2p", jsonrpcWithContext(nt, jsonrpcHandleP2P), nil, nil)
	mr.RegisterMethod("logger", jsonrpcWithContext(nt, jsonrpcHandleLogger), nil, nil)
	return mr
}

type jsonrpcHandlerFunc func(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error)

func (f jsonrpcHandlerFunc) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	return f(c, params)
}

type rpcContextKey string

var (
	rpcContextKeyParamMap  = rpcContextKey("param")
	rpcContextKeyTransport = rpcContextKey("transport")
	rpcContextKeyP2P       = rpcContextKey("p2p")
)

func jsonrpcWithContext(nt module.NetworkTransport, next jsonrpcHandlerFunc) jsonrpcHandlerFunc {
	t := nt.(*transport)
	p2pMap := t.pd.p2pMap
	return func(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
		m := make(map[string]interface{})
		if err := jsonrpc.Unmarshal(params, &m); err != nil {
			log.Println("jsonrpcWithChannel jsonrpc.Unmarshal", err)
		}
		ctx := context.WithValue(c, rpcContextKeyParamMap, m)
		//log.Println("jsonrpcWithChannel param", m)

		ctx = context.WithValue(ctx, rpcContextKeyTransport, t)
		if channel, ok := m["channel"]; ok {
			if p2p, ok := p2pMap[channel.(string)]; ok {
				ctx = context.WithValue(ctx, rpcContextKeyP2P, p2p)
			}
		}
		return next.ServeJSONRPC(ctx, params)
	}
}

func _getParamString(c context.Context, k string) (string, *jsonrpc.Error) {
	m, err := _getParamMap(c)
	if err != nil {
		return "", err
	}
	v, ok := m[k]
	if !ok {
		log.Println("_getParamString not exists", k)
		return "", jsonrpc.ErrInvalidParams()
	}
	s, ok := v.(string)
	if !ok || s == "" {
		log.Println("_getParamString invalid param value to string")
		return "", jsonrpc.ErrInvalidParams()
	}
	return s, nil
}

func _getParamStringArray(c context.Context, k string) ([]string, *jsonrpc.Error) {
	m, err := _getParamMap(c)
	if err != nil {
		return nil, err
	}
	v, ok := m[k]
	if !ok {
		log.Println("_getParamStringArray not exists", k)
		return nil, jsonrpc.ErrInvalidParams()
	}
	a, ok := v.([]interface{})
	if !ok {
		log.Printf("_getParamStringArray invalid param value to []interface{} from %#v", v)
		return nil, jsonrpc.ErrInvalidParams()
	}
	arr := make([]string, len(a))
	for i, e := range a {
		s, ok := e.(string)
		if !ok {
			log.Printf("_getParamStringArray invalid param value to string from %#v", e)
			return nil, jsonrpc.ErrInvalidParams()
		}
		arr[i] = s
	}
	return arr, nil
}

func _getParamMap(c context.Context) (map[string]interface{}, *jsonrpc.Error) {
	v := c.Value(rpcContextKeyParamMap)
	if v == nil {
		log.Println("_getParamMap not exists rpcContextKeyParamMap")
		return nil, jsonrpc.ErrInvalidParams()
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		log.Println("_getParamMap invalid context value to map[string]interface{}")
		return nil, jsonrpc.ErrInvalidParams()
	}
	return m, nil
}

func _getTransport(c context.Context) (*transport, *jsonrpc.Error) {
	v := c.Value(rpcContextKeyTransport)
	if v == nil {
		log.Println("_getTransport not exists rpcContextKeyTransport")
		return nil, jsonrpc.ErrInvalidParams()
	}
	t, ok := v.(*transport)
	if !ok {
		log.Println("_getTransport invalid context value to *transport")
		return nil, jsonrpc.ErrInternal()
	}
	return t, nil
}

func _getP2P(c context.Context) (*PeerToPeer, *jsonrpc.Error) {
	v := c.Value(rpcContextKeyP2P)
	if v == nil {
		log.Println("_getP2P not exists rpcContextKeyP2P")
		return nil, jsonrpc.ErrInvalidParams()
	}
	p2p, ok := v.(*PeerToPeer)
	if !ok {
		log.Println("_getP2P invalid context value to *PeerToPeer")
		return nil, jsonrpc.ErrInternal()
	}
	return p2p, nil
}
func jsonrpcHandleSendQuery(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	id, err := _getParamString(c, "id")
	if err != nil {
		return nil, err
	}
	p := p2p.getPeer(NewPeerIDFromString(id))
	p2p.sendQuery(p)
	return nil, nil
}
func jsonrpcHandleDial(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	addr, err := _getParamString(c, "addr")
	if err != nil {
		return nil, err
	}
	dErr := p2p.dial(NetAddress(addr))
	if dErr != nil {
		log.Println("jsonrpcHandleDial dial fail", dErr.Error())
		return nil, jsonrpc.ErrInternal()
	}
	return nil, nil
}
func jsonrpcHandleP2P(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	m["self"] = toMap(p2p.self)
	m["seeds"] = p2p.seeds.Map()
	m["roots"] = p2p.roots.Map()
	m["friends"] = toMapArray(p2p.friends)
	m["parent"] = toMap(p2p.parent)
	m["children"] = toMapArray(p2p.children)
	m["uncles"] = toMapArray(p2p.uncles)
	m["nephews"] = toMapArray(p2p.nephews)
	m["orphanages"] = toMapArray(p2p.orphanages)
	return m, nil
}

func jsonrpcHandleLogger(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	logger, err := _getParamString(c, "logger")
	if err != nil {
		return nil, err
	}
	excludes, err := _getParamStringArray(c, "excludes")
	if err != nil {
		return nil, err
	}
	switch logger {
	case "global":
		singletonLoggerExcludes = excludes[:]
	case "PeerToPeer":
		p2p.log.excludes = excludes[:]
		//NetworkManager
		//Membership
		//Transport
		//Listener
		//Dialer
		//PeerDispatcher
		//ChannelNegotiator
		//Authenticator
	default:
		// ignore
	}
	return excludes, nil
}

func toMapArray(s *PeerSet) []map[string]interface{} {
	rarr := make([]map[string]interface{}, s.Len())
	for i, v := range s.Array() {
		rarr[i] = toMap(v)
	}
	return rarr
}
func toMap(p *Peer) map[string]interface{} {
	m := make(map[string]interface{})
	if p != nil {
		m["id"] = p.id.String()
		m["addr"] = p.netAddress
		m["in"] = p.incomming
		m["channel"] = p.channel
		m["role"] = p.role
		m["conn"] = p.connType
		m["rtt"] = p.rtt.String()
	}
	return m
}
