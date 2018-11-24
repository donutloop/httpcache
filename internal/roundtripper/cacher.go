package roundtripper

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/donutloop/httpcache/internal/cache"
	"net/http"
	"net/http/httputil"
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
	cachedResponse, ok := t.Cache.Get(clonedRequest)
	if !ok {
		proxyResponse, err := t.Transport.RoundTrip(req)
		if err != nil {
			return nil, err
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
