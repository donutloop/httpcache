package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func NewPanic(next http.Handler, loggerFunc func(v ...interface{})) *Panic {
	return  &Panic{
		Next: next,
		loggerFunc: loggerFunc,
	}
}

// Panic recovers from API panics and logs encountered panics
type Panic struct {
	Next http.Handler
	loggerFunc func(v ...interface{})
}

// It recovers from panics of all next handlers and logs them
func (h *Panic) ServeHTTP(r http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			h.loggerFunc("begin: recovered from panic")
			h.loggerFunc(fmt.Sprintf("unkown value of recover (%v)", r))
			h.loggerFunc(fmt.Sprintf("url %v", req.URL.String()))
			h.loggerFunc(fmt.Sprintf("method %v", req.Method))
			h.loggerFunc(fmt.Sprintf("remote address %v", req.RemoteAddr))
			h.loggerFunc(fmt.Sprintf("stack strace of cause \n %v", string(debug.Stack())))
			h.loggerFunc("end: recovered from panic")
		}
	}()
	h.Next.ServeHTTP(r, req)
}
