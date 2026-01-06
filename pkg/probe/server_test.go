package probe_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"go-backend-example/pkg/probe"
)

func TestServer(t *testing.T) {
	rq := require.New(t)

	testCases := []struct {
		name          string
		listenAddress string
		endpoint      string
		statusCode    int
		appName       string
		appVersion    string
		body          []byte
	}{
		{
			name:          "Health handler",
			listenAddress: ":10001",
			endpoint:      "http://:10001/healthz",
			statusCode:    http.StatusOK,
			appName:       "app-1",
			appVersion:    "v0.0.1",
			body:          []byte(`{"name":"app-1","version":"v0.0.1"}`),
		},
		{
			name:          "Ready handler",
			listenAddress: ":10002",
			endpoint:      "http://:10002/ready",
			statusCode:    http.StatusOK,
			appName:       "app-2",
			appVersion:    "v0.0.2",
			body:          []byte(`{"name":"app-2","version":"v0.0.2"}`),
		},
		{
			name:          "Invalid endpoint",
			listenAddress: ":10003",
			endpoint:      "http://:10003/invalid",
			statusCode:    http.StatusNotFound,
			body:          []byte("404 page not found\n"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(*testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			probeServer := probe.NewServer(
				tc.listenAddress,
				probe.Options{
					Name:    tc.appName,
					Version: tc.appVersion,
				},
			)

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				return probeServer.Run(ctx)
			})

			// Wait for server to start.
			time.Sleep(time.Second)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tc.endpoint, http.NoBody)
			rq.NoError(err)

			resp, err := http.DefaultClient.Do(req)
			rq.NoError(err)

			defer resp.Body.Close()

			rq.Equal(tc.statusCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			rq.NoError(err)

			rq.Equal(tc.body, bodyBytes)

			cancel()

			rq.NoError(g.Wait())
		})
	}
}
