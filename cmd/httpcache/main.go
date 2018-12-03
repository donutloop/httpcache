package main

import (
	"flag"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"github.com/donutloop/httpcache/internal/handler"
	"github.com/donutloop/httpcache/internal/middleware"
	"github.com/donutloop/httpcache/internal/size"
	"github.com/donutloop/httpcache/internal/xhttp"
	"log"
	"net"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Lshortfile | log.Ltime)

	fs := flag.NewFlagSet("http-proxy", flag.ExitOnError)
	var (
		httpAddr                       = fs.String("http", ":80", "serve HTTP on this address (optional)")
		tlsAddr                        = fs.String("tls", "", "serve TLS on this address (optional)")
		cert                           = fs.String("cert", "server.crt", "TLS certificate")
		key                            = fs.String("key", "server.key", "TLS key")
		cap                            = fs.Int64("cap", 100, "capacity of cache")
		responseBodyContentLenghtLimit = fs.Int64("rbcl", 500*size.MB, "response size limit")
		expire                         = fs.Int64("expire", 5, "the items in the cache expire after or expire never")
	)
	fs.Usage = usageFor(fs, "httpcache [flags]")
	fs.Parse(os.Args[1:])

	logger := log.New(os.Stderr, "", log.LstdFlags)

	logger.Print(
		"\n",
		fmt.Sprintf("http addr: %v \n", *httpAddr),
		fmt.Sprintf("tls addr: %v \n", *tlsAddr),
		fmt.Sprintf("cap: %v \n", *cap),
		fmt.Sprintf("responseBodyContentLenghtLimit: %v \n", *responseBodyContentLenghtLimit),
		fmt.Sprintf("expire: %v \n", *expire),
	)

	e := time.Duration(*expire) * (time.Hour * 24)
	c := cache.NewLRUCache(*cap, e)
	{
		c.OnEviction = func(key string) {
			logger.Println(fmt.Sprintf("cache item is older then %v dayes (key: %s)", e, key))
			c.Delete(key)
		}
	}

	proxy := handler.NewProxy(c, logger.Println, *responseBodyContentLenghtLimit)
	stats := handler.NewStats(c, logger.Println)

	mux := http.NewServeMux()
	mux.Handle("/stats", stats)
	mux.Handle("/", proxy)

	stack := middleware.NewPanic(mux, logger.Println)

	if *httpAddr != "" {
		listener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			log.Fatal(err)
		}

		xserver := xhttp.Server{
			Server:          &http.Server{Addr: *httpAddr, Handler: stack},
			Logger:          logger,
			Listener:        listener,
			ShutdownTimeout: 3 * time.Second,
		}
		if err := xserver.Start(); err != nil {
			xserver.Stop()
		}
	} else {
		logger.Printf("not serving HTTP")
	}

	if *tlsAddr != "" {

		listener, err := net.Listen("tcp", *tlsAddr)
		if err != nil {
			logger.Fatal(err)
		}

		xserver := xhttp.Server{
			Server:          &http.Server{Addr: *tlsAddr, Handler: stack},
			Logger:          logger,
			Listener:        listener,
			ShutdownTimeout: 3 * time.Second,
		}
		if err := xserver.StartTLS(*cert, *key); err != nil {
			xserver.Stop()
		}
	} else {
		logger.Printf("not serving TLS")
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
