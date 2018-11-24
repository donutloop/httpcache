package handler

import (
	"encoding/json"
	"fmt"
	"github.com/donutloop/httpcache/internal/cache"
	"net/http"
	"time"
)

func NewStats(c *cache.LRUCache, logger func(v ...interface{})) *Stats {
	return &Stats{
		c:      c,
		logger: logger,
	}
}

type StatsResponse struct {
	Length   int64     `json:"length"`
	Size     int64     `json:"size"`
	Capacity int64     `json:"capacity"`
	Oldest   time.Time `json:"oldest"`
}

type Stats struct {
	c      *cache.LRUCache
	logger func(v ...interface{})
}

func (s *Stats) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	domainResp := s.Endpoint()

	v, err := json.Marshal(domainResp)
	if err != nil {
		s.logger(fmt.Sprintf("could not marshal response (%v)", err))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(v)
}

func (s *Stats) Endpoint() *StatsResponse {

	length, size, capacity, oldest := s.c.Stats()

	resp := &StatsResponse{
		Length:   length,
		Size:     size,
		Capacity: capacity,
		Oldest:   oldest,
	}

	return resp
}
