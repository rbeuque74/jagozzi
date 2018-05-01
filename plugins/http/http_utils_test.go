package http

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

type HttpServerTimeout struct {
	Sleep      time.Duration
	StatusCode int
}

func (s HttpServerTimeout) ServeHTTP(respW http.ResponseWriter, req *http.Request) {
	if s.Sleep != 0 {
		time.Sleep(s.Sleep)
	}

	if s.StatusCode != 0 {
		respW.WriteHeader(s.StatusCode)
	}

	io.WriteString(respW, "OK\n")
}

func NewHTTPServer(t *testing.T, cfg HttpServerTimeout) func(context.Context) error {
	srv := http.Server{
		Addr:    ":8080",
		Handler: cfg,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}()
	return srv.Shutdown
}
