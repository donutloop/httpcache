package main

import (
	"flag"
	"github.com/donutloop/httpcache/internal/xhttp"
	"log"
	"net/http"
	"strconv"
)

func main() {
	var port int
	var capOfCache int64
	flag.IntVar(&port, "port", 8080, "server is listing on port")
	flag.Int64Var(&capOfCache, "cap", 100, "capacity of cache")
	flag.Parse()

	proxy := xhttp.NewProxy(capOfCache)
	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	if err := http.ListenAndServe(":"+strconv.Itoa(port), proxy); err != nil {
		log.Fatalf("couldn't start proxy server on port (%v)", port)
	}
}
