package handler

import (
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"github.com/donutloop/httpcache/internal/roundtripper"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

func NewProxy(cache *cache.LRUCache, logger func(v ...interface{})) *Proxy {
	return &Proxy{
		client: &http.Client{
			Transport: &roundtripper.LoggedTransport{
				Transport: &roundtripper.CacheTransport{
					Transport: http.DefaultTransport,
					Cache:    cache,
				},
				Logger: logger,
			}},
		logger: logger,
	}
}

type Proxy struct {
	client *http.Client
	logger func(v ...interface{})
}

func (p *Proxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	req.RequestURI = ""
	proxyResponse, err := p.client.Do(req)
	if err != nil {
		p.logger(err.Error())
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