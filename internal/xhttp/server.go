package xhttp

import (
	"errors"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

func NewProxy(capacity int64, errorLogger func(v ...interface{})) *Proxy {
	return &Proxy{
		cache:  cache.NewLRUCache(capacity),
		client: &http.Client{},
		ErrorLogger: errorLogger,
	}
}

type Proxy struct {
	cache  *cache.LRUCache
	client *http.Client
	ErrorLogger func(v ...interface{})
}

func (p *Proxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	proxyResponse, err := p.Do(req)
	if err != nil {
		p.ErrorLogger(err.Error())
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
		p.ErrorLogger(fmt.Sprintf("proxy couldn't read body of response (%v)", err))
		requestDumped, responseDumped, err := dump(req, proxyResponse)
		if err == nil {
			p.ErrorLogger(fmt.Sprintf("request: %#v", requestDumped))
			p.ErrorLogger(fmt.Sprintf("response: %#v", responseDumped))
		}
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(proxyResponse.StatusCode)
	resp.Write(body)
}

func (p *Proxy) Do(req *http.Request) (*http.Response, error) {
	clonedRequest := CloneRequest(req)
	cachedResponse, ok := p.cache.Get(clonedRequest)
	if !ok {
		req.RequestURI = ""
		proxyResponse, err := p.client.Do(req)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
		}
		cachedResponse = &cache.CachedResponse{Resp: proxyResponse}
		p.cache.Set(clonedRequest, cachedResponse)
		return cachedResponse.Resp, nil
	}
	return cachedResponse.Resp, nil
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

func Hsts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// CloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func CloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	r2.RequestURI = ""
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}
