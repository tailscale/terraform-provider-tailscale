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

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

type TestServer struct {
	t *testing.T

	Method string
	Path   string
	Body   *bytes.Buffer

	ResponseCode int
	ResponseBody interface{}
}

func NewTestHarness(t *testing.T) (*tsclient.Client, *TestServer) {
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
	client := &tsclient.Client{
		BaseURL: parsedBaseURL,
		APIKey:  "not-a-real-key",
		Tailnet: "example.com",
	}

	return client, testServer
}

func (t *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Method = r.Method
	t.Path = r.URL.Path

	t.Body = bytes.NewBuffer([]byte{})
	_, err := io.Copy(t.Body, r.Body)
	assert.NoError(t.t, err)

	w.WriteHeader(t.ResponseCode)
	switch body := t.ResponseBody.(type) {
	case []byte:
		_, err := w.Write(body)
		assert.NoError(t.t, err)
	default:
		assert.NoError(t.t, json.NewEncoder(w).Encode(body))
	}
}
