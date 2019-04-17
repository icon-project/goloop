package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"strconv"

	"github.com/icon-project/goloop/chain"
	"github.com/spf13/cobra"
)

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

func (c *UnixDomainSockHttpClient) Do(method, url string, reqPtr, respPtr interface{}) (resp *http.Response, err error) {
	var reqB io.Reader
	if reqPtr != nil {
		b, mErr := json.Marshal(reqPtr)
		if mErr != nil {
			err = mErr
			return
		}
		reqB = bytes.NewBuffer(b)

	}
	req, err := http.NewRequest(method, "http://localhost"+url, reqB)
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
		if err = json.NewDecoder(resp.Body).Decode(respPtr); err != nil {
			return
		}
	}
	return
}

func (c *UnixDomainSockHttpClient) Get(url string, ptr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodGet, url, nil, ptr)
}
func (c *UnixDomainSockHttpClient) Post(url string) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, url, nil, nil)
}
func (c *UnixDomainSockHttpClient) PostWithJson(url string, ptr interface{}) (resp *http.Response, err error) {
	return c.Do(http.MethodPost, url, ptr, nil)
}

func (c *UnixDomainSockHttpClient) PostWithReader(url string, ptr interface{}, fieldname string, r io.Reader) (resp *http.Response, err error) {
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
	req, err := http.NewRequest(http.MethodPost, "http://localhost"+url, buf)
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

func (c *UnixDomainSockHttpClient) PostWithFile(url string, ptr interface{}, fieldname, filename string) (resp *http.Response, err error) {
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
	req, err := http.NewRequest(http.MethodPost, "http://localhost"+url, buf)
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
func (c *UnixDomainSockHttpClient) Delete(url string) (resp *http.Response, err error) {
	return c.Do(http.MethodDelete, url, nil, nil)
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

func JsonIntend(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", nil
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, b, "", "  ")
	if err != nil {
		return "", nil
	}
	return string(buf.Bytes()), nil
}

var (
	genesisZip, genesisPath string
	joinChainParam          JoinChainParam
)

func NewChainCmd(cfg *GoLoopConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chain",
		Short: "Manage chains",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.DisableFlagsInUseLine = true
	cmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List chains",
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			l := make([]*ChainView, 0)
			resp, err := hc.Get(UrlChain, &l)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(l)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	})
	joinCmd := &cobra.Command{
		Use:   "join NID",
		Short: "Join chain",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			var err error
			if joinChainParam.NID, err = strconv.Atoi(args[0]); err != nil {
				fmt.Println("cannot parse NID", err)
				return
			}

			var resp *http.Response

			if len(genesisZip) > 0 {
				resp, err = hc.PostWithFile(UrlChain, &joinChainParam, "genesisZip", genesisZip)
			} else if len(genesisPath) > 0 {
				buf := bytes.NewBuffer(nil)
				err = chain.WriteGenesisStorageFromPath(buf, genesisPath)
				if err != nil {
					fmt.Println(err)
					return
				}
				resp, err = hc.PostWithReader(UrlChain, &joinChainParam, "genesisZip", buf)
			} else {
				fmt.Println("There is no genesis")
				return
			}

			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	joinCmd.Flags().StringVar(&genesisZip, "genesis", "", "Genesis storage path")
	joinCmd.Flags().StringVar(&genesisPath, "genesis_template", "", "Genesis template directory or file")
	joinCmd.Flags().StringVar(&joinChainParam.SeedAddr, "seed", "", "Ip-port of Seed")
	joinCmd.Flags().UintVar(&joinChainParam.Role, "role", 2, "[0:None, 1:Seed, 2:Validator, 3:Both]")

	leaveCmd := &cobra.Command{
		Use:                   "leave NID",
		Short:                 "Leave chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			resp, err := hc.Delete(UrlChain + "/" + args[0])
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	cmd.AddCommand(joinCmd, leaveCmd)
	inspectCmd := &cobra.Command{
		Use:                   "inspect NID",
		Short:                 "Inspect chain",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			v := &ChainInspectView{}
			resp, err := hc.Get(UrlChain+"/"+args[0], v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	}
	startCmd := &cobra.Command{
		Use:                   "start NID",
		Short:                 "Chain start",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/start")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	stopCmd := &cobra.Command{
		Use:                   "stop NID",
		Short:                 "Chain stop",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			resp, err := hc.Post(UrlChain + "/" + args[0] + "/stop")
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(b))
		},
	}
	cmd.AddCommand(inspectCmd, startCmd, stopCmd)
	return cmd
}

func NewSystemCmd(cfg *GoLoopConfig) *cobra.Command {
	c := &cobra.Command{
		Use:                   "system",
		Short:                 "System info",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hc := NewUnixDomainSockHttpClient(cfg.CliSocket)
			v := &SystemView{}
			resp, err := hc.Get(UrlSystem, v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			s, err := JsonIntend(v)
			if err != nil {
				fmt.Println(err, resp)
				return
			}
			fmt.Println(s)
		},
	}
	return c
}
