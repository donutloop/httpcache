package handler

import (
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"github.com/donutloop/httpcache/internal/roundtripper"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

func NewProxy(cache *cache.LRUCache, logger func(v ...interface{}), contentLength int64, ping *Ping, stats *Stats) *Proxy {
	return &Proxy{
		client: &http.Client{
			Transport: &roundtripper.LoggedTransport{
				Transport: &roundtripper.CacheTransport{
					Transport: &roundtripper.ResponseBodyLimitRoundTripper{
						Transport: http.DefaultTransport,
						Limit:     contentLength,
					},
					Cache: cache,
				},
				Logger: logger,
			}},
		logger: logger,
		ping:   ping,
		stats:  stats,
	}
}

type Proxy struct {
	client *http.Client
	logger func(v ...interface{})
	ping   *Ping
	stats  *Stats
}

func (p *Proxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	if req.URL.Path == "/ping" {
		p.ping.ServeHTTP(resp, req)
		return
	}

	if req.URL.Path == "/stats" {
		p.ping.ServeHTTP(resp, req)
		return
	}

	req.RequestURI = ""
	if req.Method == http.MethodConnect {
		p.ProxyHTTPS(resp, req)
		return
	}

	proxyResponse, err := p.client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), roundtripper.ResponseIsToLarge.Error()) {
			resp.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	for k, vv := range proxyResponse.Header {
		for _, v := range vv {
			resp.Header().Add(k, v)
		}
	}

	body, err := ioutil.ReadAll(proxyResponse.Body)
	if err != nil {
		p.logger(fmt.Sprintf("proxy couldn't read body of response (%v)", err))
		requestDumped, responseDumped, err := dump(req, proxyResponse)
		if err == nil {
			p.logger(fmt.Sprintf("request: %#v", requestDumped))
			p.logger(fmt.Sprintf("response: %#v", responseDumped))
		}
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(proxyResponse.StatusCode)
	resp.Write(body)
}

func (p *Proxy) ProxyHTTPS(rw http.ResponseWriter, req *http.Request) {
	hij, ok := rw.(http.Hijacker)
	if !ok {
		p.logger("proxy https error: http server does not support hijacker")
		return
	}

	clientConn, _, err := hij.Hijack()
	if err != nil {
		p.logger("proxy https error: %v", err)
		return
	}

	proxyConn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		p.logger("proxy https error: %v", err)
		return
	}

	_, err = clientConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	if err != nil {
		p.logger("proxy https error: %v", err)
		return
	}

	go func() {
		io.Copy(clientConn, proxyConn)
		clientConn.Close()
		proxyConn.Close()
	}()

	io.Copy(proxyConn, clientConn)
	proxyConn.Close()
	clientConn.Close()
}

type requestDump []byte

type responseDump []byte

func dump(request *http.Request, response *http.Response) (requestDump, responseDump, error) {
	dumpedResponse, err := httputil.DumpResponse(response, true)
	if err != nil {
		return nil, nil, err
	}
	dumpedRequest, err := httputil.DumpRequest(request, true)
	if err != nil {
		return nil, nil, err
	}
	return dumpedRequest, dumpedResponse, nil
}
