package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/icon-project/goloop/server/jsonrpc"
)

type JsonRpcClient struct {
	hc       *http.Client
	Endpoint string
	CustomHeader map[string]string
}

func NewJsonRpcClient(hc *http.Client, endpoint string) *JsonRpcClient {
	return &JsonRpcClient{hc: hc, Endpoint: endpoint}
}

func (c *JsonRpcClient) _do(req *http.Request) (resp *http.Response, err error) {
	resp, err = c.hc.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http-status(%s) is not StatusOK", resp.Status)
		return
	}
	return
}

func (c *JsonRpcClient) Do(method string, reqPtr, respPtr interface{}) (jrResp *jsonrpc.Response, err error) {
	jrReq := &jsonrpc.Request{
		ID: time.Now().UnixNano() / int64(time.Millisecond),
		Version: jsonrpc.Version,
		Method:  method,
	}
	if reqPtr != nil {
		b, mErr := json.Marshal(reqPtr)
		if mErr != nil {
			err = mErr
			return
		}
		jrReq.Params = json.RawMessage(b)
	}
	reqB, err := json.Marshal(jrReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(reqB))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range c.CustomHeader {
		req.Header.Set(k, v)
	}

	resp, err := c._do(req)
	if err != nil {
		if resp != nil {
			jrResp, _ = decodeResponseBody(resp, nil)
			err = fmt.Errorf("resp:%+v,err:%+v", jrResp, err)
		}
		return
	}
	jrResp, err = decodeResponseBody(resp, respPtr)
	return
}

func (c *JsonRpcClient) Raw(reqB []byte) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(reqB))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range c.CustomHeader {
		req.Header.Set(k, v)
	}

	return c._do(req)
}

func decodeResponseBody(resp *http.Response, respPtr interface{}) (jrResp *jsonrpc.Response, err error) {
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&jrResp); err != nil {
		return
	}
	if respPtr != nil {
		rb, mErr := json.Marshal(jrResp.Result)
		if mErr != nil {
			err = mErr
			return
		}
		err = json.Unmarshal(rb, respPtr)
		if err != nil {
			return
		}
	}
	return
}

