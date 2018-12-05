package handler

import (
	"net/http"
)

func NewPing(logger func(v ...interface{})) *Ping {
	return &Ping{
		logger: logger,
	}
}

type Ping struct {
	logger func(v ...interface{})
}

func (s *Ping) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	s.logger("pinged cache")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("ok"))
}
