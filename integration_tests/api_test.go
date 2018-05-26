package integration_tests

import (
	"fmt"
	"github.com/donutloop/httpcache/internal/xhttp"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

var client *http.Client

func TestMain(m *testing.M) {

	proxy := xhttp.NewProxy(100)
	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	proxyServer := httptest.NewServer(proxy)

	transport := &http.Transport{
		Proxy: SetProxyURL(proxyServer.URL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client = &http.Client{
		Transport: transport,
	}

	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func SetProxyURL(proxy string) func(req *http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy address %q: %v", proxy, err)
		}
		return proxyURL, nil
	}
}

func TestProxy(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"count": 10}`))
		return
	}
	server := httptest.NewServer(http.HandlerFunc(handler))

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code is bad (%v)", resp.StatusCode)
	}
}

func BenchmarkProxy(b *testing.B) {
	servers := make([]*httptest.Server, 0)
	for i := 0; i < 10; i++ {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": "` + generateData(256) + `"}`))
			return
		}
		server := httptest.NewServer(http.HandlerFunc(handler))
		servers = append(servers, server)
	}

	b.N = 10

	for n := 0; n < b.N; n++ {
		req, err := http.NewRequest(http.MethodGet, servers[rand.Intn(9)].URL, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			b.Logf("status code is bad (%v)", resp.StatusCode)
		}
	}
}

func generateData(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}