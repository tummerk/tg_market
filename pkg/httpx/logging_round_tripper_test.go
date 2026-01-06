package httpx_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"

	"go-backend-example/pkg/contextx"
	"go-backend-example/pkg/httpx"
	"go-backend-example/pkg/logx"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary //nolint:gochecknoglobals // skip

func TestLoggingRoundTripper(t *testing.T) {
	const testResponseBody = `{"key":"value","password":"qwerty"}`

	rq := require.New(t)
	testLogFieldMaxLen10 := 10

	testCases := []struct {
		name                string
		handlerFunc         http.HandlerFunc
		statusCode          int
		responseBody        string
		sensitiveDataMasker *httpx.SensitiveDataMaskerMock
		logFieldMaxLen      int
		check               func(req, resp string)
	}{
		{
			name: "Staus 200",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			check: func(req, resp string) {
				rq.Contains(req, "GET / HTTP/1.1")
				rq.Contains(resp, "HTTP/1.1 200 OK")
			},
			statusCode: http.StatusOK,
		},
		{
			name: "Status 404",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(testResponseBody))
			}),
			check: func(req, resp string) {
				rq.Contains(req, "GET / HTTP/1.1")
				rq.Contains(resp, "HTTP/1.1 404 Not Found")
				rq.Contains(resp, testResponseBody)
			},
			statusCode:   http.StatusNotFound,
			responseBody: testResponseBody,
		},
		{
			name: "Staus 200 (masked)",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(testResponseBody))
			}),
			check: func(req, resp string) {
				rq.Contains(req, "GET / HTTP/1.1")
				rq.Contains(resp, "HTTP/1.1 200 OK")
				rq.Contains(resp, `{"key":"value",<...>}`)
			},
			statusCode:   http.StatusOK,
			responseBody: testResponseBody,
			sensitiveDataMasker: &httpx.SensitiveDataMaskerMock{
				MaskFunc: func(input []byte) []byte {
					return regexp.MustCompile(`"password":".+?"`).ReplaceAll(input, []byte("<...>"))
				},
			},
		},
		{
			name: "Status 200 (with log field size limit)",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(testResponseBody))
			}),
			check: func(req, resp string) {
				rq.Equal("GET / HTTP", req)
				rq.Equal("HTTP/1.1 2", resp)
			},
			statusCode:     http.StatusOK,
			responseBody:   testResponseBody,
			logFieldMaxLen: testLogFieldMaxLen10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(*testing.T) {
			httpServer := httptest.NewServer(tc.handlerFunc)
			defer httpServer.Close()

			var buf bytes.Buffer

			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			ctx := contextx.WithLogger(context.Background(), logger)

			var opts []httpx.Option

			if tc.sensitiveDataMasker != nil {
				opts = append(opts, httpx.WithSensitiveDataMasker(tc.sensitiveDataMasker))
			}

			if tc.logFieldMaxLen != 0 {
				opts = append(opts, httpx.WithLogFieldMaxLen(tc.logFieldMaxLen))
			}

			client := &http.Client{
				Transport: httpx.NewLoggingRoundTripper(
					http.DefaultTransport,
					opts...,
				),
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpServer.URL, http.NoBody)
			rq.NoError(err)

			resp, err := client.Do(req)
			rq.NoError(err)

			defer resp.Body.Close()

			logLines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))

			rq.Equal(tc.statusCode, resp.StatusCode)
			rq.Len(logLines, 2)

			var request, response map[string]any

			rq.NoError(json.Unmarshal(logLines[0], &request))
			rq.NoError(json.Unmarshal(logLines[1], &response))

			if tc.check != nil {
				tc.check(
					request[logx.FieldRequestBody].(string),
					response[logx.FieldResponseBody].(string),
				)
			}

			_, ok := response[logx.FieldDurationMs].(float64)
			rq.True(ok)

			const xidLen = 20

			rq.Len(request[logx.FieldRequestID], xidLen)
			rq.Len(request[logx.FieldRequestID], xidLen)

			if tc.responseBody != "" {
				bodyBytes, err := io.ReadAll(resp.Body)
				rq.NoError(err)

				rq.Equal(tc.responseBody, string(bodyBytes))
			}
		})
	}
}
