package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type FakeTestHTTPServer struct {
	Sleep      time.Duration
	StatusCode int
	JSONBody   bool
}

func (s FakeTestHTTPServer) ServeHTTP(respW http.ResponseWriter, req *http.Request) {
	if s.Sleep != 0 {
		time.Sleep(s.Sleep)
	}

	if s.StatusCode != 0 {
		respW.WriteHeader(s.StatusCode)
	}

	if s.JSONBody {
		io.WriteString(respW, "{\"message\":\"json message field\"}\n")
	} else {
		io.WriteString(respW, "OK\n")
	}
}

func NewHTTPServer(t *testing.T, serverHandler FakeTestHTTPServer) (string, http.Client, func()) {
	ts := httptest.NewServer(serverHandler)
	return ts.URL, *ts.Client(), func() {
		ts.Close()
	}
}
