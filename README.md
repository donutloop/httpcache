# httpcache

[![Build Status](https://travis-ci.org/donutloop/httpcache.svg?branch=master)](https://travis-ci.org/donutloop/httpcache)
[![Coverage Status](https://coveralls.io/repos/github/donutloop/httpcache/badge.svg)](https://coveralls.io/github/donutloop/httpcache)

An HTTP server that proxies all requests to other HTTP servers and this servers caches all incoming responses objects 

## Backend Requirements

* [golang](https://golang.org/) - The Go Programming Language
* [docker](https://www.docker.com/) - Build, Manage and Secure Your Apps Anywhere. Your Way.

## Prepare GO development environment

Follow [install guide](https://golang.org/doc/install) to install golang.

## Build without docker

```bash
mkdir -p $GOPATH/src/github.com/donutloop/ && cd $GOPATH/src/github.com/donutloop/

git clone git@github.com:donutloop/httpcache.git

cd httpcache

go build ./cmd/httpcache
```

## Build with docker

```bash
mkdir -p $GOPATH/src/github.com/donutloop/ && cd $GOPATH/src/github.com/donutloop/

git clone git@github.com:donutloop/httpcache.git

docker build .
```

## Usage 

```bash 
USAGE
  httpcache [flags]

FLAGS
  -cap 100          capacity of cache
  -cert server.crt  TLS certificate
  -expire 5         the items in the cache expire after or expire never
  -http :80         serve HTTP on this address (optional)
  -key server.key   TLS key
  -rbcl 524288000   response size limit
  -tls              serve TLS on this address (optional)
```