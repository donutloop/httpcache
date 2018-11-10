package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/donutloop/httpcache/internal/xhttp"
	"log"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Lshortfile | log.Ltime)

	fs := flag.NewFlagSet("http-proxy", flag.ExitOnError)
	var (
		httpAddr = fs.String("http", ":80", "serve HTTP on this address (optional)")
		tlsAddr  = fs.String("tls", "", "serve TLS on this address (optional)")
		cert     = fs.String("cert", "server.crt", "TLS certificate")
		key      = fs.String("key", "server.key", "TLS key")
		cap      = fs.Int64("cap", 100, "capacity of cache")
	)
	fs.Usage = usageFor(fs, "httpcache [flags]")
	fs.Parse(os.Args[1:])

	proxy := xhttp.NewProxy(*cap)
	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	if *httpAddr != "" {
		server := &http.Server{Addr: *httpAddr, Handler: proxy}
		log.Printf("serving HTTP on %s", *httpAddr)
		if err := server.ListenAndServe(); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			server.Shutdown(ctx)
		}
	} else {
		log.Printf("not serving HTTP")
	}

	if *tlsAddr != "" {
		server := &http.Server{Addr: *tlsAddr, Handler: xhttp.Hsts(proxy)}
		log.Printf("serving TLS on %s", *tlsAddr)
		if err := server.ListenAndServeTLS(*cert, *key); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			server.Shutdown(ctx)
		}
	} else {
		log.Printf("not serving TLS")
	}
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stdout, "USAGE\n")
		fmt.Fprintf(os.Stdout, "  %s\n", short)
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "FLAGS\n")
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			def := f.DefValue
			if def == "" {
				def = "..."
			}
			fmt.Fprintf(tw, "  -%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		tw.Flush()
	}
}
