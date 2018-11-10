package roundtripper

import (
	"fmt"
	"net/http"
	"time"
)

// A LoggedTransport prints URLs and timings for each HTTP request.
type LoggedTransport struct {
	Logger    func(v ...interface{})
	Transport http.RoundTripper // underlying transport (or default if nil)
}

func (t *LoggedTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	start := time.Now()
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		t.Logger(fmt.Sprintf("HTTP %s %s: error: %s\n", req.Method, req.URL, err))
		return nil, err
	}

	t.Logger(fmt.Sprintf("HTTP %s %s %d [%s rtt]\n", req.Method, req.URL, resp.StatusCode, time.Since(start)))

	return resp, nil
}
