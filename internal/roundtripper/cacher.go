package roundtripper

import (
	"errors"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"net/http"
)

type CacheTransport struct {
	Cache     *cache.LRUCache
	Transport http.RoundTripper // underlying transport (or default if nil)
}

func (t *CacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedRequest := cloneRequest(req)
	cachedResponse, ok := t.Cache.Get(clonedRequest)
	if !ok {
		proxyResponse, err := t.Transport.RoundTrip(req)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
		}
		cachedResponse = &cache.CachedResponse{Resp: proxyResponse}
		t.Cache.Set(clonedRequest, cachedResponse)
		return cachedResponse.Resp, nil
	}
	return cachedResponse.Resp, nil
}

// CloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}
