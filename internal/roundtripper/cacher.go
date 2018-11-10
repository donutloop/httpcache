package roundtripper

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"net/http"
	"net/http/httputil"
	"strings"
)

type CacheTransport struct {
	Cache     *cache.LRUCache
	Transport http.RoundTripper // underlying transport (or default if nil)
}

func (t *CacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {


	clonedRequest, err := makeHashFromRequest(req)
	if err != nil {
		return nil, err
	}

	cacheControlHeader := req.Header.Get("Cache-Control")
	if cacheControlHeader != "" {
		cacheControlHeaders := strings.Split(cacheControlHeader, ",")

		hasMustRevalidate := containsHeaderValue(cacheControlHeaders, "must-revalidate")
		hasNoCache := containsHeaderValue(cacheControlHeaders, "no-cache")
		hasNoStore := containsHeaderValue(cacheControlHeaders, "no-store")

		if hasMustRevalidate && hasNoCache {
			t.Cache.Delete(clonedRequest)
			proxyResponse, err := t.Transport.RoundTrip(req)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
			}
			cachedResponse := &cache.CachedResponse{Resp: proxyResponse}
			t.Cache.Set(clonedRequest, cachedResponse)
			return proxyResponse, nil
		} else if hasNoCache {
			proxyResponse, err := t.Transport.RoundTrip(req)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
			}
			cachedResponse := &cache.CachedResponse{Resp: proxyResponse}
			t.Cache.Set(clonedRequest, cachedResponse)
			return proxyResponse, nil
		} else if hasMustRevalidate {
			t.Cache.Delete(clonedRequest)
			proxyResponse, err := t.Transport.RoundTrip(req)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
			}
			cachedResponse := &cache.CachedResponse{Resp: proxyResponse}
			t.Cache.Set(clonedRequest, cachedResponse)
			return proxyResponse, nil
		} else if hasNoStore {
			t.Cache.Delete(clonedRequest)
			proxyResponse, err := t.Transport.RoundTrip(req)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("proxy couldn't forward request to destination server (%v)", err))
			}
			return proxyResponse, nil
		}
	}

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
func makeHashFromRequest(r *http.Request) (string, error) {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}

	d, err := httputil.DumpRequest(r, true)
	if err != nil {
		return "", err
	}

	hasher := md5.New()
	hasher.Write([]byte(d))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func containsHeaderValue(headerValues []string, value string) bool {
	for _, v := range headerValues {
		if v == value {
			return true
		}
	}
	return false
}
