// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tailscale.com/client/tailscale/v2"
)

type TestResponse struct {
	Code int
	Body interface{}
}

type TestServer struct {
	t *testing.T

	Method string
	Path   string
	Body   *bytes.Buffer

	calls     int
	Responses []TestResponse

	ResponseCode int
	ResponseBody interface{}
}

func NewTestHarness(t *testing.T) (*tailscale.Client, *TestServer) {
	t.Helper()

	testServer := &TestServer{
		t: t,
	}

	mux := http.NewServeMux()
	mux.Handle("/", testServer)
	svr := &http.Server{
		Handler: mux,
	}

	// Start a listener on a random port
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	go func() {
		_ = svr.Serve(listener)
	}()

	// When the test is over, close the server
	t.Cleanup(func() {
		assert.NoError(t, svr.Close())
	})

	baseURL := fmt.Sprintf("http://localhost:%v", listener.Addr().(*net.TCPAddr).Port)
	parsedBaseURL, err := url.Parse(baseURL)
	require.NoError(t, err)
	client := &tailscale.Client{
		BaseURL: parsedBaseURL,
		APIKey:  "not-a-real-key",
		Tailnet: "example.com",
	}

	return client, testServer
}

func (t *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Method = r.Method
	t.Path = r.URL.Path

	var resp TestResponse
	if len(t.Responses) > 0 {
		next := min(t.calls, len(t.Responses)-1)
		t.calls += 1
		resp = t.Responses[next]
	} else {
		resp = TestResponse{
			Code: t.ResponseCode,
			Body: t.ResponseBody,
		}
	}

	t.Body = bytes.NewBuffer([]byte{})
	_, err := io.Copy(t.Body, r.Body)
	assert.NoError(t.t, err)
	w.WriteHeader(resp.Code)
	switch body := resp.Body.(type) {
	case []byte:
		_, err := w.Write(body)
		assert.NoError(t.t, err)
	default:
		assert.NoError(t.t, json.NewEncoder(w).Encode(body))
	}
}
