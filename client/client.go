package client

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/sonnt85/goshellbox/lib"
	"github.com/sonnt85/goshellbox/params"
)

// Version WebShell Client current version
const Version = "2.0"

// UserAgent Request header[User-Agent]
var UserAgent = fmt.Sprintf("goshellbox-client/%s (%s; %s; %s)", Version, runtime.GOOS, runtime.GOARCH, runtime.Version())

// ShellBoxClient connect to ShellBoxServer
type ShellBoxClient struct {
	Client *http.Client
	Dialer *websocket.Dialer
}

// Init http client
func (c *ShellBoxClient) Init(https bool, crt, key, rootcrt string) {
	if https {
		tlsConfig := &tls.Config{}
		if crt != "" && key != "" && rootcrt != "" {
			cliCrt, err := tls.LoadX509KeyPair(crt, key)
			if err != nil {
				log.Fatalln("Load crt or key file failed:", err.Error())
			}
			tlsConfig.RootCAs = lib.ReadCertPool(rootcrt)
			tlsConfig.Certificates = []tls.Certificate{cliCrt}
		} else if crt != "" {
			tlsConfig.RootCAs = lib.ReadCertPool(crt)
		} else {
			tlsConfig.InsecureSkipVerify = true
		}
		c.Client = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
		c.Dialer = &websocket.Dialer{TLSClientConfig: tlsConfig}
	} else {
		c.Client = &http.Client{}
		c.Dialer = &websocket.Dialer{}
	}
}

// Run ShellBoxClient
// func (c *ShellBoxClient) Run(https bool, username, password, host, post, contentpath string) {
func (c *ShellBoxClient) Run(parma *params.Parameter) {
	path, err := LoginServer(parma.IsHttps, parma.Username, parma.Password, parma.URLNoScheme, c.GetJSON)
	if err != nil {
		log.Println("Login to Server failed:", err.Error())
		return
	}
	ConnectSocket(parma, UserAgent, path, c.GetWebsocket)
}

// GetRes http get request
func (c *ShellBoxClient) GetRes(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UserAgent)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// GetJSON http get request and parse JSON
func (c *ShellBoxClient) GetJSON(url string) (map[string]interface{}, error) {
	res, err := c.GetRes(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New("response status is " + strconv.Itoa(res.StatusCode))
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(bytes, &data)
	return data, err
}

// GetWebsocket get websocket connection
func (c *ShellBoxClient) GetWebsocket(url string, headers ...map[string]string) (*websocket.Conn, error) {
	h := make(http.Header)
	h["User-Agent"] = []string{UserAgent}
	if len(headers) > 0 {
		for _, header := range headers {
			for k, v := range header {
				h.Set(k, v)
			}
		}
	}
	skt, _, err := c.Dialer.Dial(url, h)
	return skt, err
}

func Run(parms *params.Parameter) {
	c := new(ShellBoxClient)
	c.Init(parms.IsHttps, parms.CrtFile, parms.KeyFile, parms.RootCrtFile)
	c.Run(parms)
}
