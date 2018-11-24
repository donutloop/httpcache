package tests

import (
	"encoding/json"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"github.com/donutloop/httpcache/internal/handler"
	"github.com/donutloop/httpcache/internal/middleware"
	"github.com/donutloop/httpcache/internal/size"
	"github.com/donutloop/httpcache/internal/xhttp"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"
)

var client *http.Client
var c *cache.LRUCache

func TestMain(m *testing.M) {
	c = cache.NewLRUCache(100, 0)
	proxy := handler.NewProxy(c, log.Println, 500*size.MB)
	stats := handler.NewStats(c, log.Println)

	mux := http.NewServeMux()
	mux.Handle("/stats", stats)
	mux.Handle("/", proxy)

	stack := middleware.NewPanic(mux, log.Println)

	proxyServer := httptest.NewServer(stack)

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

func TestProxyHandler(t *testing.T) {
	defer c.Reset()

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

	if c.Length() != 1 {
		t.Fatalf("cache length is bad, got=%d", c.Length())
	}
}

func TestStatsHandler(t *testing.T) {
	defer c.Reset()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"count": 10}`))
		return
	}

	server := httptest.NewServer(http.HandlerFunc(testHandler))

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(req.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code is bad (%v)", resp.StatusCode)
	}

	if c.Length() != 1 {
		t.Fatalf("cache length is bad, got=%d", c.Length())
	}

	req, err = http.NewRequest(http.MethodGet, server.URL+"/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(req.URL)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code is bad (%v)", resp.StatusCode)
	}

	statsResponse := &handler.StatsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(statsResponse); err != nil {
		b, err := httputil.DumpResponse(resp, true)
		if err == nil {
			t.Log(string(b))
		}
		t.Fatalf("could not decode incoming response (%v)", err)
	}

	if statsResponse.Length != 1 {
		t.Fatalf("cache length is bad, got=%d", c.Length())
	}

	t.Log(fmt.Sprintf("%#v", statsResponse))
}

func TestProxyHandler_ResponseBodyContentLengthLimit(t *testing.T) {
	c1 := cache.NewLRUCache(100, 1*time.Second)
	{
		c1.OnEviction = func(key string) {
			c1.Delete(key)
		}
	}
	cl := 1 * size.KB
	t.Log("size: ", cl)

	go func() {
		logger := log.New(os.Stderr, "", log.LstdFlags)

		proxy := handler.NewProxy(c1, logger.Println, cl)
		mux := http.NewServeMux()
		mux.Handle("/", proxy)

		listener, err := net.Listen("tcp", "localhost:4528")
		if err != nil {
			logger.Fatal(err)
		}

		xserver := xhttp.Server{
			Server:   &http.Server{Addr: "localhost:4528", Handler: proxy},
			Logger:   logger,
			Listener: listener,
		}
		if err := xserver.Start(); err != nil {
			xserver.Stop()
		}
	}()

	<-time.After(1 * time.Second)

	transport := &http.Transport{
		Proxy: SetProxyURL("http://localhost:4528"),
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

	client := &http.Client{
		Transport: transport,
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := make([]byte, 2*size.KB, 2*size.KB)
		w.Write(data)
		return
	}

	server := httptest.NewServer(http.HandlerFunc(testHandler))

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(req.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status code is bad (%v)", resp.StatusCode)
	}

	<-time.After(2 * time.Second)

	if c1.Length() != 0 {
		t.Fatalf("cache length is bad, got=%d", c1.Length())
	}
}

func TestProxyHandler_GC(t *testing.T) {
	c1 := cache.NewLRUCache(100, 1*time.Second)
	{
		c1.OnEviction = func(key string) {
			c1.Delete(key)
		}
	}

	go func() {
		logger := log.New(os.Stderr, "", log.LstdFlags)

		proxy := handler.NewProxy(c1, logger.Println, 3*size.MB)
		mux := http.NewServeMux()
		mux.Handle("/", proxy)

		listener, err := net.Listen("tcp", "localhost:4568")
		if err != nil {
			logger.Fatal(err)
		}

		xserver := xhttp.Server{
			Server:   &http.Server{Addr: "localhost:4568", Handler: proxy},
			Logger:   logger,
			Listener: listener,
		}
		if err := xserver.Start(); err != nil {
			xserver.Stop()
		}
	}()

	<-time.After(1 * time.Second)

	transport := &http.Transport{
		Proxy: SetProxyURL("http://localhost:4568"),
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

	client := &http.Client{
		Transport: transport,
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"count": 10}`))
		return
	}

	server := httptest.NewServer(http.HandlerFunc(testHandler))

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(req.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code is bad (%v)", resp.StatusCode)
	}

	<-time.After(2 * time.Second)

	if c1.Length() != 0 {
		t.Fatalf("cache length is bad, got=%d", c1.Length())
	}
}

func TestProxyHttpServer(t *testing.T) {

	c1 := cache.NewLRUCache(100, 0)
	go func() {
		logger := log.New(os.Stderr, "", log.LstdFlags)

		proxy := handler.NewProxy(c1, logger.Println, 5*size.MB)
		mux := http.NewServeMux()
		mux.Handle("/", proxy)

		listener, err := net.Listen("tcp", "localhost:4567")
		if err != nil {
			logger.Fatal(err)
		}

		xserver := xhttp.Server{
			Server:   &http.Server{Addr: "localhost:4567", Handler: proxy},
			Logger:   logger,
			Listener: listener,
		}
		if err := xserver.Start(); err != nil {
			xserver.Stop()
		}
	}()

	<-time.After(1 * time.Second)

	transport := &http.Transport{
		Proxy: SetProxyURL("http://localhost:4567"),
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

	client := &http.Client{
		Transport: transport,
	}

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

	v := struct {
		Count int
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}

	if v.Count != 10 {
		t.Fatalf("count is bad, got=%d", v.Count)
	}

	if c1.Length() != 1 {
		t.Fatalf("cache length is bad, got=%d", c1.Length())
	}
}

func BenchmarkProxy(b *testing.B) {
	defer c.Reset()

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
