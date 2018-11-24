package roundtripper

import (
	"errors"
	"net/http"
)

var ResponseIsToLarge = errors.New("response body is to large for the cache")

type ResponseBodyLimitRoundTripper struct {
	Limit     int64
	Transport http.RoundTripper // underlying transport (or default if nil)
}

func (t *ResponseBodyLimitRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if response.ContentLength > t.Limit {
		return nil, ResponseIsToLarge
	}

	return response, nil
}
