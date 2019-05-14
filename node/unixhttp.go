package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	BaseUnixDomainSockHttpEndpoint = "http://localhost"
)

type UnixDomainSockHttpServer struct {
	srv      http.Server
	l        net.Listener
	e        *echo.Echo
	sockPath string
}

func NewUnixDomainSockHttpServer(sockPath string, e *echo.Echo) *UnixDomainSockHttpServer {
	if e == nil {
		e = echo.New()
	}
	s := &UnixDomainSockHttpServer{
		e:        e,
		sockPath: sockPath,
	}
	s.srv.Handler = s.e
	s.srv.ErrorLog = s.e.StdLogger
	return s
}

func (s *UnixDomainSockHttpServer) Start() error {
	if err := os.RemoveAll(s.sockPath); err != nil {
		return err
	}
	l, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return err
	}
	s.l = l
	if err := s.srv.Serve(s.l); err != nil {
		return err
	}
	return nil
}

func (s *UnixDomainSockHttpServer) Stop() error {
	ctx, cf := context.WithTimeout(context.Background(), 5*time.Second)
	defer cf()
	return s.srv.Shutdown(ctx)
}

type UnixDomainSockHttpClient struct {
	hc       *http.Client
	sockPath string
}

func NewUnixDomainSockHttpClient(sockPath string) *UnixDomainSockHttpClient {
	c := &UnixDomainSockHttpClient{
		sockPath: sockPath,
	}
	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (conn net.Conn, e error) {
				return net.Dial("unix", c.sockPath)
			},
		},
	}
	c.hc = hc
	return c
}

func (c *UnixDomainSockHttpClient) _do(req *http.Request) (resp *http.Response, err error) {
	resp, err = c.hc.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(resp.Status)
		return
	}
	return
}

func (c *UnixDomainSockHttpClient) Do(method, reqUrl string, reqPtr, respPtr interface{}) (resp *http.Response, err error) {
	var reqB io.Reader
	if reqPtr != nil {
		b, mErr := json.Marshal(reqPtr)
		if mErr != nil {
			err = mErr
			return
		}
		reqB = bytes.NewBuffer(b)
	}
	log.Println(reqUrl)
	req, err := http.NewRequest(method, BaseUnixDomainSockHttpEndpoint+reqUrl, reqB)
	if err != nil {
		return
	}

	//if reqB != nil {
	//	log.Println("Using json header")
	req.Header.Set("Content-Type", "application/json")
	//} else {
	//	log.Println("Using text header")
	//	req.Header.Set("Accept","*/*")
	//}

	resp, err = c._do(req)
	if err != nil {
		return
	}

	if respPtr != nil {
		defer resp.Body.Close()
		switch ptr := respPtr.(type) {
		case *string:
			var b []byte
			b, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("ioutil.ReadAll:",err)
				return
			}
			*ptr = string(b)
		default:
			if err = json.NewDecoder(resp.Body).Decode(ptr); err != nil {
				log.Println("json.NewDecoder:",err)
				return
			}
		}

	}
	return
}

type StreamCallbackFunc func(respPtr interface{}) error

func (c *UnixDomainSockHttpClient) Stream(reqUrl string, reqPtr, respPtr interface{},
	respFunc StreamCallbackFunc, cancelCh <-chan bool, reqParams ...*url.Values) (resp *http.Response, err error) {
	var reqB io.Reader
	if reqPtr != nil {
		b, mErr := json.Marshal(reqPtr)
		if mErr != nil {
			err = mErr
			return
		}
		reqB = bytes.NewBuffer(b)

	}
	req, err := http.NewRequest(http.MethodGet, BaseUnixDomainSockHttpEndpoint+UrlWithParams(reqUrl, reqParams...), reqB)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c._do(req)
	if err != nil {
		return
	}
	if respFunc != nil {
		ch := make(chan interface{})
		dec := json.NewDecoder(resp.Body)
		defer resp.Body.Close()

		go func() {
			for {
				if err := dec.Decode(respPtr); err != nil {
					ch <- err
					return
				}
				ch <- respPtr
			}
		}()

		for {
			select {
			case <-cancelCh:
				return
			case v := <-ch:
				if de, ok := v.(error); ok {
					err = de
					return
				}
				if err = respFunc(v); err != nil {
					return
				}
			}
		}
	}
	return
}

func (c *UnixDomainSockHttpClient) Get(reqUrl string, ptr interface{}, reqParams ...*url.Values) (resp *http.Response, err error) {
	return c.Do(http.MethodGet, UrlWithParams(reqUrl, reqParams...), nil, ptr)
}
func (c *UnixDomainSockHttpClient) Post(reqUrl string) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, reqUrl, nil, nil)
}
func (c *UnixDomainSockHttpClient) PostWithJson(reqUrl string, ptr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, reqUrl, ptr, nil)
}

func (c *UnixDomainSockHttpClient) PostWithReader(reqUrl string, ptr interface{}, fieldname string, r io.Reader) (resp *http.Response, err error) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	if err = MultipartCopy(mw, fieldname, r); err != nil {
		return
	}
	if err = MultipartJson(mw, "json", ptr); err != nil {
		return
	}
	if err = mw.Close(); err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, BaseUnixDomainSockHttpEndpoint+reqUrl, buf)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err = c._do(req)
	if err != nil {
		return
	}
	return
}

func (c *UnixDomainSockHttpClient) PostWithFile(reqUrl string, ptr interface{}, fieldname, filename string) (resp *http.Response, err error) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	if err = MultipartFile(mw, fieldname, filename); err != nil {
		return
	}
	if err = MultipartJson(mw, "json", ptr); err != nil {
		return
	}
	if err = mw.Close(); err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, BaseUnixDomainSockHttpEndpoint+reqUrl, buf)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err = c._do(req)
	if err != nil {
		return
	}
	return
}
func (c *UnixDomainSockHttpClient) Delete(reqUrl string) (resp *http.Response, err error) {
	return c.Do(http.MethodDelete, reqUrl, nil, nil)
}

func UrlWithParams(reqUrl string, reqParams ...*url.Values) string {
	reqUrlWithParams := reqUrl
	if len(reqParams) > 0 {
		reqUrlWithParams += "?"
		for i, p := range reqParams {
			if i != 0 {
				reqUrlWithParams += "&"
			}
			reqUrlWithParams += p.Encode()
		}
	}
	return reqUrlWithParams
}

func MultipartCopy(mw *multipart.Writer, fieldname string, r io.Reader) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="blob"`, fieldname))
	h.Set("Content-Type", "application/zip")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err = io.Copy(pw, r); err != nil {
		return err
	}
	return nil
}

func MultipartFile(mw *multipart.Writer, fieldname, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	pw, err := mw.CreateFormFile(fieldname, path.Base(filename))
	if err != nil {
		return err
	}
	if _, err = io.Copy(pw, f); err != nil {
		return err
	}
	return nil
}
func MultipartJson(mw *multipart.Writer, fieldname string, v interface{}) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="blob"`, fieldname))
	h.Set("Content-Type", "application/json")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(pw).Encode(v); err != nil {
		return err
	}
	return nil
}
