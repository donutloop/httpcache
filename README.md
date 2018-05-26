# httpcache

An HTTP server that proxies all requests to other HTTP servers and this servers caches all incoming responses objects 

# Installation 

```bash
go get github.com/donutloop/httpcache
```

# Usage 

```bash 
USAGE
  httpcache [flags]

FLAGS
  -cap 100          capacity of cache
  -cert server.crt  TLS certificate
  -http :80         serve HTTP on this address (optional)
  -key server.key   TLS key
  -tls              serve TLS on this address (optional)
```