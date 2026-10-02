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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

	HandleRequest func(method string, path string) TestResponse

	calls     int
	Responses []TestResponse

	ResponseCode int
	ResponseBody interface{}
}

func NewTestHarness(t *testing.T) (baseURL string, server *TestServer) {
	t.Helper()

	testServer := &TestServer{
		t: t,
	}
	testServer.HandleRequest = func(method, path string) TestResponse {
		return TestResponse{
			Code: testServer.ResponseCode,
			Body: testServer.ResponseBody,
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", testServer)
	svr := &http.Server{
		Handler: mux,
	}

	// Start a listener on a random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = svr.Serve(listener)
	}()

	// When the test is over, close the server
	t.Cleanup(func() {
		if err := svr.Close(); err != nil {
			t.Fatal(err)
		}
	})

	baseURL = fmt.Sprintf("http://localhost:%v", listener.Addr().(*net.TCPAddr).Port)

	return baseURL, testServer
}

func (t *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Method = r.Method
	t.Path = r.URL.Path

	resp := t.HandleRequest(r.Method, t.Path)

	t.Body = bytes.NewBuffer([]byte{})
	if _, err := io.Copy(t.Body, r.Body); err != nil {
		t.t.Fatalf("Copy: %v", err)
	}
	w.WriteHeader(resp.Code)
	switch body := resp.Body.(type) {
	case []byte:
		if _, err := w.Write(body); err != nil {
			t.t.Fatalf("Write: %v", err)
		}
	default:
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.t.Fatalf("Encode: %v", err)
		}
	}
}

func (t *TestServer) SetResponses(responses []TestResponse) {
	t.Responses = responses
	t.calls = 0
	t.HandleRequest = func(method, path string) TestResponse {
		if len(t.Responses) > 0 {
			next := min(t.calls, len(t.Responses)-1)
			t.calls += 1
			return t.Responses[next]
		}
		return TestResponse{}
	}
}

type expectedErrorTestCase struct {
	Name        string
	Config      string
	ExpectError *regexp.Regexp
}

// runExpectedErrorTests checks that known invalid configurations are
// rejected with the correct error message.
func runExpectedErrorTests(t *testing.T, testCases []expectedErrorTestCase) {
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV5ProviderFactories: testProviderFactories(t),
				Steps: []resource.TestStep{
					{Config: tt.Config, ExpectError: tt.ExpectError},
				},
			})
		})
	}
}
