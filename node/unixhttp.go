package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/server"
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
		e.HTTPErrorHandler = server.HTTPErrorHandler
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

//socket path platform-specific length Mac&BSD:104, Linux:108
//when net.Dial return error as
//  (*net.OpError).Err.(*os.SyscallError).Err.(syscall.Errno) == syscall.EINVAL
//[TBD] symbolic link cannot resolved
func resolveSocketPath(sockPath string) string {
	wd, err := filepath.Abs(".")
	if err != nil {
		return sockPath
	}
	relPath, err := filepath.Rel(wd, sockPath)
	if err != nil {
		return sockPath
	}
	if len(relPath) > len(sockPath) {
		return sockPath
	}
	return relPath
}

func NewUnixDomainSockHttpClient(sockPath string) *UnixDomainSockHttpClient {
	c := &UnixDomainSockHttpClient{
		sockPath: sockPath,
	}
	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (conn net.Conn, e error) {
				sockPath := resolveSocketPath(c.sockPath)
				return net.Dial("unix", sockPath)
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
		err = NewRestError(resp)
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
	err = decodeResponseBody(resp, respPtr)
	return
}

func decodeResponseBody(resp *http.Response, respPtr interface{}) error {
	if respPtr != nil {
		defer resp.Body.Close()
		switch ptr := respPtr.(type) {
		case *string:
			var b []byte
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed read err=%+v", err)
			}
			*ptr = string(b)
		default:
			if err := json.NewDecoder(resp.Body).Decode(ptr); err != nil {
				return fmt.Errorf("failed json decode err=%+v", err)
			}
		}
	}
	return nil
}

func FileDownload(resp *http.Response) (b []byte, fileName string, err error) {
	hcd := resp.Header.Get(echo.HeaderContentDisposition)
	if hcd == "" {
		err = fmt.Errorf("not exists HeaderContentDisposition")
		return
	}
	s := strings.Split(hcd, ";")
	if len(s) != 2 {
		err = fmt.Errorf("invalid HeaderContentDisposition %s", hcd)
		return
	}
	//dispositionType := s[0]
	fileName = strings.TrimSpace(s[1])
	if strings.HasPrefix(fileName, "filename=") {
		fileName = fileName[len("filename="):]
		fileName = strings.Trim(fileName, "\"")
		defer resp.Body.Close()
		b, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed read err=%+v", err)
			return
		}
		return
	} else {
		err = fmt.Errorf("not exists filename")
		return
	}
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

func (c *UnixDomainSockHttpClient) Get(reqUrl string, respPtr interface{}, reqParams ...*url.Values) (resp *http.Response, err error) {
	return c.Do(http.MethodGet, UrlWithParams(reqUrl, reqParams...), nil, respPtr)
}
func (c *UnixDomainSockHttpClient) Post(reqUrl string, respPtr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, reqUrl, nil, respPtr)
}
func (c *UnixDomainSockHttpClient) PostWithJson(reqUrl string, reqPtr interface{}, respPtr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, reqUrl, reqPtr, respPtr)
}

func (c *UnixDomainSockHttpClient) PostWithReader(reqUrl string, reqPtr interface{}, fieldName string, r io.Reader, respPtr interface{}) (resp *http.Response, err error) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	if err = MultipartCopy(mw, fieldName, r); err != nil {
		return
	}
	if err = MultipartJson(mw, "json", reqPtr); err != nil {
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
	err = decodeResponseBody(resp, respPtr)
	return
}

func (c *UnixDomainSockHttpClient) PostWithFile(reqUrl string, reqPtr interface{}, fieldName, fileName string, respPtr interface{}) (resp *http.Response, err error) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	if err = MultipartFile(mw, fieldName, fileName); err != nil {
		return
	}
	if err = MultipartJson(mw, "json", reqPtr); err != nil {
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
	err = decodeResponseBody(resp, respPtr)
	return
}
func (c *UnixDomainSockHttpClient) Delete(reqUrl string, respPtr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodDelete, reqUrl, nil, respPtr)
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

func MultipartCopy(mw *multipart.Writer, fieldName string, r io.Reader) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="blob"`, fieldName))
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

func MultipartFile(mw *multipart.Writer, fieldName, fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	pw, err := mw.CreateFormFile(fieldName, path.Base(fileName))
	if err != nil {
		return err
	}
	if _, err = io.Copy(pw, f); err != nil {
		return err
	}
	return nil
}
func MultipartJson(mw *multipart.Writer, fieldName string, v interface{}) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"`, fieldName))
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

type RestError struct {
	status   int
	response string
	message  string
}

func (e *RestError) Error() string {
	return e.message
}

func (e *RestError) StatusCode() int {
	return e.status
}

func (e *RestError) Response() string {
	return e.response
}

func NewRestError(r *http.Response) error {
	var response string
	if rb, err := io.ReadAll(r.Body); err != nil {
		response = fmt.Sprintf("Fail to read body err=%+v", err)
	} else {
		response = string(rb)
	}
	return &RestError{
		status:   r.StatusCode,
		message:  "HTTP " + r.Status,
		response: response,
	}
}
